package deploy

import (
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// The redis image drops root in docker-entrypoint.sh; a bare `sh -c redis-server`
// bypasses it and runs the server as root.
func TestRedisSpecRunsThroughImageEntrypoint(t *testing.T) {
	spec, err := specFor(&entities.UserDatabase{Engine: entities.DatabaseEngineRedis}, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.cmd) != 3 || !strings.HasPrefix(spec.cmd[2], "exec docker-entrypoint.sh redis-server ") {
		t.Fatalf("redis cmd = %q, want exec docker-entrypoint.sh redis-server ...", spec.cmd)
	}
	if !strings.Contains(spec.cmd[2], `--requirepass "$REDIS_PASSWORD"`) || !strings.Contains(spec.cmd[2], "--appendonly yes") {
		t.Errorf("redis cmd lost its flags: %q", spec.cmd)
	}
}

func TestLoopbackBinding(t *testing.T) {
	exposed, bindings := loopbackBinding(5432)
	if _, ok := exposed["5432/tcp"]; !ok {
		t.Fatalf("exposed = %v", exposed)
	}
	b := bindings["5432/tcp"]
	if len(b) != 1 || b[0].HostIP != "127.0.0.1" || b[0].HostPort != "" {
		t.Fatalf("binding = %+v, want 127.0.0.1 with a docker-picked port", b)
	}
}
