package collaboration

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
)

type repositoryStub struct {
	members []Member
	events  []AuditEvent
	apiKeys []APIKey
}

func (r *repositoryStub) ListAPIKeys(
	_ context.Context,
	_ uuid.UUID,
) ([]APIKey, error) {
	return append([]APIKey(nil), r.apiKeys...), nil
}

func (r *repositoryStub) CreateAPIKey(
	_ context.Context,
	key APIKey,
	event AuditEvent,
) error {
	r.apiKeys = append(r.apiKeys, key)
	r.events = append(r.events, event)
	return nil
}

func (r *repositoryStub) RevokeAPIKey(
	_ context.Context,
	_ uuid.UUID,
	keyID uuid.UUID,
	revokedAt time.Time,
	event AuditEvent,
) error {
	for index := range r.apiKeys {
		if r.apiKeys[index].ID == keyID && r.apiKeys[index].RevokedAt == nil {
			r.apiKeys[index].RevokedAt = &revokedAt
			r.events = append(r.events, event)
			return nil
		}
	}
	return ErrAPIKeyMissing
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

func TestServiceCreatesAndRevokesHashedAPIKey(t *testing.T) {
	t.Parallel()

	repository := &repositoryStub{}
	service := NewService(repository)
	ctx := testPrincipal(uuid.New(), identity.RoleAdmin)
	created, err := service.CreateAPIKey(ctx, CreateAPIKeyInput{
		Name: "CI deploy", Role: identity.RoleOperator,
	})
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if created.Token == "" || created.Prefix == "" ||
		len(repository.apiKeys) != 1 ||
		len(repository.apiKeys[0].TokenHash) != sha256.Size {
		t.Fatalf("CreateAPIKey() = %#v", created)
	}
	if string(repository.apiKeys[0].TokenHash) == created.Token {
		t.Fatal("repository received the plaintext API key")
	}
	if err := service.RevokeAPIKey(ctx, created.ID); err != nil {
		t.Fatalf("RevokeAPIKey() error = %v", err)
	}
	if repository.apiKeys[0].RevokedAt == nil || len(repository.events) != 2 {
		t.Fatalf(
			"revoked key/events = %#v/%#v",
			repository.apiKeys[0],
			repository.events,
		)
	}
}

func TestServiceRejectsPrivilegedOrLongLivedAPIKey(t *testing.T) {
	t.Parallel()

	service := NewService(&repositoryStub{})
	ctx := testPrincipal(uuid.New(), identity.RoleOwner)
	if _, err := service.CreateAPIKey(ctx, CreateAPIKeyInput{
		Name: "Owner key", Role: identity.RoleOwner,
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateAPIKey(owner) error = %v", err)
	}
	expiresAt := time.Now().Add(366 * 24 * time.Hour)
	if _, err := service.CreateAPIKey(ctx, CreateAPIKeyInput{
		Name: "Long lived", Role: identity.RoleViewer, ExpiresAt: &expiresAt,
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateAPIKey(long lived) error = %v", err)
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
