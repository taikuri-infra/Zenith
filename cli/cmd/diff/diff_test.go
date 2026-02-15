package diff

import (
	"testing"
)

func TestCompareSpecs_Identical(t *testing.T) {
	local := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
		"port":     8080,
	}
	remote := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
		"port":     8080,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 0 {
		t.Errorf("Expected no changes for identical specs, got %d", len(changes))
	}
}

func TestCompareSpecs_Modified(t *testing.T) {
	local := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 3,
		"port":     8080,
	}
	remote := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
		"port":     8080,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}

	if changes[0].Field != "replicas" {
		t.Errorf("Expected changed field 'replicas', got '%s'", changes[0].Field)
	}
	if changes[0].OldValue != "2" {
		t.Errorf("Expected old value '2', got '%s'", changes[0].OldValue)
	}
	if changes[0].NewValue != "3" {
		t.Errorf("Expected new value '3', got '%s'", changes[0].NewValue)
	}
}

func TestCompareSpecs_FieldAdded(t *testing.T) {
	local := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
		"domain":   "app.example.com",
	}
	remote := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change (new field), got %d", len(changes))
	}

	if changes[0].Field != "domain" {
		t.Errorf("Expected field 'domain', got '%s'", changes[0].Field)
	}
	if changes[0].OldValue != "" {
		t.Errorf("Expected empty old value for new field, got '%s'", changes[0].OldValue)
	}
	if changes[0].NewValue != "app.example.com" {
		t.Errorf("Expected new value 'app.example.com', got '%s'", changes[0].NewValue)
	}
}

func TestCompareSpecs_FieldRemoved(t *testing.T) {
	local := map[string]interface{}{
		"image": "nginx:latest",
	}
	remote := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change (removed field), got %d", len(changes))
	}

	if changes[0].Field != "replicas" {
		t.Errorf("Expected field 'replicas', got '%s'", changes[0].Field)
	}
	if changes[0].OldValue != "2" {
		t.Errorf("Expected old value '2', got '%s'", changes[0].OldValue)
	}
	if changes[0].NewValue != "" {
		t.Errorf("Expected empty new value for removed field, got '%s'", changes[0].NewValue)
	}
}

func TestCompareSpecs_MultipleChanges(t *testing.T) {
	local := map[string]interface{}{
		"image":    "nginx:1.25",
		"replicas": 5,
		"port":     9090,
		"domain":   "new.example.com",
	}
	remote := map[string]interface{}{
		"image":    "nginx:1.24",
		"replicas": 2,
		"port":     8080,
		"env":      "production",
	}

	changes := CompareSpecs(local, remote)

	// 3 modified (image, replicas, port), 1 added (domain), 1 removed (env) = 5
	if len(changes) != 5 {
		t.Fatalf("Expected 5 changes, got %d", len(changes))
	}

	changeMap := make(map[string]FieldChange)
	for _, c := range changes {
		changeMap[c.Field] = c
	}

	// Verify image change
	if c, ok := changeMap["image"]; ok {
		if c.OldValue != "nginx:1.24" || c.NewValue != "nginx:1.25" {
			t.Errorf("Image change: expected '1.24' -> '1.25', got '%s' -> '%s'", c.OldValue, c.NewValue)
		}
	} else {
		t.Error("Expected image change")
	}

	// Verify domain addition
	if c, ok := changeMap["domain"]; ok {
		if c.OldValue != "" || c.NewValue != "new.example.com" {
			t.Errorf("Domain addition: expected '' -> 'new.example.com', got '%s' -> '%s'", c.OldValue, c.NewValue)
		}
	} else {
		t.Error("Expected domain change")
	}

	// Verify env removal
	if c, ok := changeMap["env"]; ok {
		if c.OldValue != "production" || c.NewValue != "" {
			t.Errorf("Env removal: expected 'production' -> '', got '%s' -> '%s'", c.OldValue, c.NewValue)
		}
	} else {
		t.Error("Expected env change")
	}
}

func TestCompareSpecs_EmptySpecs(t *testing.T) {
	local := map[string]interface{}{}
	remote := map[string]interface{}{}

	changes := CompareSpecs(local, remote)
	if len(changes) != 0 {
		t.Errorf("Expected no changes for empty specs, got %d", len(changes))
	}
}

func TestCompareSpecs_AllNew(t *testing.T) {
	local := map[string]interface{}{
		"image": "nginx:latest",
		"port":  8080,
	}
	remote := map[string]interface{}{}

	changes := CompareSpecs(local, remote)
	if len(changes) != 2 {
		t.Fatalf("Expected 2 changes (all new), got %d", len(changes))
	}

	for _, c := range changes {
		if c.OldValue != "" {
			t.Errorf("Expected empty old value for new field %s, got '%s'", c.Field, c.OldValue)
		}
	}
}

func TestCompareSpecs_AllRemoved(t *testing.T) {
	local := map[string]interface{}{}
	remote := map[string]interface{}{
		"image": "nginx:latest",
		"port":  8080,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 2 {
		t.Fatalf("Expected 2 changes (all removed), got %d", len(changes))
	}

	for _, c := range changes {
		if c.NewValue != "" {
			t.Errorf("Expected empty new value for removed field %s, got '%s'", c.Field, c.NewValue)
		}
	}
}

func TestDiffEntry_Types(t *testing.T) {
	entries := []DiffEntry{
		{Kind: "App", Name: "app1", Action: "added"},
		{Kind: "App", Name: "app2", Action: "modified"},
		{Kind: "Database", Name: "db1", Action: "deleted"},
		{Kind: "App", Name: "app3", Action: "unchanged"},
	}

	actions := make(map[string]int)
	for _, e := range entries {
		actions[e.Action]++
	}

	if actions["added"] != 1 {
		t.Errorf("Expected 1 added, got %d", actions["added"])
	}
	if actions["modified"] != 1 {
		t.Errorf("Expected 1 modified, got %d", actions["modified"])
	}
	if actions["deleted"] != 1 {
		t.Errorf("Expected 1 deleted, got %d", actions["deleted"])
	}
	if actions["unchanged"] != 1 {
		t.Errorf("Expected 1 unchanged, got %d", actions["unchanged"])
	}
}

func TestFieldChange_Representation(t *testing.T) {
	tests := []struct {
		name     string
		change   FieldChange
		isAdd    bool
		isMod    bool
		isDel    bool
	}{
		{
			name:  "addition",
			change: FieldChange{Field: "domain", OldValue: "", NewValue: "app.example.com"},
			isAdd: true,
		},
		{
			name:  "modification",
			change: FieldChange{Field: "replicas", OldValue: "2", NewValue: "3"},
			isMod: true,
		},
		{
			name:  "deletion",
			change: FieldChange{Field: "env", OldValue: "production", NewValue: ""},
			isDel: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAdd := tt.change.OldValue == "" && tt.change.NewValue != ""
			isMod := tt.change.OldValue != "" && tt.change.NewValue != ""
			isDel := tt.change.OldValue != "" && tt.change.NewValue == ""

			if isAdd != tt.isAdd {
				t.Errorf("Expected isAdd=%v, got %v", tt.isAdd, isAdd)
			}
			if isMod != tt.isMod {
				t.Errorf("Expected isMod=%v, got %v", tt.isMod, isMod)
			}
			if isDel != tt.isDel {
				t.Errorf("Expected isDel=%v, got %v", tt.isDel, isDel)
			}
		})
	}
}
