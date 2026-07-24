package httpapi

import (
	"net/http"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/reports"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type reportsHandler struct {
	service *reports.Service
}

func (h reportsHandler) listComments(w http.ResponseWriter, r *http.Request) {
	runID, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	comments, err := h.service.ListComments(r.Context(), runID)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, comments)
}

func (h reportsHandler) createComment(w http.ResponseWriter, r *http.Request) {
	runID, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	var input struct {
		Body string `json:"body"`
	}
	if !decodeIdentityJSON(w, r, &input) {
		return
	}
	comment, err := h.service.CreateComment(r.Context(), runID, input.Body)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (h reportsHandler) deleteComment(w http.ResponseWriter, r *http.Request) {
	runID, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	commentID, err := uuid.Parse(chi.URLParam(r, "commentID"))
	if err != nil {
		writeAPIError(w, r, reports.ErrCommentMissing)
		return
	}
	if err := h.service.DeleteComment(r.Context(), runID, commentID); err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h reportsHandler) listPublicLinks(w http.ResponseWriter, r *http.Request) {
	runID, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	links, err := h.service.ListPublicLinks(r.Context(), runID)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, links)
}

func (h reportsHandler) createPublicLink(w http.ResponseWriter, r *http.Request) {
	runID, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	var input struct {
		ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	}
	if !decodeIdentityJSON(w, r, &input) {
		return
	}
	link, err := h.service.CreatePublicLink(r.Context(), runID, input.ExpiresAt)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, link)
}

func (h reportsHandler) revokePublicLink(w http.ResponseWriter, r *http.Request) {
	runID, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	linkID, err := uuid.Parse(chi.URLParam(r, "linkID"))
	if err != nil {
		writeAPIError(w, r, reports.ErrPublicLinkMissing)
		return
	}
	if err := h.service.RevokePublicLink(r.Context(), runID, linkID); err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h reportsHandler) publicReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.PublicReport(
		r.Context(),
		chi.URLParam(r, "token"),
	)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, report)
}
