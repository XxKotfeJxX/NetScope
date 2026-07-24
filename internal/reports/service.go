package reports

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/XxKotfeJxX/netscope/internal/collaboration"
	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
)

const publicTokenPrefix = "ns_share_"

type Service struct {
	repository Repository
	runs       *diagnostics.Service
}

func NewService(repository Repository, runs *diagnostics.Service) *Service {
	return &Service{repository: repository, runs: runs}
}

func (s *Service) ListComments(
	ctx context.Context,
	runID uuid.UUID,
) ([]Comment, error) {
	principal, err := requirePrincipal(ctx, identity.RoleViewer)
	if err != nil {
		return nil, err
	}
	if _, err := s.runs.Get(ctx, runID); err != nil {
		return nil, err
	}
	return s.repository.ListComments(ctx, principal.Workspace.ID, runID)
}

func (s *Service) CreateComment(
	ctx context.Context,
	runID uuid.UUID,
	body string,
) (Comment, error) {
	principal, err := requirePrincipal(ctx, identity.RoleOperator)
	if err != nil {
		return Comment{}, err
	}
	body = strings.TrimSpace(body)
	if len(body) < 1 || len(body) > 2000 ||
		strings.IndexFunc(body, invalidCommentRune) >= 0 {
		return Comment{}, fmt.Errorf(
			"%w: comment must contain 1 to 2000 printable characters",
			ErrInvalidInput,
		)
	}
	if _, err := s.runs.Get(ctx, runID); err != nil {
		return Comment{}, err
	}
	now := time.Now().UTC()
	comment := Comment{
		ID: uuid.New(), WorkspaceID: principal.Workspace.ID, RunID: runID,
		AuthorID: principal.User.ID, AuthorName: principal.User.DisplayName,
		AuthorEmail: principal.User.Email, Body: body,
		CreatedAt: now, UpdatedAt: now,
	}
	event := reportAudit(
		principal,
		"report.comment_created",
		"run_comment",
		comment.ID,
		map[string]any{"runId": runID},
		now,
	)
	if err := s.repository.CreateComment(ctx, comment, event); err != nil {
		return Comment{}, err
	}
	return comment, nil
}

func (s *Service) DeleteComment(
	ctx context.Context,
	runID uuid.UUID,
	commentID uuid.UUID,
) error {
	principal, err := requirePrincipal(ctx, identity.RoleOperator)
	if err != nil {
		return err
	}
	if _, err := s.runs.Get(ctx, runID); err != nil {
		return err
	}
	comment, err := s.repository.Comment(
		ctx,
		principal.Workspace.ID,
		runID,
		commentID,
	)
	if err != nil {
		return err
	}
	if comment.AuthorID != principal.User.ID &&
		!identity.RoleAtLeast(principal.Workspace.Role, identity.RoleAdmin) {
		return identity.ErrForbidden
	}
	now := time.Now().UTC()
	event := reportAudit(
		principal,
		"report.comment_deleted",
		"run_comment",
		commentID,
		map[string]any{"runId": runID},
		now,
	)
	return s.repository.DeleteComment(
		ctx,
		principal.Workspace.ID,
		runID,
		commentID,
		event,
	)
}

func (s *Service) ListPublicLinks(
	ctx context.Context,
	runID uuid.UUID,
) ([]PublicLink, error) {
	principal, err := requirePrincipal(ctx, identity.RoleViewer)
	if err != nil {
		return nil, err
	}
	if _, err := s.runs.Get(ctx, runID); err != nil {
		return nil, err
	}
	return s.repository.ListPublicLinks(ctx, principal.Workspace.ID, runID)
}

func (s *Service) CreatePublicLink(
	ctx context.Context,
	runID uuid.UUID,
	expiresAt *time.Time,
) (CreatedPublicLink, error) {
	principal, err := requirePrincipal(ctx, identity.RoleOperator)
	if err != nil {
		return CreatedPublicLink{}, err
	}
	if _, err := s.runs.Get(ctx, runID); err != nil {
		return CreatedPublicLink{}, err
	}
	now := time.Now().UTC()
	expiry := now.Add(30 * 24 * time.Hour)
	if expiresAt != nil {
		expiry = expiresAt.UTC()
	}
	if !expiry.After(now) || expiry.After(now.Add(365*24*time.Hour)) {
		return CreatedPublicLink{}, fmt.Errorf(
			"%w: public link expiry must be within the next 365 days",
			ErrInvalidInput,
		)
	}
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return CreatedPublicLink{}, fmt.Errorf("generate public report token: %w", err)
	}
	secret := base64.RawURLEncoding.EncodeToString(randomBytes)
	token := publicTokenPrefix + secret
	hash := sha256.Sum256([]byte(token))
	link := PublicLink{
		ID: uuid.New(), WorkspaceID: principal.Workspace.ID, RunID: runID,
		TokenPrefix: publicTokenPrefix + secret[:10], TokenHash: hash[:],
		CreatedBy: principal.User.ID, ExpiresAt: expiry, CreatedAt: now,
	}
	event := reportAudit(
		principal,
		"report.public_link_created",
		"public_report_link",
		link.ID,
		map[string]any{"runId": runID, "prefix": link.TokenPrefix},
		now,
	)
	if err := s.repository.CreatePublicLink(ctx, link, event); err != nil {
		return CreatedPublicLink{}, err
	}
	return CreatedPublicLink{PublicLink: link, Token: token}, nil
}

func (s *Service) RevokePublicLink(
	ctx context.Context,
	runID uuid.UUID,
	linkID uuid.UUID,
) error {
	principal, err := requirePrincipal(ctx, identity.RoleOperator)
	if err != nil {
		return err
	}
	if _, err := s.runs.Get(ctx, runID); err != nil {
		return err
	}
	now := time.Now().UTC()
	event := reportAudit(
		principal,
		"report.public_link_revoked",
		"public_report_link",
		linkID,
		map[string]any{"runId": runID},
		now,
	)
	return s.repository.RevokePublicLink(
		ctx,
		principal.Workspace.ID,
		runID,
		linkID,
		now,
		event,
	)
}

func (s *Service) PublicReport(
	ctx context.Context,
	token string,
) (PublicReport, error) {
	if !strings.HasPrefix(token, publicTokenPrefix) ||
		len(token) < len(publicTokenPrefix)+32 {
		return PublicReport{}, ErrPublicLinkMissing
	}
	hash := sha256.Sum256([]byte(token))
	now := time.Now().UTC()
	link, err := s.repository.ResolvePublicLink(ctx, hash[:], now)
	if err != nil {
		return PublicReport{}, err
	}
	run, err := s.runs.Get(identity.WithSystemAccess(ctx), link.RunID)
	if err != nil {
		return PublicReport{}, ErrPublicLinkMissing
	}
	return PublicReport{
		WorkspaceName: link.WorkspaceName, PublishedAt: link.CreatedAt,
		ExpiresAt: link.ExpiresAt, Run: run,
	}, nil
}

func requirePrincipal(
	ctx context.Context,
	role identity.Role,
) (identity.Principal, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return identity.Principal{}, identity.ErrUnauthenticated
	}
	if !identity.RoleAtLeast(principal.Workspace.Role, role) {
		return identity.Principal{}, identity.ErrForbidden
	}
	return principal, nil
}

func invalidCommentRune(value rune) bool {
	return unicode.IsControl(value) && value != '\n' && value != '\t'
}

func reportAudit(
	principal identity.Principal,
	action string,
	resourceType string,
	resourceID uuid.UUID,
	metadata map[string]any,
	createdAt time.Time,
) collaboration.AuditEvent {
	return collaboration.AuditEvent{
		ID: uuid.New(), WorkspaceID: principal.Workspace.ID,
		ActorUserID: principal.User.ID, Action: action,
		ResourceType: resourceType, ResourceID: &resourceID,
		Metadata: metadata, CreatedAt: createdAt,
	}
}
