package reports

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/collaboration"
	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

type repositoryStub struct {
	comments []Comment
	links    []PublicLink
	events   []collaboration.AuditEvent
	name     string
}

func (r *repositoryStub) ListComments(
	_ context.Context,
	workspaceID uuid.UUID,
	runID uuid.UUID,
) ([]Comment, error) {
	result := make([]Comment, 0)
	for _, comment := range r.comments {
		if comment.WorkspaceID == workspaceID && comment.RunID == runID {
			result = append(result, comment)
		}
	}
	return result, nil
}

func (r *repositoryStub) Comment(
	_ context.Context,
	workspaceID uuid.UUID,
	runID uuid.UUID,
	commentID uuid.UUID,
) (Comment, error) {
	for _, comment := range r.comments {
		if comment.WorkspaceID == workspaceID && comment.RunID == runID &&
			comment.ID == commentID {
			return comment, nil
		}
	}
	return Comment{}, ErrCommentMissing
}

func (r *repositoryStub) CreateComment(
	_ context.Context,
	comment Comment,
	event collaboration.AuditEvent,
) error {
	r.comments = append(r.comments, comment)
	r.events = append(r.events, event)
	return nil
}

func (r *repositoryStub) DeleteComment(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	commentID uuid.UUID,
	event collaboration.AuditEvent,
) error {
	for index := range r.comments {
		if r.comments[index].ID == commentID {
			r.comments = append(r.comments[:index], r.comments[index+1:]...)
			r.events = append(r.events, event)
			return nil
		}
	}
	return ErrCommentMissing
}

func (r *repositoryStub) ListPublicLinks(
	_ context.Context,
	workspaceID uuid.UUID,
	runID uuid.UUID,
) ([]PublicLink, error) {
	result := make([]PublicLink, 0)
	for _, link := range r.links {
		if link.WorkspaceID == workspaceID && link.RunID == runID {
			result = append(result, link)
		}
	}
	return result, nil
}

func (r *repositoryStub) CreatePublicLink(
	_ context.Context,
	link PublicLink,
	event collaboration.AuditEvent,
) error {
	r.links = append(r.links, link)
	r.events = append(r.events, event)
	return nil
}

func (r *repositoryStub) RevokePublicLink(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	linkID uuid.UUID,
	revokedAt time.Time,
	event collaboration.AuditEvent,
) error {
	for index := range r.links {
		if r.links[index].ID == linkID && r.links[index].RevokedAt == nil {
			r.links[index].RevokedAt = &revokedAt
			r.events = append(r.events, event)
			return nil
		}
	}
	return ErrPublicLinkMissing
}

func (r *repositoryStub) ResolvePublicLink(
	_ context.Context,
	hash []byte,
	_ time.Time,
) (ResolvedPublicLink, error) {
	for _, link := range r.links {
		if bytes.Equal(link.TokenHash, hash) && link.RevokedAt == nil {
			return ResolvedPublicLink{
				PublicLink:    link,
				WorkspaceName: r.name,
			}, nil
		}
	}
	return ResolvedPublicLink{}, ErrPublicLinkMissing
}

type runRepositoryStub struct {
	run diagnostics.DiagnosticRun
}

func (r runRepositoryStub) Create(context.Context, diagnostics.DiagnosticRun) error {
	return nil
}

func (r runRepositoryStub) GetByID(
	_ context.Context,
	id uuid.UUID,
) (diagnostics.DiagnosticRun, error) {
	if r.run.ID != id {
		return diagnostics.DiagnosticRun{}, diagnostics.ErrRunNotFound
	}
	return r.run, nil
}

func (r runRepositoryStub) List(
	context.Context,
	diagnostics.ListFilter,
) (diagnostics.Page, error) {
	return diagnostics.Page{}, nil
}

func (r runRepositoryStub) UpdateStatus(
	context.Context,
	uuid.UUID,
	diagnostics.RunStatus,
) error {
	return nil
}

func (r runRepositoryStub) SaveResult(
	context.Context,
	diagnostics.CheckResult,
) error {
	return nil
}

func TestReportCollaborationLifecycle(t *testing.T) {
	t.Parallel()

	workspaceID := uuid.New()
	runID := uuid.New()
	userID := uuid.New()
	repository := &repositoryStub{name: "Acme Production"}
	service := testService(repository, workspaceID, runID)
	ctx := reportPrincipal(userID, workspaceID, identity.RoleOperator)

	comment, err := service.CreateComment(ctx, runID, "  Looks healthy.  ")
	if err != nil || comment.Body != "Looks healthy." {
		t.Fatalf("CreateComment() = %#v, %v", comment, err)
	}
	comments, err := service.ListComments(ctx, runID)
	if err != nil || len(comments) != 1 {
		t.Fatalf("ListComments() = %#v, %v", comments, err)
	}
	created, err := service.CreatePublicLink(ctx, runID, nil)
	if err != nil || created.Token == "" || len(created.TokenHash) != 32 {
		t.Fatalf("CreatePublicLink() = %#v, %v", created, err)
	}
	public, err := service.PublicReport(context.Background(), created.Token)
	if err != nil || public.Run.ID != runID ||
		public.WorkspaceName != "Acme Production" {
		t.Fatalf("PublicReport() = %#v, %v", public, err)
	}
	if err := service.DeleteComment(ctx, runID, comment.ID); err != nil {
		t.Fatalf("DeleteComment() error = %v", err)
	}
	if err := service.RevokePublicLink(ctx, runID, created.ID); err != nil {
		t.Fatalf("RevokePublicLink() error = %v", err)
	}
	if _, err := service.PublicReport(
		context.Background(),
		created.Token,
	); !errors.Is(err, ErrPublicLinkMissing) {
		t.Fatalf("PublicReport(revoked) error = %v", err)
	}
	if len(repository.events) != 4 {
		t.Fatalf("audit events = %#v", repository.events)
	}
}

func TestViewerCannotMutateReportCollaboration(t *testing.T) {
	t.Parallel()

	workspaceID := uuid.New()
	runID := uuid.New()
	service := testService(&repositoryStub{}, workspaceID, runID)
	ctx := reportPrincipal(
		uuid.New(),
		workspaceID,
		identity.RoleViewer,
	)
	if _, err := service.CreateComment(
		ctx,
		runID,
		"not allowed",
	); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("CreateComment(viewer) error = %v", err)
	}
	if _, err := service.CreatePublicLink(
		ctx,
		runID,
		nil,
	); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("CreatePublicLink(viewer) error = %v", err)
	}
}

func testService(
	repository Repository,
	workspaceID uuid.UUID,
	runID uuid.UUID,
) *Service {
	runService := diagnostics.NewService(
		runRepositoryStub{run: diagnostics.DiagnosticRun{
			ID: runID, WorkspaceID: workspaceID, TargetInput: "example.com",
			Status: diagnostics.RunCompleted, CreatedAt: time.Now(),
		}},
		nil,
		target.Policy{},
		time.Second,
		30*time.Second,
	)
	return NewService(repository, runService)
}

func reportPrincipal(
	userID uuid.UUID,
	workspaceID uuid.UUID,
	role identity.Role,
) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{
		Account: identity.Account{User: identity.User{
			ID: userID, Email: "operator@example.com", DisplayName: "Operator",
		}},
		Workspace: identity.Workspace{ID: workspaceID, Role: role},
	})
}
