package handlers_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// chattyLogs emits a line every 50ms until its ctx is cancelled, then signals.
type chattyLogs struct{ returned chan struct{} }

func (chattyLogs) Tail(context.Context, *entities.App, time.Duration, int) ([]dto.MonitoringLogEntry, error) {
	return nil, nil
}
func (l chattyLogs) Stream(ctx context.Context, _ *entities.App, out chan<- dto.MonitoringLogEntry) error {
	defer close(l.returned)
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			select {
			case out <- dto.MonitoringLogEntry{Line: "tick"}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func TestStreamLogs_ClientDisconnectStopsFollower(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	a, _ := repo.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "u1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	src := chattyLogs{returned: make(chan struct{})}
	svc := services.NewMonitoringService(nil, nil, nil, repo)
	svc.SetLogSource(src)
	h := handlers.NewMonitoringHandler(svc)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/apps/:appId/logs/stream", injectUserID("u1"), h.StreamLogs)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go app.Listener(ln)  //nolint:errcheck
	defer app.Shutdown() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://"+ln.Addr().String()+"/apps/"+a.ID+"/logs/stream", nil)
	resp, err := (&http.Client{Transport: &http.Transport{DisableKeepAlives: true}}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 64)
	if _, err := resp.Body.Read(buf); err != nil {
		t.Fatalf("no stream data: %v", err)
	}
	cancel() // client goes away
	resp.Body.Close()

	select {
	case <-src.returned:
	case <-time.After(3 * time.Second):
		t.Fatal("log follower still running after client disconnect")
	}
}
