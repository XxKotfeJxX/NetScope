package collaboration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
)

const defaultAPIKeyTTL = 90 * 24 * time.Hour
const maximumAPIKeyTTL = 365 * 24 * time.Hour

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListMembers(ctx context.Context) ([]Member, error) {
	principal, err := principal(ctx)
	if err != nil {
		return nil, err
	}
	return s.repository.ListMembers(ctx, principal.Workspace.ID)
}

func (s *Service) AddMember(
	ctx context.Context,
	input AddMemberInput,
) (Member, error) {
	principal, err := principal(ctx)
	if err != nil {
		return Member{}, err
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	address, parseErr := mail.ParseAddress(email)
	if parseErr != nil || strings.ToLower(address.Address) != email ||
		!validRole(input.Role) {
		return Member{}, fmt.Errorf("%w: valid email and role are required", ErrInvalidInput)
	}
	if err := canAssign(principal.Workspace.Role, input.Role); err != nil {
		return Member{}, err
	}
	event := newAuditEvent(
		principal,
		"workspace.member_added",
		nil,
		map[string]any{"email": email, "role": input.Role},
	)
	return s.repository.AddMember(
		ctx,
		principal.Workspace.ID,
		email,
		input.Role,
		event,
	)
}

func (s *Service) UpdateMemberRole(
	ctx context.Context,
	userID uuid.UUID,
	role identity.Role,
) (Member, error) {
	principal, err := principal(ctx)
	if err != nil {
		return Member{}, err
	}
	if !validRole(role) {
		return Member{}, fmt.Errorf("%w: invalid workspace role", ErrInvalidInput)
	}
	current, err := s.repository.Member(ctx, principal.Workspace.ID, userID)
	if err != nil {
		return Member{}, err
	}
	if err := canManage(principal, current); err != nil {
		return Member{}, err
	}
	if err := canAssign(principal.Workspace.Role, role); err != nil {
		return Member{}, err
	}
	event := newAuditEvent(
		principal,
		"workspace.member_role_updated",
		&userID,
		map[string]any{"from": current.Role, "to": role},
	)
	return s.repository.UpdateMemberRole(
		ctx,
		principal.Workspace.ID,
		userID,
		role,
		event,
	)
}

func (s *Service) RemoveMember(ctx context.Context, userID uuid.UUID) error {
	principal, err := principal(ctx)
	if err != nil {
		return err
	}
	current, err := s.repository.Member(ctx, principal.Workspace.ID, userID)
	if err != nil {
		return err
	}
	if err := canManage(principal, current); err != nil {
		return err
	}
	event := newAuditEvent(
		principal,
		"workspace.member_removed",
		&userID,
		map[string]any{"email": current.Email, "role": current.Role},
	)
	return s.repository.RemoveMember(
		ctx,
		principal.Workspace.ID,
		userID,
		event,
	)
}

func (s *Service) ListAudit(
	ctx context.Context,
	page int,
	pageSize int,
) (AuditPage, error) {
	principal, err := principal(ctx)
	if err != nil {
		return AuditPage{}, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	items, total, err := s.repository.ListAudit(
		ctx,
		principal.Workspace.ID,
		page,
		pageSize,
	)
	if err != nil {
		return AuditPage{}, err
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return AuditPage{
		Items: items, Page: page, PageSize: pageSize,
		TotalItems: total, TotalPages: totalPages,
	}, nil
}

func (s *Service) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	principal, err := principal(ctx)
	if err != nil {
		return nil, err
	}
	return s.repository.ListAPIKeys(ctx, principal.Workspace.ID)
}

func (s *Service) CreateAPIKey(
	ctx context.Context,
	input CreateAPIKeyInput,
) (CreatedAPIKey, error) {
	principal, err := principal(ctx)
	if err != nil {
		return CreatedAPIKey{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 100 || strings.IndexFunc(
		name,
		func(value rune) bool { return value < 32 || value == 127 },
	) >= 0 {
		return CreatedAPIKey{}, fmt.Errorf(
			"%w: API key name must contain 1 to 100 printable characters",
			ErrInvalidInput,
		)
	}
	if input.Role != identity.RoleOperator && input.Role != identity.RoleViewer {
		return CreatedAPIKey{}, fmt.Errorf(
			"%w: API key role must be operator or viewer",
			ErrInvalidInput,
		)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(defaultAPIKeyTTL)
	if input.ExpiresAt != nil {
		expiresAt = input.ExpiresAt.UTC()
	}
	if !expiresAt.After(now) || expiresAt.After(now.Add(maximumAPIKeyTTL)) {
		return CreatedAPIKey{}, fmt.Errorf(
			"%w: API key expiry must be within the next 365 days",
			ErrInvalidInput,
		)
	}
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return CreatedAPIKey{}, fmt.Errorf("generate API key: %w", err)
	}
	secret := base64.RawURLEncoding.EncodeToString(randomBytes)
	token := identity.APIKeyTokenPrefix + secret
	tokenHash := sha256.Sum256([]byte(token))
	key := APIKey{
		ID: uuid.New(), WorkspaceID: principal.Workspace.ID, Name: name,
		Prefix: identity.APIKeyTokenPrefix + secret[:10], Role: input.Role,
		CreatedBy: principal.User.ID, ExpiresAt: expiresAt,
		CreatedAt: now, TokenHash: tokenHash[:],
	}
	event := AuditEvent{
		ID: uuid.New(), WorkspaceID: principal.Workspace.ID,
		ActorUserID: principal.User.ID, Action: "workspace.api_key_created",
		ResourceType: "api_key", ResourceID: &key.ID,
		Metadata: map[string]any{
			"name": key.Name, "prefix": key.Prefix, "role": key.Role,
		},
		CreatedAt: now,
	}
	if err := s.repository.CreateAPIKey(ctx, key, event); err != nil {
		return CreatedAPIKey{}, err
	}
	return CreatedAPIKey{APIKey: key, Token: token}, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, keyID uuid.UUID) error {
	principal, err := principal(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	event := AuditEvent{
		ID: uuid.New(), WorkspaceID: principal.Workspace.ID,
		ActorUserID: principal.User.ID, Action: "workspace.api_key_revoked",
		ResourceType: "api_key", ResourceID: &keyID,
		Metadata: map[string]any{}, CreatedAt: now,
	}
	return s.repository.RevokeAPIKey(
		ctx,
		principal.Workspace.ID,
		keyID,
		now,
		event,
	)
}

func principal(ctx context.Context) (identity.Principal, error) {
	value, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return identity.Principal{}, identity.ErrUnauthenticated
	}
	if !identity.RoleAtLeast(value.Workspace.Role, identity.RoleAdmin) {
		return identity.Principal{}, identity.ErrForbidden
	}
	return value, nil
}

func validRole(role identity.Role) bool {
	return role == identity.RoleOwner || role == identity.RoleAdmin ||
		role == identity.RoleOperator || role == identity.RoleViewer
}

func canAssign(actor identity.Role, assigned identity.Role) error {
	if actor == identity.RoleOwner {
		return nil
	}
	if actor == identity.RoleAdmin &&
		(assigned == identity.RoleOperator || assigned == identity.RoleViewer) {
		return nil
	}
	return identity.ErrForbidden
}

func canManage(actor identity.Principal, member Member) error {
	if member.UserID == actor.User.ID {
		return identity.ErrForbidden
	}
	if actor.Workspace.Role == identity.RoleOwner {
		return nil
	}
	if actor.Workspace.Role == identity.RoleAdmin &&
		member.Role != identity.RoleOwner && member.Role != identity.RoleAdmin {
		return nil
	}
	return identity.ErrForbidden
}

func newAuditEvent(
	principal identity.Principal,
	action string,
	resourceID *uuid.UUID,
	metadata map[string]any,
) AuditEvent {
	return AuditEvent{
		ID: uuid.New(), WorkspaceID: principal.Workspace.ID,
		ActorUserID: principal.User.ID, Action: action,
		ResourceType: "workspace_member", ResourceID: resourceID,
		Metadata: metadata, CreatedAt: time.Now().UTC(),
	}
}
