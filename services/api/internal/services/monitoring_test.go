package services

import (
	"testing"
)

// --- podSelector tests ---

func TestPodSelector(t *testing.T) {
	expected := "zenith.dev/app=my-app"
	got := podSelector("my-app")
	if got != expected {
		t.Errorf("podSelector('my-app') = '%s', want '%s'", got, expected)
	}
}

// --- podRegex tests ---

func TestPodRegex(t *testing.T) {
	expected := "my-app-.*"
	got := podRegex("my-app")
	if got != expected {
		t.Errorf("podRegex('my-app') = '%s', want '%s'", got, expected)
	}
}
