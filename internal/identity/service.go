package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

const sessionTokenPrefix = "ns_session_"

var slugSeparators = regexp.MustCompile(`[^a-z0-9]+`)

type RegistrationInput struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"displayName"`
	WorkspaceName string `json:"workspaceName"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Service struct {
	repository Repository
	hasher     PasswordHasher
	sessionTTL time.Duration
}

func NewService(
	repository Repository,
	hasher PasswordHasher,
	sessionTTL time.Duration,
) *Service {
	if sessionTTL <= 0 {
		sessionTTL = 30 * 24 * time.Hour
	}
	return &Service{
		repository: repository,
		hasher:     hasher,
		sessionTTL: sessionTTL,
	}
}

func (s *Service) Register(
	ctx context.Context,
	input RegistrationInput,
) (AuthResult, error) {
	email, displayName, workspaceName, err := normalizeRegistration(input)
	if err != nil {
		return AuthResult{}, err
	}
	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return AuthResult{}, err
	}
	now := time.Now().UTC()
	user := User{
		ID: uuid.New(), Email: email, DisplayName: displayName,
		CreatedAt: now, UpdatedAt: now,
	}
	workspace := Workspace{
		ID: uuid.New(), Name: workspaceName,
		Slug: workspaceSlug(workspaceName, user.ID), Role: RoleOwner,
		CreatedBy: user.ID, CreatedAt: now, UpdatedAt: now,
	}
	membership := Membership{
		WorkspaceID: workspace.ID, UserID: user.ID,
		Role: RoleOwner, CreatedAt: now,
	}
	session, token, err := newSession(user.ID, s.sessionTTL)
	if err != nil {
		return AuthResult{}, err
	}
	if err := s.repository.CreateRegistration(
		ctx,
		user,
		passwordHash,
		workspace,
		membership,
		session,
	); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{
		Account: Account{
			User: user, Workspaces: []Workspace{workspace},
			ExpiresAt: session.ExpiresAt,
		},
		Token: token,
	}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	user, encodedHash, err := s.repository.UserByEmail(ctx, email)
	if err != nil {
		if err == ErrUserNotFound {
			dummyPasswordCheck(input.Password)
			return AuthResult{}, ErrUnauthenticated
		}
		return AuthResult{}, err
	}
	valid, err := s.hasher.Verify(encodedHash, input.Password)
	if err != nil || !valid {
		return AuthResult{}, ErrUnauthenticated
	}
	session, token, err := newSession(user.ID, s.sessionTTL)
	if err != nil {
		return AuthResult{}, err
	}
	if err := s.repository.CreateSession(ctx, session); err != nil {
		return AuthResult{}, err
	}
	workspaces, err := s.repository.ListWorkspaces(ctx, user.ID)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{
		Account: Account{
			User: user, Workspaces: workspaces, ExpiresAt: session.ExpiresAt,
		},
		Token: token,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (Account, error) {
	tokenHash, err := sessionTokenHash(token)
	if err != nil {
		return Account{}, ErrUnauthenticated
	}
	session, user, err := s.repository.SessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if err == ErrUnauthenticated {
			return Account{}, err
		}
		return Account{}, err
	}
	workspaces, err := s.repository.ListWorkspaces(ctx, user.ID)
	if err != nil {
		return Account{}, err
	}
	return Account{
		User: user, Workspaces: workspaces, ExpiresAt: session.ExpiresAt,
	}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	tokenHash, err := sessionTokenHash(token)
	if err != nil {
		return nil
	}
	return s.repository.DeleteSession(ctx, tokenHash)
}

func (s *Service) CreateWorkspace(
	ctx context.Context,
	userID uuid.UUID,
	name string,
) (Workspace, error) {
	name = strings.TrimSpace(name)
	if !validDisplayText(name, 1, 100) {
		return Workspace{}, fmt.Errorf(
			"%w: workspace name must contain 1 to 100 printable characters",
			ErrInvalidInput,
		)
	}
	now := time.Now().UTC()
	workspace := Workspace{
		ID: uuid.New(), Name: name, Slug: workspaceSlug(name, uuid.New()),
		Role: RoleOwner, CreatedBy: userID, CreatedAt: now, UpdatedAt: now,
	}
	membership := Membership{
		WorkspaceID: workspace.ID, UserID: userID,
		Role: RoleOwner, CreatedAt: now,
	}
	if err := s.repository.CreateWorkspace(ctx, workspace, membership); err != nil {
		return Workspace{}, err
	}
	return workspace, nil
}

func (s *Service) ListWorkspaces(
	ctx context.Context,
	userID uuid.UUID,
) ([]Workspace, error) {
	return s.repository.ListWorkspaces(ctx, userID)
}

func (s *Service) SelectWorkspace(
	ctx context.Context,
	account Account,
	requested string,
) (Principal, error) {
	if len(account.Workspaces) == 0 {
		return Principal{}, ErrWorkspaceNotFound
	}
	if requested == "" {
		return Principal{Account: account, Workspace: account.Workspaces[0]}, nil
	}
	id, err := uuid.Parse(requested)
	if err != nil {
		return Principal{}, ErrWorkspaceNotFound
	}
	for _, workspace := range account.Workspaces {
		if workspace.ID == id {
			return Principal{Account: account, Workspace: workspace}, nil
		}
	}
	if _, err := s.repository.Membership(ctx, account.User.ID, id); err != nil {
		return Principal{}, ErrWorkspaceNotFound
	}
	return Principal{}, ErrWorkspaceNotFound
}

func normalizeRegistration(
	input RegistrationInput,
) (string, string, string, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	address, err := mail.ParseAddress(email)
	if err != nil || strings.ToLower(address.Address) != email ||
		len(email) > 254 {
		return "", "", "", fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}
	if len(input.Password) < 10 || len(input.Password) > 128 {
		return "", "", "", fmt.Errorf(
			"%w: password must contain 10 to 128 characters",
			ErrInvalidInput,
		)
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if !validDisplayText(displayName, 1, 80) {
		return "", "", "", fmt.Errorf(
			"%w: display name must contain 1 to 80 printable characters",
			ErrInvalidInput,
		)
	}
	workspaceName := strings.TrimSpace(input.WorkspaceName)
	if workspaceName == "" {
		workspaceName = displayName + " Workspace"
	}
	if !validDisplayText(workspaceName, 1, 100) {
		return "", "", "", fmt.Errorf(
			"%w: workspace name must contain 1 to 100 printable characters",
			ErrInvalidInput,
		)
	}
	return email, displayName, workspaceName, nil
}

func validDisplayText(value string, minimum int, maximum int) bool {
	if len(value) < minimum || len(value) > maximum {
		return false
	}
	return strings.IndexFunc(value, unicode.IsControl) < 0
}

func workspaceSlug(name string, entropy uuid.UUID) string {
	slug := strings.Trim(slugSeparators.ReplaceAllString(
		strings.ToLower(name),
		"-",
	), "-")
	if slug == "" {
		slug = "workspace"
	}
	if len(slug) > 48 {
		slug = strings.TrimRight(slug[:48], "-")
	}
	return slug + "-" + strings.ReplaceAll(entropy.String()[:8], "-", "")
}

func newSession(userID uuid.UUID, ttl time.Duration) (Session, string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return Session{}, "", fmt.Errorf("generate session token: %w", err)
	}
	token := sessionTokenPrefix + base64.RawURLEncoding.EncodeToString(randomBytes)
	tokenHash := sha256.Sum256([]byte(token))
	now := time.Now().UTC()
	return Session{
		ID: uuid.New(), UserID: userID, TokenHash: tokenHash[:],
		ExpiresAt: now.Add(ttl), CreatedAt: now, LastUsedAt: now,
	}, token, nil
}

func sessionTokenHash(token string) ([]byte, error) {
	if !strings.HasPrefix(token, sessionTokenPrefix) ||
		len(token) < len(sessionTokenPrefix)+32 {
		return nil, ErrUnauthenticated
	}
	hash := sha256.Sum256([]byte(token))
	return hash[:], nil
}

func dummyPasswordCheck(password string) {
	_ = argon2.IDKey(
		[]byte(password),
		[]byte("netscope-dummy!!"),
		3,
		64*1024,
		2,
		32,
	)
}
