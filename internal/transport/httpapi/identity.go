package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/identity"
)

const sessionCookieName = "netscope_session"

type accountContextKey struct{}

type identityHandler struct {
	service      *identity.Service
	cookieSecure bool
}

type currentAccountResponse struct {
	identity.Account
	ActiveWorkspace identity.Workspace `json:"activeWorkspace"`
}

func (h identityHandler) register(w http.ResponseWriter, r *http.Request) {
	var input identity.RegistrationInput
	if !decodeIdentityJSON(w, r, &input) {
		return
	}
	result, err := h.service.Register(r.Context(), input)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	h.setSessionCookie(w, result.Token, result.ExpiresAt)
	writeJSON(w, http.StatusCreated, currentAccountResponse{
		Account: result.Account, ActiveWorkspace: result.Workspaces[0],
	})
}

func (h identityHandler) login(w http.ResponseWriter, r *http.Request) {
	var input identity.LoginInput
	if !decodeIdentityJSON(w, r, &input) {
		return
	}
	result, err := h.service.Login(r.Context(), input)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	h.setSessionCookie(w, result.Token, result.ExpiresAt)
	response := currentAccountResponse{Account: result.Account}
	if len(result.Workspaces) > 0 {
		response.ActiveWorkspace = result.Workspaces[0]
	}
	writeJSON(w, http.StatusOK, response)
}

func (h identityHandler) logout(w http.ResponseWriter, r *http.Request) {
	token := identityToken(r)
	if err := h.service.Logout(r.Context(), token); err != nil {
		writeAPIError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/",
		HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
		MaxAge: -1, Expires: time.Unix(1, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h identityHandler) me(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeAPIError(w, r, identity.ErrUnauthenticated)
		return
	}
	principal, err := h.service.SelectWorkspace(
		r.Context(),
		account,
		strings.TrimSpace(r.Header.Get("X-Workspace-ID")),
	)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, currentAccountResponse{
		Account: account, ActiveWorkspace: principal.Workspace,
	})
}

func (h identityHandler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeAPIError(w, r, identity.ErrUnauthenticated)
		return
	}
	workspaces, err := h.service.ListWorkspaces(r.Context(), account.User.ID)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, workspaces)
}

func (h identityHandler) createWorkspace(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeAPIError(w, r, identity.ErrUnauthenticated)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeIdentityJSON(w, r, &input) {
		return
	}
	workspace, err := h.service.CreateWorkspace(
		r.Context(),
		account.User.ID,
		input.Name,
	)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, workspace)
}

func (h identityHandler) requireAccount(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account, err := h.service.Authenticate(r.Context(), identityToken(r))
		if err != nil {
			writeAPIError(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), accountContextKey{}, account)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h identityHandler) setSessionCookie(
	w http.ResponseWriter,
	token string,
	expiresAt time.Time,
) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/",
		HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
		Expires: expiresAt,
		MaxAge:  int(time.Until(expiresAt).Seconds()),
	})
}

func identityToken(r *http.Request) string {
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(authorization, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		return cookie.Value
	}
	return ""
}

func accountFromContext(ctx context.Context) (identity.Account, bool) {
	account, ok := ctx.Value(accountContextKey{}).(identity.Account)
	return account, ok
}

func decodeIdentityJSON(w http.ResponseWriter, r *http.Request, payload any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(payload); err != nil {
		writeAPIError(w, r, errInvalidRequest)
		return false
	}
	return true
}
