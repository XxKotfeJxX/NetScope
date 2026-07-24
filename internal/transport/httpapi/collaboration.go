package httpapi

import (
	"net/http"
	"strconv"

	"github.com/XxKotfeJxX/netscope/internal/collaboration"
	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type collaborationHandler struct {
	service *collaboration.Service
}

func (h collaborationHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	members, err := h.service.ListMembers(r.Context())
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, members)
}

func (h collaborationHandler) addMember(w http.ResponseWriter, r *http.Request) {
	var input collaboration.AddMemberInput
	if !decodeIdentityJSON(w, r, &input) {
		return
	}
	member, err := h.service.AddMember(r.Context(), input)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

func (h collaborationHandler) updateMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := memberID(w, r)
	if !ok {
		return
	}
	var input struct {
		Role identity.Role `json:"role"`
	}
	if !decodeIdentityJSON(w, r, &input) {
		return
	}
	member, err := h.service.UpdateMemberRole(r.Context(), userID, input.Role)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (h collaborationHandler) removeMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := memberID(w, r)
	if !ok {
		return
	}
	if err := h.service.RemoveMember(r.Context(), userID); err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h collaborationHandler) listAudit(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	result, err := h.service.ListAudit(r.Context(), page, pageSize)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func memberID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	value, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeAPIError(w, r, collaboration.ErrMemberMissing)
		return uuid.Nil, false
	}
	return value, true
}
