package collaboration

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
)

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
