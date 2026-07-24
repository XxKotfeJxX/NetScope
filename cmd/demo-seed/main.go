package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

type config struct {
	apiURL        string
	webOrigin     string
	email         string
	password      string
	displayName   string
	workspaceName string
}

type seeder struct {
	config      config
	client      *http.Client
	workspaceID string
}

type seedSummary struct {
	targetsCreated int
	runsCreated    int
}

type accountResponse struct {
	ActiveWorkspace struct {
		ID string `json:"id"`
	} `json:"activeWorkspace"`
}

type page[T any] struct {
	Items []T `json:"items"`
}

type targetRecord struct {
	Address string `json:"address"`
}

type runRecord struct {
	Target string `json:"target"`
}

type demoTarget struct {
	Name    string
	Address string
	Tags    []string
	Checks  []string
	Ports   []int
}

var demoTargets = []demoTarget{
	{
		Name: "Example Domain", Address: "example.com",
		Tags: []string{"demo", "web"}, Checks: []string{"dns", "tcp", "tls", "http"},
		Ports: []int{80, 443},
	},
	{
		Name: "GitHub", Address: "github.com",
		Tags: []string{"demo", "developer"}, Checks: []string{"dns", "tcp", "tls", "http"},
		Ports: []int{443},
	},
	{
		Name: "Cloudflare Resolver", Address: "1.1.1.1",
		Tags: []string{"demo", "dns"}, Checks: []string{"dns", "tcp"},
		Ports: []int{53, 443},
	},
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "demo seed configuration:", err)
		os.Exit(1)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create cookie jar:", err)
		os.Exit(1)
	}
	client := &http.Client{Jar: jar, Timeout: 30 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	summary, err := seed(ctx, cfg, client)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed demo workspace:", err)
		os.Exit(1)
	}
	fmt.Printf(
		"Demo workspace ready for %s (%d targets and %d runs created).\n",
		cfg.email,
		summary.targetsCreated,
		summary.runsCreated,
	)
}

func loadConfig() (config, error) {
	cfg := config{
		apiURL:        env("NETSCOPE_API_URL", "http://localhost:8080"),
		webOrigin:     env("NETSCOPE_WEB_ORIGIN", "http://localhost:5173"),
		email:         env("DEMO_EMAIL", "demo@netscope.local"),
		password:      strings.TrimSpace(os.Getenv("DEMO_PASSWORD")),
		displayName:   env("DEMO_DISPLAY_NAME", "Demo Operator"),
		workspaceName: env("DEMO_WORKSPACE_NAME", "NetScope Demo"),
	}
	if len(cfg.password) < 12 {
		return config{}, errors.New("DEMO_PASSWORD must contain at least 12 characters")
	}
	for name, value := range map[string]string{
		"NETSCOPE_API_URL":    cfg.apiURL,
		"NETSCOPE_WEB_ORIGIN": cfg.webOrigin,
	} {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") {
			return config{}, fmt.Errorf("%s must be an HTTP(S) URL", name)
		}
	}
	cfg.apiURL = strings.TrimRight(cfg.apiURL, "/")
	cfg.webOrigin = strings.TrimRight(cfg.webOrigin, "/")
	return cfg, nil
}

func seed(
	ctx context.Context,
	cfg config,
	client *http.Client,
) (seedSummary, error) {
	s := &seeder{config: cfg, client: client}
	if err := s.authenticate(ctx); err != nil {
		return seedSummary{}, err
	}

	existingTargets, err := s.listTargets(ctx)
	if err != nil {
		return seedSummary{}, err
	}
	existingRuns, err := s.listRuns(ctx)
	if err != nil {
		return seedSummary{}, err
	}
	targets := make(map[string]bool, len(existingTargets))
	for _, target := range existingTargets {
		targets[target.Address] = true
	}
	runs := make(map[string]bool, len(existingRuns))
	for _, run := range existingRuns {
		runs[run.Target] = true
	}

	var summary seedSummary
	for _, target := range demoTargets {
		if !targets[target.Address] {
			if err := s.createTarget(ctx, target); err != nil {
				return summary, err
			}
			summary.targetsCreated++
		}
		if !runs[target.Address] {
			if err := s.createRun(ctx, target); err != nil {
				return summary, err
			}
			summary.runsCreated++
		}
	}
	return summary, nil
}

func (s *seeder) authenticate(ctx context.Context) error {
	registration := map[string]string{
		"email": s.config.email, "password": s.config.password,
		"displayName": s.config.displayName, "workspaceName": s.config.workspaceName,
	}
	status, body, err := s.request(ctx, http.MethodPost, "/api/v1/auth/register", registration)
	if err != nil {
		return err
	}
	if status == http.StatusConflict {
		status, body, err = s.request(ctx, http.MethodPost, "/api/v1/auth/login", map[string]string{
			"email": s.config.email, "password": s.config.password,
		})
		if err != nil {
			return err
		}
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return apiResponseError("authenticate demo account", status, body)
	}
	var account accountResponse
	if err := json.Unmarshal(body, &account); err != nil {
		return fmt.Errorf("decode demo account: %w", err)
	}
	if account.ActiveWorkspace.ID == "" {
		return errors.New("demo account response omitted active workspace")
	}
	s.workspaceID = account.ActiveWorkspace.ID
	return nil
}

func (s *seeder) listTargets(ctx context.Context) ([]targetRecord, error) {
	status, body, err := s.request(
		ctx,
		http.MethodGet,
		"/api/v1/targets?page=1&pageSize=100",
		nil,
	)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, apiResponseError("list demo targets", status, body)
	}
	var result page[targetRecord]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode demo targets: %w", err)
	}
	return result.Items, nil
}

func (s *seeder) listRuns(ctx context.Context) ([]runRecord, error) {
	status, body, err := s.request(
		ctx,
		http.MethodGet,
		"/api/v1/runs?page=1&pageSize=100",
		nil,
	)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, apiResponseError("list demo runs", status, body)
	}
	var result page[runRecord]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode demo runs: %w", err)
	}
	return result.Items, nil
}

func (s *seeder) createTarget(ctx context.Context, target demoTarget) error {
	payload := map[string]any{
		"name": target.Name, "address": target.Address, "tags": target.Tags,
		"checks": target.Checks, "intervalSeconds": 300, "failureThreshold": 3,
		"options": runOptions(target.Ports),
	}
	status, body, err := s.request(
		ctx,
		http.MethodPost,
		"/api/v1/targets",
		payload,
	)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return apiResponseError("create demo target "+target.Address, status, body)
	}
	return nil
}

func (s *seeder) createRun(ctx context.Context, target demoTarget) error {
	payload := map[string]any{
		"target": target.Address, "checks": target.Checks,
		"options": runOptions(target.Ports),
	}
	status, body, err := s.request(ctx, http.MethodPost, "/api/v1/runs", payload)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return apiResponseError("create demo run "+target.Address, status, body)
	}
	return nil
}

func runOptions(ports []int) map[string]any {
	return map[string]any{
		"timeoutMs": 5000, "tcpPorts": ports, "httpMethod": "GET",
		"followRedirects": true, "maxRedirects": 5, "ipVersion": "auto",
		"pingPackets": 4, "maxHops": 20,
	}
}

func (s *seeder) request(
	ctx context.Context,
	method string,
	path string,
	payload any,
) (int, []byte, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, fmt.Errorf("encode %s: %w", path, err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		s.config.apiURL+path,
		body,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("create %s request: %w", path, err)
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet && method != http.MethodHead {
		request.Header.Set("Origin", s.config.webOrigin)
	}
	if s.workspaceID != "" {
		request.Header.Set("X-Workspace-ID", s.workspaceID)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return 0, nil, fmt.Errorf("read %s response: %w", path, err)
	}
	return response.StatusCode, responseBody, nil
}

func apiResponseError(action string, status int, body []byte) error {
	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &response) == nil && response.Error.Code != "" {
		return fmt.Errorf(
			"%s: HTTP %d %s: %s",
			action,
			status,
			response.Error.Code,
			response.Error.Message,
		)
	}
	return fmt.Errorf("%s: HTTP %d", action, status)
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
