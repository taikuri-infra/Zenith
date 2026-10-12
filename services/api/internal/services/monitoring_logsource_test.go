package services

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type fakeLogs struct{}

func (fakeLogs) Tail(_ context.Context, _ *entities.App, _ time.Duration, _ int) ([]dto.MonitoringLogEntry, error) {
	return []dto.MonitoringLogEntry{{Line: "GET / 200"}, {Line: "ERROR boom"}, {Line: "GET /health 200"}}, nil
}
func (fakeLogs) Stream(_ context.Context, _ *entities.App, out chan<- dto.MonitoringLogEntry) error {
	out <- dto.MonitoringLogEntry{Line: "live"}
	return nil
}

func TestMonitoring_UsesLogSourceWithSearch(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	a, _ := repo.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "u1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	s := NewMonitoringService(nil, nil, nil, repo)
	s.SetLogSource(fakeLogs{})

	all, err := s.GetLogs(context.Background(), "u1", a.ID, "", "", 100, time.Hour)
	if err != nil || all.Total != 3 {
		t.Fatalf("all: %+v %v", all, err)
	}
	f, _ := s.GetLogs(context.Background(), "u1", a.ID, "", "boom", 100, time.Hour)
	if f.Total != 1 || f.Entries[0].Line != "ERROR boom" {
		t.Errorf("search: %+v", f)
	}
	ch := make(chan dto.MonitoringLogEntry, 1)
	if err := s.StreamLogs(context.Background(), "u1", a.ID, ch); err != nil {
		t.Fatal(err)
	}
	if (<-ch).Line != "live" {
		t.Error("stream did not use the log source")
	}
}
