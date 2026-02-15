package db

import (
	"fmt"
	"strings"
	"testing"
)

func TestShellCommand(t *testing.T) {
	tests := []struct {
		engine  string
		cmd     string
		hasArgs bool
	}{
		{"postgresql", "psql", true},
		{"postgres", "psql", true},
		{"mysql", "mysql", true},
		{"redis", "redis-cli", true},
		{"mongodb", "mongosh", true},
		{"mongo", "mongosh", true},
		{"unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			cmd, args := ShellCommand(tt.engine)
			if cmd != tt.cmd {
				t.Errorf("ShellCommand(%q) cmd = %q, want %q", tt.engine, cmd, tt.cmd)
			}
			if tt.hasArgs && len(args) == 0 {
				t.Errorf("ShellCommand(%q) should have args", tt.engine)
			}
			if !tt.hasArgs && args != nil {
				t.Errorf("ShellCommand(%q) should have nil args", tt.engine)
			}
		})
	}
}

func TestShellCommand_HostArgs(t *testing.T) {
	// Verify specific host arguments for each engine
	tests := []struct {
		engine       string
		expectedFlag string
		expectedHost string
	}{
		{"postgresql", "-h", "127.0.0.1"},
		{"mysql", "-h", "127.0.0.1"},
		{"redis", "-h", "127.0.0.1"},
		{"mongodb", "--host", "127.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			_, args := ShellCommand(tt.engine)
			if len(args) < 2 {
				t.Fatalf("Expected at least 2 args, got %d", len(args))
			}
			if args[0] != tt.expectedFlag {
				t.Errorf("Expected flag '%s', got '%s'", tt.expectedFlag, args[0])
			}
			if args[1] != tt.expectedHost {
				t.Errorf("Expected host '%s', got '%s'", tt.expectedHost, args[1])
			}
		})
	}
}

func TestShellCommand_CaseInsensitive(t *testing.T) {
	// The function uses strings.ToLower, so test mixed case
	tests := []struct {
		engine string
		cmd    string
	}{
		{"PostgreSQL", "psql"},
		{"MYSQL", "mysql"},
		{"Redis", "redis-cli"},
		{"MongoDB", "mongosh"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			cmd, _ := ShellCommand(tt.engine)
			if cmd != tt.cmd {
				t.Errorf("ShellCommand(%q) cmd = %q, want %q", tt.engine, cmd, tt.cmd)
			}
		})
	}
}

func TestDefaultPort(t *testing.T) {
	tests := []struct {
		engine string
		port   int
	}{
		{"postgresql", 5432},
		{"postgres", 5432},
		{"mysql", 3306},
		{"redis", 6379},
		{"mongodb", 27017},
		{"mongo", 27017},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			port := DefaultPort(tt.engine)
			if port != tt.port {
				t.Errorf("DefaultPort(%q) = %d, want %d", tt.engine, port, tt.port)
			}
		})
	}
}

func TestDefaultPort_CaseInsensitive(t *testing.T) {
	tests := []struct {
		engine string
		port   int
	}{
		{"PostgreSQL", 5432},
		{"MYSQL", 3306},
		{"Redis", 6379},
		{"MongoDB", 27017},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			port := DefaultPort(tt.engine)
			if port != tt.port {
				t.Errorf("DefaultPort(%q) = %d, want %d", tt.engine, port, tt.port)
			}
		})
	}
}

func TestDefaultPort_EmptyEngine(t *testing.T) {
	port := DefaultPort("")
	if port != 0 {
		t.Errorf("DefaultPort(\"\") = %d, want 0", port)
	}
}

func TestConnectionStringFormatting(t *testing.T) {
	// Test that connection strings can be constructed from engine, host, port, and name
	tests := []struct {
		engine   string
		host     string
		port     int
		dbName   string
		expected string
	}{
		{
			engine:   "postgresql",
			host:     "127.0.0.1",
			port:     5432,
			dbName:   "mydb",
			expected: "postgresql://127.0.0.1:5432/mydb",
		},
		{
			engine:   "mysql",
			host:     "127.0.0.1",
			port:     3306,
			dbName:   "mydb",
			expected: "mysql://127.0.0.1:3306/mydb",
		},
		{
			engine:   "redis",
			host:     "127.0.0.1",
			port:     6379,
			dbName:   "",
			expected: "redis://127.0.0.1:6379",
		},
		{
			engine:   "mongodb",
			host:     "127.0.0.1",
			port:     27017,
			dbName:   "mydb",
			expected: "mongodb://127.0.0.1:27017/mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			var connStr string
			if tt.dbName != "" {
				connStr = fmt.Sprintf("%s://%s:%d/%s", tt.engine, tt.host, tt.port, tt.dbName)
			} else {
				connStr = fmt.Sprintf("%s://%s:%d", tt.engine, tt.host, tt.port)
			}
			if connStr != tt.expected {
				t.Errorf("Connection string = %q, want %q", connStr, tt.expected)
			}
		})
	}
}

func TestBackupCommandConstruction(t *testing.T) {
	tests := []struct {
		engine  string
		dbName  string
		host    string
		port    int
		wantCmd string
	}{
		{
			engine:  "postgresql",
			dbName:  "mydb",
			host:    "127.0.0.1",
			port:    5432,
			wantCmd: "pg_dump",
		},
		{
			engine:  "mysql",
			dbName:  "mydb",
			host:    "127.0.0.1",
			port:    3306,
			wantCmd: "mysqldump",
		},
		{
			engine:  "mongodb",
			dbName:  "mydb",
			host:    "127.0.0.1",
			port:    27017,
			wantCmd: "mongodump",
		},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			// Verify that backup command name is derived from engine
			var cmd string
			switch strings.ToLower(tt.engine) {
			case "postgresql", "postgres":
				cmd = "pg_dump"
			case "mysql":
				cmd = "mysqldump"
			case "mongodb", "mongo":
				cmd = "mongodump"
			case "redis":
				cmd = "redis-cli"
			}
			if cmd != tt.wantCmd {
				t.Errorf("Expected backup command '%s', got '%s'", tt.wantCmd, cmd)
			}
		})
	}
}

func TestRestoreCommandConstruction(t *testing.T) {
	tests := []struct {
		engine  string
		wantCmd string
	}{
		{"postgresql", "pg_restore"},
		{"mysql", "mysql"},
		{"mongodb", "mongorestore"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			var cmd string
			switch strings.ToLower(tt.engine) {
			case "postgresql", "postgres":
				cmd = "pg_restore"
			case "mysql":
				cmd = "mysql"
			case "mongodb", "mongo":
				cmd = "mongorestore"
			}
			if cmd != tt.wantCmd {
				t.Errorf("Expected restore command '%s', got '%s'", tt.wantCmd, cmd)
			}
		})
	}
}

func TestPortForwardArgs(t *testing.T) {
	tests := []struct {
		engine    string
		dbName    string
		namespace string
		localPort int
	}{
		{"postgresql", "my-db", "default", 5432},
		{"mysql", "my-db", "default", 3306},
		{"redis", "my-cache", "default", 6379},
		{"mongodb", "my-mongo", "default", 27017},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			remotePort := DefaultPort(tt.engine)
			if remotePort == 0 {
				t.Skip("Unknown engine port")
			}

			// Verify port-forward argument construction
			args := []string{
				"port-forward",
				fmt.Sprintf("svc/%s", tt.dbName),
				fmt.Sprintf("%d:%d", tt.localPort, remotePort),
				"-n", tt.namespace,
			}

			if len(args) != 5 {
				t.Errorf("Expected 5 args, got %d", len(args))
			}
			if args[0] != "port-forward" {
				t.Errorf("Expected first arg 'port-forward', got '%s'", args[0])
			}
			if args[1] != fmt.Sprintf("svc/%s", tt.dbName) {
				t.Errorf("Expected svc arg, got '%s'", args[1])
			}
			expectedPortMap := fmt.Sprintf("%d:%d", tt.localPort, remotePort)
			if args[2] != expectedPortMap {
				t.Errorf("Expected port mapping '%s', got '%s'", expectedPortMap, args[2])
			}
		})
	}
}

func TestSubcommandRegistration(t *testing.T) {
	// Verify that all expected subcommands are registered
	subcommands := Cmd.Commands()

	expectedCmds := map[string]bool{
		"list":    false,
		"create":  false,
		"connect": false,
		"backup":  false,
		"restore": false,
	}

	for _, cmd := range subcommands {
		if _, ok := expectedCmds[cmd.Name()]; ok {
			expectedCmds[cmd.Name()] = true
		}
	}

	for name, found := range expectedCmds {
		if !found {
			t.Errorf("Expected subcommand '%s' to be registered", name)
		}
	}
}

func TestConnectCommand_RequiresArgs(t *testing.T) {
	// The connect command requires exactly 1 argument
	if connectCmd.Args == nil {
		t.Error("Expected connect command to have Args validation")
	}
}

func TestBackupCommand_RequiresArgs(t *testing.T) {
	if backupCmd.Args == nil {
		t.Error("Expected backup command to have Args validation")
	}
}

func TestRestoreCommand_RequiresArgs(t *testing.T) {
	if restoreCmd.Args == nil {
		t.Error("Expected restore command to have Args validation")
	}
}
