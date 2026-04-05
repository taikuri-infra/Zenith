package services

import (
	"strings"
	"testing"
)

// --- getEmailContent tests ---

func TestGetEmailContent_Welcome(t *testing.T) {
	subject, body := getEmailContent("welcome", "Alice", "https://app.zenith.dev")
	if subject != "Welcome to Zenith!" {
		t.Errorf("Expected welcome subject, got '%s'", subject)
	}
	if !strings.Contains(body, "Alice") {
		t.Error("Expected body to contain user name")
	}
	if !strings.Contains(body, "https://app.zenith.dev") {
		t.Error("Expected body to contain app URL")
	}
}

func TestGetEmailContent_Day1Deploy(t *testing.T) {
	subject, body := getEmailContent("day1_deploy", "Bob", "https://app.zenith.dev")
	if subject == "" {
		t.Error("Expected non-empty subject")
	}
	if !strings.Contains(body, "Bob") {
		t.Error("Expected body to contain user name")
	}
}

func TestGetEmailContent_Day3Engage(t *testing.T) {
	subject, body := getEmailContent("day3_engage", "Charlie", "https://app.zenith.dev")
	if subject == "" {
		t.Error("Expected non-empty subject")
	}
	if !strings.Contains(body, "Charlie") {
		t.Error("Expected body to contain user name")
	}
}

func TestGetEmailContent_Day3Nudge(t *testing.T) {
	subject, body := getEmailContent("day3_nudge", "Dave", "https://app.zenith.dev")
	if subject == "" {
		t.Error("Expected non-empty subject")
	}
	if !strings.Contains(body, "Dave") {
		t.Error("Expected body to contain user name")
	}
}

func TestGetEmailContent_Day7Trial(t *testing.T) {
	subject, body := getEmailContent("day7_trial", "Eve", "https://app.zenith.dev")
	if subject == "" {
		t.Error("Expected non-empty subject")
	}
	if !strings.Contains(body, "Eve") {
		t.Error("Expected body to contain user name")
	}
	if !strings.Contains(body, "billing") {
		t.Error("Expected body to contain billing link")
	}
}

func TestGetEmailContent_Day14Value(t *testing.T) {
	subject, body := getEmailContent("day14_value", "Frank", "https://app.zenith.dev")
	if subject == "" {
		t.Error("Expected non-empty subject")
	}
	if !strings.Contains(body, "Frank") {
		t.Error("Expected body to contain user name")
	}
	if !strings.Contains(body, "billing") {
		t.Error("Expected body to contain billing link")
	}
}

func TestGetEmailContent_Unknown(t *testing.T) {
	subject, body := getEmailContent("unknown_template", "Gina", "https://app.zenith.dev")
	if subject != "Update from Zenith" {
		t.Errorf("Expected default subject, got '%s'", subject)
	}
	if !strings.Contains(body, "Gina") {
		t.Error("Expected body to contain user name")
	}
}

func TestGetEmailContent_AllTemplatesHaveLinks(t *testing.T) {
	templates := []string{"welcome", "day1_deploy", "day3_engage", "day3_nudge", "day7_trial", "day14_value", "unknown"}
	for _, tmpl := range templates {
		_, body := getEmailContent(tmpl, "Test", "https://app.zenith.dev")
		if !strings.Contains(body, "https://app.zenith.dev") {
			t.Errorf("Template %s body should contain app URL", tmpl)
		}
	}
}
