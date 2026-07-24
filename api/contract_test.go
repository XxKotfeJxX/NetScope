package api_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIContractStructure(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(source, &document); err != nil {
		t.Fatalf("parse OpenAPI contract: %v", err)
	}
	if document["openapi"] != "3.1.0" {
		t.Fatalf("openapi = %v, want 3.1.0", document["openapi"])
	}
	info := object(t, document, "info")
	version, _ := info["version"].(string)
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(version) {
		t.Fatalf("info.version = %q, want semantic version", version)
	}

	paths := object(t, document, "paths")
	for _, required := range []string{
		"/healthz",
		"/readyz",
		"/api/v1/capabilities",
		"/api/v1/auth/register",
		"/api/v1/auth/login",
		"/api/v1/me",
		"/api/v1/workspaces",
		"/api/v1/workspace/members",
		"/api/v1/workspace/api-keys",
		"/api/v1/runs",
		"/api/v1/runs/{id}",
		"/api/v1/runs/{id}/comments",
		"/api/v1/runs/{id}/public-links",
		"/api/v1/public/reports/{token}",
		"/api/v1/targets",
		"/api/v1/targets/{targetID}",
		"/api/v1/monitoring",
	} {
		if _, exists := paths[required]; !exists {
			t.Errorf("required path %s is undocumented", required)
		}
	}

	operationIDs := make(map[string]string)
	methods := map[string]bool{
		"delete": true, "get": true, "head": true, "options": true,
		"patch": true, "post": true, "put": true,
	}
	for path, rawItem := range paths {
		item, ok := rawItem.(map[string]any)
		if !ok {
			t.Errorf("path %s is not an object", path)
			continue
		}
		for method, rawOperation := range item {
			if !methods[method] {
				continue
			}
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				t.Errorf("%s %s is not an object", method, path)
				continue
			}
			id, _ := operation["operationId"].(string)
			if id == "" {
				t.Errorf("%s %s has no operationId", method, path)
			} else if previous, duplicate := operationIDs[id]; duplicate {
				t.Errorf("operationId %q is used by %s and %s %s", id, previous, method, path)
			} else {
				operationIDs[id] = method + " " + path
			}
			responses, ok := operation["responses"].(map[string]any)
			if !ok || len(responses) == 0 {
				t.Errorf("%s %s has no responses", method, path)
			}
		}
	}
	if len(operationIDs) < 30 {
		t.Fatalf("only %d operations were validated", len(operationIDs))
	}

	components := object(t, document, "components")
	responses := object(t, components, "responses")
	if _, exists := responses["TooManyRequests"]; !exists {
		t.Error("components.responses.TooManyRequests is missing")
	}
	securitySchemes := object(t, components, "securitySchemes")
	for _, scheme := range []string{"sessionCookie", "bearerSession"} {
		if _, exists := securitySchemes[scheme]; !exists {
			t.Errorf("security scheme %s is missing", scheme)
		}
	}
	validateReferences(t, document, document, "$")
}

func object(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%s is not an object", key)
	}
	return value
}

func validateReferences(
	t *testing.T,
	root map[string]any,
	value any,
	location string,
) {
	t.Helper()
	switch current := value.(type) {
	case map[string]any:
		for key, nested := range current {
			if key == "$ref" {
				reference, ok := nested.(string)
				if !ok || !strings.HasPrefix(reference, "#/") {
					t.Errorf("%s.$ref = %v, only internal references are allowed", location, nested)
					continue
				}
				if _, err := resolveReference(root, reference); err != nil {
					t.Errorf("%s.$ref: %v", location, err)
				}
				continue
			}
			validateReferences(t, root, nested, location+"."+key)
		}
	case []any:
		for index, nested := range current {
			validateReferences(t, root, nested, fmt.Sprintf("%s[%d]", location, index))
		}
	}
}

func resolveReference(root map[string]any, reference string) (any, error) {
	var current any = root
	for _, rawPart := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
		part := strings.ReplaceAll(strings.ReplaceAll(rawPart, "~1", "/"), "~0", "~")
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s traverses a non-object", reference)
		}
		current, ok = object[part]
		if !ok {
			return nil, fmt.Errorf("%s does not resolve", reference)
		}
	}
	return current, nil
}
