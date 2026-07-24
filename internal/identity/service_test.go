package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type identityRepositoryStub struct {
	users      map[string]storedUser
	sessions   map[string]Session
	workspaces map[uuid.UUID][]Workspace
}

type storedUser struct {
	user User
	hash string
}

func newIdentityRepositoryStub() *identityRepositoryStub {
	return &identityRepositoryStub{
		users: make(map[string]storedUser), sessions: make(map[string]Session),
		workspaces: make(map[uuid.UUID][]Workspace),
	}
}

func (r *identityRepositoryStub) CreateRegistration(
	_ context.Context,
	user User,
	passwordHash string,
	workspace Workspace,
	_ Membership,
	session Session,
) error {
	if _, exists := r.users[user.Email]; exists {
		return ErrEmailExists
	}
	r.users[user.Email] = storedUser{user: user, hash: passwordHash}
	r.sessions[string(session.TokenHash)] = session
	r.workspaces[user.ID] = []Workspace{workspace}
	return nil
}

func (r *identityRepositoryStub) UserByEmail(
	_ context.Context,
	email string,
) (User, string, error) {
	stored, exists := r.users[email]
	if !exists {
		return User{}, "", ErrUserNotFound
	}
	return stored.user, stored.hash, nil
}

func (r *identityRepositoryStub) CreateSession(
	_ context.Context,
	session Session,
) error {
	r.sessions[string(session.TokenHash)] = session
	return nil
}

func (r *identityRepositoryStub) SessionByTokenHash(
	_ context.Context,
	tokenHash []byte,
) (Session, User, error) {
	session, exists := r.sessions[string(tokenHash)]
	if !exists || !session.ExpiresAt.After(time.Now()) {
		return Session{}, User{}, ErrUnauthenticated
	}
	for _, stored := range r.users {
		if stored.user.ID == session.UserID {
			return session, stored.user, nil
		}
	}
	return Session{}, User{}, ErrUnauthenticated
}

func (r *identityRepositoryStub) DeleteSession(
	_ context.Context,
	tokenHash []byte,
) error {
	delete(r.sessions, string(tokenHash))
	return nil
}

func (r *identityRepositoryStub) ListWorkspaces(
	_ context.Context,
	userID uuid.UUID,
) ([]Workspace, error) {
	return r.workspaces[userID], nil
}

func (r *identityRepositoryStub) CreateWorkspace(
	_ context.Context,
	workspace Workspace,
	membership Membership,
) error {
	r.workspaces[membership.UserID] = append(
		r.workspaces[membership.UserID],
		workspace,
	)
	return nil
}

func (r *identityRepositoryStub) Membership(
	_ context.Context,
	userID uuid.UUID,
	workspaceID uuid.UUID,
) (Membership, error) {
	for _, workspace := range r.workspaces[userID] {
		if workspace.ID == workspaceID {
			return Membership{
				UserID: userID, WorkspaceID: workspaceID, Role: workspace.Role,
			}, nil
		}
	}
	return Membership{}, ErrWorkspaceNotFound
}

func TestServiceRegistrationAndAuthentication(t *testing.T) {
	t.Parallel()

	repository := newIdentityRepositoryStub()
	service := NewService(
		repository,
		NewPasswordHasher(testPasswordParams()),
		time.Hour,
	)
	result, err := service.Register(context.Background(), RegistrationInput{
		Email: "  Owner@Example.com ", Password: "strong-password",
		DisplayName: "Ada Operator", WorkspaceName: "Acme Production",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.User.Email != "owner@example.com" ||
		len(result.Workspaces) != 1 ||
		result.Workspaces[0].Role != RoleOwner ||
		result.Token == "" {
		t.Fatalf("Register() = %#v", result)
	}

	account, err := service.Authenticate(context.Background(), result.Token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if account.User.ID != result.User.ID ||
		account.Workspaces[0].ID != result.Workspaces[0].ID {
		t.Fatalf("Authenticate() = %#v", account)
	}
}

func TestServiceLoginAndLogout(t *testing.T) {
	t.Parallel()

	repository := newIdentityRepositoryStub()
	service := NewService(
		repository,
		NewPasswordHasher(testPasswordParams()),
		time.Hour,
	)
	_, err := service.Register(context.Background(), RegistrationInput{
		Email: "operator@example.com", Password: "strong-password",
		DisplayName: "Operator",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := service.Login(context.Background(), LoginInput{
		Email: "operator@example.com", Password: "wrong-password",
	}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Login(wrong) error = %v", err)
	}
	result, err := service.Login(context.Background(), LoginInput{
		Email: "operator@example.com", Password: "strong-password",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := service.Logout(context.Background(), result.Token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := service.Authenticate(
		context.Background(),
		result.Token,
	); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Authenticate(after logout) error = %v", err)
	}
}

func TestServiceCreatesAndSelectsWorkspace(t *testing.T) {
	t.Parallel()

	repository := newIdentityRepositoryStub()
	service := NewService(
		repository,
		NewPasswordHasher(testPasswordParams()),
		time.Hour,
	)
	result, err := service.Register(context.Background(), RegistrationInput{
		Email: "owner@example.com", Password: "strong-password",
		DisplayName: "Owner",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	created, err := service.CreateWorkspace(
		context.Background(),
		result.User.ID,
		"Platform",
	)
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}
	account, err := service.Authenticate(context.Background(), result.Token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	principal, err := service.SelectWorkspace(
		context.Background(),
		account,
		created.ID.String(),
	)
	if err != nil || principal.Workspace.ID != created.ID {
		t.Fatalf("SelectWorkspace() = %#v, %v", principal, err)
	}
}
