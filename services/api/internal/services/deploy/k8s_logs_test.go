package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestK8sLogSource_Tail(t *testing.T) {
	c := k8sclient.NewMemoryClient()
	app := &entities.App{ID: "app-1", Name: "web"}
	c.SetPodLogs("zenith.dev/app-id=app-1", "2026-10-25T10:00:00.5Z hello\n2026-10-25T10:00:01Z world\nno-timestamp line\n")
	src := NewK8sLogSource(c, "zenith-apps")

	got, err := src.Tail(context.Background(), app, time.Hour, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Line != "hello" || got[1].Line != "world" || got[2].Line != "no-timestamp line" {
		t.Fatalf("entries = %+v", got)
	}
	if got[0].Timestamp.Year() != 2026 {
		t.Fatalf("timestamp not parsed: %v", got[0].Timestamp)
	}
}

func TestK8sLogSource_NoPodsIsEmptyNotError(t *testing.T) {
	src := NewK8sLogSource(k8sclient.NewMemoryClient(), "zenith-apps")
	got, err := src.Tail(context.Background(), &entities.App{ID: "gone"}, time.Hour, 10)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestK8sLogSource_Stream(t *testing.T) {
	c := k8sclient.NewMemoryClient()
	c.SetPodLogs("zenith.dev/app-id=a", "2026-10-25T10:00:00Z one\n2026-10-25T10:00:01Z two\n")
	out := make(chan dto.MonitoringLogEntry, 4)
	if err := NewK8sLogSource(c, "zenith-apps").Stream(context.Background(), &entities.App{ID: "a"}, out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("streamed %d entries", len(out))
	}
}
