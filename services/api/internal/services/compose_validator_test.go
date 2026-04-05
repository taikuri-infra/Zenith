package services

import (
	"strings"
	"testing"
)

func TestValidateCompose_EmptyServices(t *testing.T) {
	parsed := &ParsedCompose{
		Services: nil,
	}
	issues := ValidateCompose(parsed)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "no app services") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'no app services' error for empty services")
	}
}

func TestValidateCompose_ValidSingleService(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "web", Port: 3000, Image: "node:18"},
		},
	}
	issues := ValidateCompose(parsed)
	for _, issue := range issues {
		if strings.HasPrefix(issue, "error:") {
			t.Errorf("Unexpected error for valid service: %s", issue)
		}
	}
}

func TestValidateCompose_DuplicateServiceNames(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "web", Port: 3000, Image: "node:18"},
			{Name: "web", Port: 8080, Image: "python:3.11"},
		},
	}
	issues := ValidateCompose(parsed)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "duplicate service name 'web'") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected duplicate service name error")
	}
}

func TestValidateCompose_DuplicateServiceAndManaged(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "db", Port: 5432, Image: "myapp:latest"},
		},
		ManagedServices: []ParsedManaged{
			{Name: "db", Type: "postgresql"},
		},
	}
	issues := ValidateCompose(parsed)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "duplicate service name 'db'") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected duplicate service name error across services and managed services")
	}
}

func TestValidateCompose_InvalidPort(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "web", Port: 99999, Image: "node:18"},
		},
	}
	issues := ValidateCompose(parsed)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "invalid port") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected invalid port error")
	}
}

func TestValidateCompose_MissingDependsOnReference(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "web", Port: 3000, Image: "node:18", DependsOn: []string{"redis"}},
		},
	}
	issues := ValidateCompose(parsed)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "depends on 'redis'") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected missing depends_on reference warning")
	}
}

func TestValidateCompose_ValidDependsOnReference(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "web", Port: 3000, Image: "node:18", DependsOn: []string{"cache"}},
		},
		ManagedServices: []ParsedManaged{
			{Name: "cache", Type: "redis"},
		},
	}
	issues := ValidateCompose(parsed)
	for _, issue := range issues {
		if strings.Contains(issue, "depends on 'cache'") {
			t.Errorf("Should not warn about valid depends_on reference: %s", issue)
		}
	}
}

func TestValidateCompose_WarningNoPortNoBuildNoImage(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "worker"},
		},
	}
	issues := ValidateCompose(parsed)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "no port, build context, or image") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning for service without port, build context, or image")
	}
}

func TestValidateCompose_NoWarningWithImage(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "worker", Image: "worker:latest"},
		},
	}
	issues := ValidateCompose(parsed)
	for _, issue := range issues {
		if strings.Contains(issue, "no port, build context, or image") {
			t.Errorf("Should not warn for service with image: %s", issue)
		}
	}
}

func TestValidateCompose_NoWarningWithBuildContext(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "worker", BuildContext: "."},
		},
	}
	issues := ValidateCompose(parsed)
	for _, issue := range issues {
		if strings.Contains(issue, "no port, build context, or image") {
			t.Errorf("Should not warn for service with build context: %s", issue)
		}
	}
}

func TestValidateCompose_MultipleIssues(t *testing.T) {
	parsed := &ParsedCompose{
		Services: []ParsedService{
			{Name: "web", Port: 99999, Image: "node:18", DependsOn: []string{"missing-dep"}},
			{Name: "web", Port: 3000, Image: "python:3.11"},
		},
	}
	issues := ValidateCompose(parsed)
	if len(issues) < 2 {
		t.Errorf("Expected at least 2 issues, got %d: %v", len(issues), issues)
	}
}
