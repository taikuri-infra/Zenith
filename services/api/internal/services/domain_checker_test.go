// services/api/internal/services/domain_checker_test.go
package services

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type fakeResolver struct{ ips map[string][]string }

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := f.ips[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	out := make([]net.IPAddr, 0, len(ips))
	for _, s := range ips {
		out = append(out, net.IPAddr{IP: net.ParseIP(s)})
	}
	return out, nil
}

type fakeProber struct {
	valid bool
	err   error
	calls int
}

func (p *fakeProber) Probe(_ context.Context, addr, serverName string) (bool, error) {
	p.calls++
	return p.valid, p.err
}

func newCheckerForTest(resolver fakeResolver, prober *fakeProber) (*DomainChecker, *memory.MemoryDomainRepository, *entities.CustomDomain) {
	repo := memory.NewMemoryDomainRepository()
	d, _ := repo.AddDomain(context.Background(), "app-1", "user-1", "app.example.com")
	c := NewDomainChecker(repo, resolver, prober, "zenith.example.com")
	c.now = func() time.Time { return time.Date(2026, 10, 16, 19, 0, 0, 0, time.UTC) }
	_ = c.MarkPending(context.Background(), d.ID)
	d, _ = repo.GetDomain(context.Background(), d.ID)
	return c, repo, d
}

func TestDomainChecker_NoRecordStaysPendingWithIP(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}}}, &fakeProber{})
	got, err := c.Check(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != entities.DomainStatusPending || got.TLSReady {
		t.Fatalf("got %s/%v", got.Status, got.TLSReady)
	}
	if got.StatusMessage != "DNS: no A/AAAA record for app.example.com; create an A record to 1.2.3.4" {
		t.Fatalf("message = %q", got.StatusMessage)
	}
	if got.LastCheckedAt == nil {
		t.Fatal("LastCheckedAt not set")
	}
}

func TestDomainChecker_WrongIPStaysPending(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"9.9.9.9"}}}, &fakeProber{})
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusPending || got.StatusMessage != "DNS: app.example.com points to 9.9.9.9; expected 1.2.3.4" {
		t.Fatalf("got %s %q", got.Status, got.StatusMessage)
	}
}

func TestDomainChecker_DNSOkCertPendingIsVerified(t *testing.T) {
	p := &fakeProber{valid: false}
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}, p)
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusVerified || got.TLSReady || got.StatusMessage != "DNS ok; waiting for the certificate" {
		t.Fatalf("got %s/%v %q", got.Status, got.TLSReady, got.StatusMessage)
	}
	if p.calls != 1 {
		t.Fatalf("prober calls = %d", p.calls)
	}
}

func TestDomainChecker_ValidCertIsActive(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4", "2001:db8::1"}, "app.example.com": {"1.2.3.4"}}}, &fakeProber{valid: true})
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusActive || !got.TLSReady || got.StatusMessage != "" {
		t.Fatalf("got %s/%v %q", got.Status, got.TLSReady, got.StatusMessage)
	}
}

func TestDomainChecker_TLSErrorKeepsVerifiedWithMessage(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}, &fakeProber{err: errors.New("dial tcp: connection refused")})
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusVerified || !strings.HasPrefix(got.StatusMessage, "TLS check failed: ") {
		t.Fatalf("got %s %q", got.Status, got.StatusMessage)
	}
}

func TestDomainChecker_ActiveGoesBackToPendingWhenDNSMoves(t *testing.T) {
	r := fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}
	c, repo, d := newCheckerForTest(r, &fakeProber{valid: true})
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusActive {
		t.Fatal("precondition: active")
	}
	r.ips["app.example.com"] = []string{"5.5.5.5"}
	got, _ = c.Check(context.Background(), got)
	if got.Status != entities.DomainStatusPending || got.TLSReady {
		t.Fatalf("got %s/%v, want pending/false", got.Status, got.TLSReady)
	}
	stored, _ := repo.GetDomain(context.Background(), d.ID)
	if stored.Status != entities.DomainStatusPending {
		t.Fatal("repo not updated")
	}
}

func TestDomainChecker_DashboardUnresolvableSkipsRound(t *testing.T) {
	p := &fakeProber{valid: true}
	c, repo, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"app.example.com": {"1.2.3.4"}}}, p)
	if !c.due(d) {
		t.Fatal("precondition: never-checked domain must be due")
	}
	c.CheckAll(context.Background())
	stored, _ := repo.GetDomain(context.Background(), d.ID)
	if p.calls != 0 || stored.Status != entities.DomainStatusPending || stored.StatusMessage != "waiting for DNS check" || stored.LastCheckedAt != nil {
		t.Fatalf("round must be skipped when the dashboard domain does not resolve: calls=%d %+v", p.calls, stored)
	}
}

type recRefresher struct{ apps []string }

func (r *recRefresher) RefreshRoutes(_ context.Context, appID string) error {
	r.apps = append(r.apps, appID)
	return nil
}

func TestDomainChecker_PendingToVerifiedRefreshesRoutes(t *testing.T) {
	r := fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}}}
	c, _, d := newCheckerForTest(r, &fakeProber{valid: false})
	rf := &recRefresher{}
	c.SetRefresher(rf)
	got, _ := c.Check(context.Background(), d)
	if len(rf.apps) != 0 {
		t.Fatalf("pending must not refresh: %v", rf.apps)
	}
	r.ips["app.example.com"] = []string{"1.2.3.4"}
	got, _ = c.Check(context.Background(), got)
	if got.Status != entities.DomainStatusVerified || len(rf.apps) != 1 || rf.apps[0] != "app-1" {
		t.Fatalf("status %s refreshes %v", got.Status, rf.apps)
	}
	got, _ = c.Check(context.Background(), got)
	if len(rf.apps) != 1 {
		t.Fatalf("no refresh within 10 min: %v", rf.apps)
	}
	base := c.now()
	c.now = func() time.Time { return base.Add(11 * time.Minute) }
	_, _ = c.Check(context.Background(), got)
	if len(rf.apps) != 2 {
		t.Fatalf("refresh after 11 min expected: %v", rf.apps)
	}
}

func TestDomainChecker_ActiveDoesNotRefresh(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}, &fakeProber{valid: true})
	rf := &recRefresher{}
	c.SetRefresher(rf)
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusActive || len(rf.apps) != 0 {
		t.Fatalf("%s %v", got.Status, rf.apps)
	}
}

type errResolver struct {
	fakeResolver
	err error
}

func (e errResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	if host == "app.example.com" {
		return nil, e.err
	}
	return e.fakeResolver.LookupIPAddr(ctx, host)
}

func TestDomainChecker_ResolverErrorKeepsActive(t *testing.T) {
	ok := fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}
	c, repo, d := newCheckerForTest(ok, &fakeProber{valid: true})
	got, _ := c.Check(context.Background(), d)
	c.resolver = errResolver{ok, errors.New("i/o timeout")}
	got, err := c.Check(context.Background(), got)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != entities.DomainStatusActive || !got.TLSReady || got.StatusMessage != "DNS check failed: i/o timeout" {
		t.Fatalf("got %s/%v %q", got.Status, got.TLSReady, got.StatusMessage)
	}
	stored, _ := repo.GetDomain(context.Background(), d.ID)
	if stored.Status != entities.DomainStatusActive || !stored.TLSReady || stored.LastCheckedAt == nil {
		t.Fatalf("stored %+v", stored)
	}
}

func TestDomainChecker_IPv6FirstStillShowsIPv4(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"2001:db8::1", "1.2.3.4"}}}, &fakeProber{})
	got, _ := c.Check(context.Background(), d)
	if got.StatusMessage != "DNS: no A/AAAA record for app.example.com; create an A record to 1.2.3.4" {
		t.Fatalf("message = %q", got.StatusMessage)
	}
}

type countResolver struct {
	fakeResolver
	dashCalls int
}

func (r *countResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	if host == "zenith.example.com" {
		r.dashCalls++
	}
	return r.fakeResolver.LookupIPAddr(ctx, host)
}

func TestDomainChecker_ExpectedIPsCached(t *testing.T) {
	cr := &countResolver{fakeResolver: fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}}}}
	c, _, d := newCheckerForTest(fakeResolver{}, &fakeProber{})
	c.resolver = cr
	_, _ = c.Check(context.Background(), d)
	_, _ = c.Check(context.Background(), d)
	if cr.dashCalls != 1 {
		t.Fatalf("dashboard lookups = %d, want 1", cr.dashCalls)
	}
	base := c.now()
	c.now = func() time.Time { return base.Add(11 * time.Minute) }
	_, _ = c.Check(context.Background(), d)
	if cr.dashCalls != 2 {
		t.Fatalf("dashboard lookups = %d, want 2", cr.dashCalls)
	}
}

func TestDomainChecker_Due(t *testing.T) {
	c := NewDomainChecker(memory.NewMemoryDomainRepository(), fakeResolver{}, &fakeProber{}, "zenith.example.com")
	now := time.Date(2026, 10, 16, 19, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	mk := func(status entities.DomainStatus, tls bool, age, sinceCheck time.Duration) *entities.CustomDomain {
		d := &entities.CustomDomain{Status: status, TLSReady: tls}
		d.CreatedAt = now.Add(-age)
		if sinceCheck >= 0 {
			t := now.Add(-sinceCheck)
			d.LastCheckedAt = &t
		}
		return d
	}
	cases := []struct {
		name string
		d    *entities.CustomDomain
		want bool
	}{
		{"never checked", mk(entities.DomainStatusPending, false, time.Minute, -1), true},
		{"young, checked 20s ago", mk(entities.DomainStatusPending, false, time.Minute, 20*time.Second), false},
		{"young, checked 40s ago", mk(entities.DomainStatusPending, false, time.Minute, 40*time.Second), true},
		{"2h old, checked 2m ago", mk(entities.DomainStatusPending, false, 2*time.Hour, 2*time.Minute), false},
		{"2h old, checked 6m ago", mk(entities.DomainStatusPending, false, 2*time.Hour, 6*time.Minute), true},
		{"2d old, checked 20m ago", mk(entities.DomainStatusVerified, false, 48*time.Hour, 20*time.Minute), false},
		{"2d old, checked 40m ago", mk(entities.DomainStatusVerified, false, 48*time.Hour, 40*time.Minute), true},
		{"active, checked 5m ago", mk(entities.DomainStatusActive, true, time.Minute, 5*time.Minute), false},
		{"active, checked 11m ago", mk(entities.DomainStatusActive, true, time.Minute, 11*time.Minute), true},
	}
	for _, tc := range cases {
		if got := c.due(tc.d); got != tc.want {
			t.Errorf("%s: due = %v, want %v", tc.name, got, tc.want)
		}
	}
}
