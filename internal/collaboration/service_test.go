package collaboration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
)

type repositoryStub struct {
	members []Member
	events  []AuditEvent
}

func (r *repositoryStub) ListMembers(
	_ context.Context,
	_ uuid.UUID,
) ([]Member, error) {
	return append([]Member(nil), r.members...), nil
}

func (r *repositoryStub) Member(
	_ context.Context,
	_ uuid.UUID,
	userID uuid.UUID,
) (Member, error) {
	for _, member := range r.members {
		if member.UserID == userID {
			return member, nil
		}
	}
	return Member{}, ErrMemberMissing
}

func (r *repositoryStub) AddMember(
	_ context.Context,
	_ uuid.UUID,
	email string,
	role identity.Role,
	event AuditEvent,
) (Member, error) {
	for _, member := range r.members {
		if member.Email == email {
			return Member{}, ErrMemberExists
		}
	}
	member := Member{
		UserID: uuid.New(), Email: email, DisplayName: "New member",
		Role: role, JoinedAt: time.Now(),
	}
	event.ResourceID = &member.UserID
	r.members = append(r.members, member)
	r.events = append(r.events, event)
	return member, nil
}

func (r *repositoryStub) UpdateMemberRole(
	_ context.Context,
	_ uuid.UUID,
	userID uuid.UUID,
	role identity.Role,
	event AuditEvent,
) (Member, error) {
	for index := range r.members {
		if r.members[index].UserID == userID {
			r.members[index].Role = role
			r.events = append(r.events, event)
			return r.members[index], nil
		}
	}
	return Member{}, ErrMemberMissing
}

func (r *repositoryStub) RemoveMember(
	_ context.Context,
	_ uuid.UUID,
	userID uuid.UUID,
	event AuditEvent,
) error {
	for index := range r.members {
		if r.members[index].UserID == userID {
			r.members = append(r.members[:index], r.members[index+1:]...)
			r.events = append(r.events, event)
			return nil
		}
	}
	return ErrMemberMissing
}

func (r *repositoryStub) ListAudit(
	_ context.Context,
	_ uuid.UUID,
	_ int,
	_ int,
) ([]AuditEvent, int, error) {
	return append([]AuditEvent(nil), r.events...), len(r.events), nil
}

func TestOwnerManagesMembersAndCreatesAuditTrail(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	repository := &repositoryStub{members: []Member{{
		UserID: ownerID, Email: "owner@example.com", DisplayName: "Owner",
		Role: identity.RoleOwner,
	}}}
	service := NewService(repository)
	ctx := testPrincipal(ownerID, identity.RoleOwner)

	added, err := service.AddMember(ctx, AddMemberInput{
		Email: " Operator@Example.com ", Role: identity.RoleOperator,
	})
	if err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}
	if added.Email != "operator@example.com" ||
		added.Role != identity.RoleOperator {
		t.Fatalf("AddMember() = %#v", added)
	}
	updated, err := service.UpdateMemberRole(
		ctx,
		added.UserID,
		identity.RoleAdmin,
	)
	if err != nil || updated.Role != identity.RoleAdmin {
		t.Fatalf("UpdateMemberRole() = %#v, %v", updated, err)
	}
	if err := service.RemoveMember(ctx, added.UserID); err != nil {
		t.Fatalf("RemoveMember() error = %v", err)
	}
	audit, err := service.ListAudit(ctx, 0, 0)
	if err != nil || audit.TotalItems != 3 || audit.Page != 1 ||
		audit.PageSize != 50 {
		t.Fatalf("ListAudit() = %#v, %v", audit, err)
	}
}

func TestAdminCannotManagePrivilegedMembers(t *testing.T) {
	t.Parallel()

	adminID := uuid.New()
	ownerID := uuid.New()
	repository := &repositoryStub{members: []Member{
		{UserID: ownerID, Email: "owner@example.com", Role: identity.RoleOwner},
		{UserID: adminID, Email: "admin@example.com", Role: identity.RoleAdmin},
	}}
	service := NewService(repository)
	ctx := testPrincipal(adminID, identity.RoleAdmin)

	if _, err := service.AddMember(ctx, AddMemberInput{
		Email: "second-admin@example.com", Role: identity.RoleAdmin,
	}); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("AddMember(admin) error = %v", err)
	}
	if err := service.RemoveMember(ctx, ownerID); !errors.Is(
		err,
		identity.ErrForbidden,
	) {
		t.Fatalf("RemoveMember(owner) error = %v", err)
	}
	if _, err := service.UpdateMemberRole(
		ctx,
		adminID,
		identity.RoleViewer,
	); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("UpdateMemberRole(self) error = %v", err)
	}
}

func TestOperatorCannotReadMembershipAdministration(t *testing.T) {
	t.Parallel()

	service := NewService(&repositoryStub{})
	ctx := testPrincipal(uuid.New(), identity.RoleOperator)
	if _, err := service.ListMembers(ctx); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("ListMembers() error = %v", err)
	}
}

func testPrincipal(userID uuid.UUID, role identity.Role) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{
		Account: identity.Account{User: identity.User{ID: userID}},
		Workspace: identity.Workspace{
			ID: uuid.New(), Role: role,
		},
	})
}
