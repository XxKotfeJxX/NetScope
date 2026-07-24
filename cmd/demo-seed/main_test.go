package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func TestSeedCreatesDemoDataIdempotently(t *testing.T) {
	t.Parallel()

	var registered bool
	var targets []targetRecord
	var runs []runRecord
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/register" {
			if registered {
				writeTestError(w, http.StatusConflict, "email_already_registered")
				return
			}
			registered = true
			http.SetCookie(w, &http.Cookie{
				Name: "netscope_session", Value: "demo-session", Path: "/",
			})
			writeTestAccount(w, http.StatusCreated)
			return
		}
		if r.URL.Path == "/api/v1/auth/login" {
			http.SetCookie(w, &http.Cookie{
				Name: "netscope_session", Value: "demo-session", Path: "/",
			})
			writeTestAccount(w, http.StatusOK)
			return
		}
		if cookie, err := r.Cookie("netscope_session"); err != nil ||
			cookie.Value != "demo-session" {
			writeTestError(w, http.StatusUnauthorized, "authentication_required")
			return
		}
		if r.Header.Get("X-Workspace-ID") != "workspace-demo" {
			writeTestError(w, http.StatusBadRequest, "workspace_required")
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/targets":
			_ = json.NewEncoder(w).Encode(page[targetRecord]{Items: targets})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/targets":
			requireOrigin(t, r)
			var target targetRecord
			if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
				t.Errorf("decode target: %v", err)
			}
			targets = append(targets, target)
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/runs":
			_ = json.NewEncoder(w).Encode(page[runRecord]{Items: runs})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs":
			requireOrigin(t, r)
			var run runRecord
			if err := json.NewDecoder(r.Body).Decode(&run); err != nil {
				t.Errorf("decode run: %v", err)
			}
			runs = append(runs, run)
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config{
		apiURL: server.URL, webOrigin: "https://netscope.example",
		email: "demo@example.test", password: "demo-password",
		displayName: "Demo Operator", workspaceName: "NetScope Demo",
	}
	client := &http.Client{Jar: jar}
	first, err := seed(context.Background(), cfg, client)
	if err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if first.targetsCreated != 3 || first.runsCreated != 3 {
		t.Fatalf("first summary = %+v", first)
	}
	second, err := seed(context.Background(), cfg, client)
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if second.targetsCreated != 0 || second.runsCreated != 0 {
		t.Fatalf("second summary = %+v", second)
	}
	if len(targets) != 3 || len(runs) != 3 {
		t.Fatalf("persisted targets = %d, runs = %d", len(targets), len(runs))
	}
}

func TestLoadConfigRequiresStrongDemoPassword(t *testing.T) {
	t.Setenv("DEMO_PASSWORD", "short")
	if _, err := loadConfig(); err == nil {
		t.Fatal("loadConfig() accepted a short demo password")
	}
}

func requireOrigin(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Origin"); got != "https://netscope.example" {
		t.Errorf("Origin = %q", got)
	}
}

func writeTestAccount(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"activeWorkspace":{"id":"workspace-demo"}}`))
}

func writeTestError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": code},
	})
}
