package memory

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// Compile-time interface check.
var _ ports.MeteringRepository = (*MemoryMeteringRepository)(nil)

// MemoryMeteringRepository provides an in-memory store for metering data.
type MemoryMeteringRepository struct {
	mu    sync.RWMutex
	usage map[string][]entities.ResourceUsage // keyed by customerID
}

// NewMemoryMeteringRepository creates a MemoryMeteringRepository pre-seeded with
// 30 days of daily usage snapshots for the 3 demo customers.
func NewMemoryMeteringRepository() *MemoryMeteringRepository {
	r := &MemoryMeteringRepository{
		usage: make(map[string][]entities.ResourceUsage),
	}

	rng := rand.New(rand.NewSource(42))
	now := time.Now()

	// Customer profiles: id -> (cpuMin, cpuMax, ramMin, ramMax, dbStorage, volume, lbCount)
	type profile struct {
		id      string
		cpuMin  float64
		cpuMax  float64
		ramMin  float64
		ramMax  float64
		s3      float64
		db      float64
		vol     float64
		lbCount int
	}

	profiles := []profile{
		{id: "cust-001", cpuMin: 8, cpuMax: 12, ramMin: 18, ramMax: 24, s3: 0.3, db: 42, vol: 180, lbCount: 2},  // Embermind (Pro)
		{id: "cust-002", cpuMin: 5, cpuMax: 8, ramMin: 11, ramMax: 16, s3: 0.1, db: 25, vol: 95, lbCount: 1},    // Acme Corp (Pro)
		{id: "cust-003", cpuMin: 1.5, cpuMax: 3, ramMin: 5, ramMax: 7, s3: 0, db: 6, vol: 30, lbCount: 1},       // Starship IO (Starter)
	}

	for _, p := range profiles {
		entries := make([]entities.ResourceUsage, 0, 30)
		for d := 29; d >= 0; d-- {
			t := now.Add(-time.Duration(d) * 24 * time.Hour)
			cpu := p.cpuMin + rng.Float64()*(p.cpuMax-p.cpuMin)
			ram := p.ramMin + rng.Float64()*(p.ramMax-p.ramMin)
			entries = append(entries, entities.ResourceUsage{
				ID:          uuid.New().String(),
				CustomerID:  p.id,
				CPUCores:    math.Round(cpu*100) / 100,
				RAMGB:       math.Round(ram*100) / 100,
				S3TB:        math.Round(p.s3*100) / 100,
				DBStorageGB: math.Round(p.db*100) / 100,
				VolumeGB:    math.Round(p.vol*100) / 100,
				LBCount:     p.lbCount,
				RecordedAt:  t,
			})
		}
		r.usage[p.id] = entries
	}

	return r
}

func (r *MemoryMeteringRepository) RecordUsage(_ context.Context, input *dto.MeteringInput) (*entities.ResourceUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := entities.ResourceUsage{
		ID:          uuid.New().String(),
		CustomerID:  input.CustomerID,
		CPUCores:    input.CPUCores,
		RAMGB:       input.RAMGB,
		S3TB:        input.S3TB,
		DBStorageGB: input.DBStorageGB,
		VolumeGB:    input.VolumeGB,
		LBCount:     input.LBCount,
		RecordedAt:  time.Now(),
	}

	r.usage[input.CustomerID] = append(r.usage[input.CustomerID], entry)
	return &entry, nil
}

func (r *MemoryMeteringRepository) GetLatestUsage(_ context.Context, customerID string) (*entities.ResourceUsage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, ok := r.usage[customerID]
	if !ok || len(entries) == 0 {
		return nil, fmt.Errorf("no usage data found")
	}

	latest := entries[len(entries)-1]
	return &latest, nil
}

func (r *MemoryMeteringRepository) GetUsageHistory(_ context.Context, customerID string, days int) ([]dto.UsageHistoryEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := r.usage[customerID]
	if len(entries) == 0 {
		return []dto.UsageHistoryEntry{}, nil
	}

	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	// Group by date
	type dayBucket struct {
		cpuSum float64
		cpuMax float64
		ramSum float64
		ramMax float64
		db     float64
		vol    float64
		lb     int
		count  int
	}
	buckets := make(map[string]*dayBucket)

	for _, e := range entries {
		if e.RecordedAt.Before(cutoff) {
			continue
		}
		dateKey := e.RecordedAt.Format("2006-01-02")
		b, ok := buckets[dateKey]
		if !ok {
			b = &dayBucket{}
			buckets[dateKey] = b
		}
		b.cpuSum += e.CPUCores
		if e.CPUCores > b.cpuMax {
			b.cpuMax = e.CPUCores
		}
		b.ramSum += e.RAMGB
		if e.RAMGB > b.ramMax {
			b.ramMax = e.RAMGB
		}
		if e.DBStorageGB > b.db {
			b.db = e.DBStorageGB
		}
		if e.VolumeGB > b.vol {
			b.vol = e.VolumeGB
		}
		if e.LBCount > b.lb {
			b.lb = e.LBCount
		}
		b.count++
	}

	result := make([]dto.UsageHistoryEntry, 0, len(buckets))
	for date, b := range buckets {
		result = append(result, dto.UsageHistoryEntry{
			Date:        date,
			CPUAvg:      math.Round(b.cpuSum/float64(b.count)*100) / 100,
			CPUMax:      math.Round(b.cpuMax*100) / 100,
			RAMAvg:      math.Round(b.ramSum/float64(b.count)*100) / 100,
			RAMMax:      math.Round(b.ramMax*100) / 100,
			DBStorageGB: math.Round(b.db*100) / 100,
			VolumeGB:    math.Round(b.vol*100) / 100,
			LBCount:     b.lb,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })
	return result, nil
}

func (r *MemoryMeteringRepository) GetPlatformUsageSummary(_ context.Context) (*dto.PlatformUsageSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var totalCPU, totalRAM, totalStorage float64
	reporting := 0

	for _, entries := range r.usage {
		if len(entries) == 0 {
			continue
		}
		latest := entries[len(entries)-1]
		totalCPU += latest.CPUCores
		totalRAM += latest.RAMGB
		totalStorage += latest.DBStorageGB + latest.VolumeGB
		reporting++
	}

	return &dto.PlatformUsageSummary{
		TotalCPU:           math.Round(totalCPU*100) / 100,
		TotalRAM:           math.Round(totalRAM*100) / 100,
		TotalStorage:       math.Round(totalStorage*100) / 100,
		CustomersReporting: reporting,
	}, nil
}
