package root

import (
	"bytes"
	"testing"
)

func TestRootCommand(t *testing.T) {
	cmd := GetRootCmd()
	if cmd.Use != "zen" {
		t.Errorf("Expected root command 'zen', got '%s'", cmd.Use)
	}
}

func TestRootCommandHelp(t *testing.T) {
	cmd := GetRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected help output, got empty string")
	}
}

func TestVersionSubcommandExists(t *testing.T) {
	cmd := GetRootCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'version' subcommand to be registered")
	}
}
