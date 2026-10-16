// services/api/internal/services/domain_checker.go
package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// DomainResolver is the DNS lookup the checker uses; *net.Resolver satisfies it.
type DomainResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// TLSProber reports whether the certificate served at addr for serverName verifies for serverName.
// (false, nil) means the handshake worked but the chain does not verify (Traefik's default certificate).
type TLSProber interface {
	Probe(ctx context.Context, addr, serverName string) (valid bool, err error)
}

// DomainChecker verifies custom domains: DNS must point at this server, then Traefik must serve a valid certificate.
type DomainChecker struct {
	repo            ports.DomainRepository
	resolver        DomainResolver
	prober          TLSProber
	dashboardDomain string
	traefikAddr     string
	stepTimeout     time.Duration
	now             func() time.Time

	refresher RouteRefresher

	mu         sync.Mutex
	refreshSt  map[string]*refreshState // per domain id
	refreshing map[string]bool          // per app id: a refresh is running
	expected   []net.IP
	expectedAt time.Time
}

// refreshState is one domain's route-refresh schedule while it is verified.
// refreshes == 0 means a refresh is due on the next verified check (entering verified, or ResetRefresh).
type refreshState struct {
	refreshes   int
	lastRefresh time.Time
}

// RouteRefresher redeploys an app's routes. Traefik requests an ACME certificate only when its
// dynamic config changes, so a refresh is what makes it try (and retry) once DNS points here.
type RouteRefresher interface {
	RefreshRoutes(ctx context.Context, appID string) error
}

// refreshGaps are the waits after refresh 1, 2 and 3; after the 4th refresh the checker stops.
var refreshGaps = []time.Duration{10 * time.Minute, 30 * time.Minute, 2 * time.Hour}

const (
	maxRefreshes   = 4
	refreshTimeout = 2 * time.Minute
	cappedMessage  = "DNS ok; certificate not issued after 4 route refreshes; check the Traefik logs or press Check now"
)

const (
	checkEvery        = 30 * time.Second
	checkEveryAfter1h = 5 * time.Minute
	checkEveryAfter1d = 30 * time.Minute
	recheckActive     = 10 * time.Minute
	expectedIPsTTL    = 10 * time.Minute
)

func NewDomainChecker(repo ports.DomainRepository, resolver DomainResolver, prober TLSProber, dashboardDomain string) *DomainChecker {
	return &DomainChecker{repo: repo, resolver: resolver, prober: prober, dashboardDomain: dashboardDomain,
		traefikAddr: "traefik:443", stepTimeout: 5 * time.Second, now: time.Now,
		refreshSt: map[string]*refreshState{}, refreshing: map[string]bool{}}
}

// ResetRefresh restarts a domain's refresh schedule, so its next verified check refreshes routes at once.
func (c *DomainChecker) ResetRefresh(id string) {
	c.mu.Lock()
	c.refreshSt[id] = &refreshState{}
	c.mu.Unlock()
}

// SetRefresher enables route refreshes when a domain is verified.
func (c *DomainChecker) SetRefresher(r RouteRefresher) { c.refresher = r }

// MarkPending sets a freshly added domain to pending with the initial message.
func (c *DomainChecker) MarkPending(ctx context.Context, id string) error {
	return c.repo.UpdateDomainCheck(ctx, id, entities.DomainStatusPending, false, "waiting for DNS check", time.Time{})
}

// Run checks due domains every 30 s until ctx ends.
func (c *DomainChecker) Run(ctx context.Context) {
	t := time.NewTicker(checkEvery)
	defer t.Stop()
	for {
		c.CheckAll(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// CheckAll runs one round over every due domain. Errors are logged, never fatal.
func (c *DomainChecker) CheckAll(ctx context.Context) {
	if _, err := c.expectedIPs(ctx); err != nil {
		slog.Warn("domain check: dashboard domain does not resolve; skipping round", "domain", c.dashboardDomain, "error", err)
		return
	}
	domains, err := c.repo.ListAllDomains(ctx)
	if err != nil {
		slog.Error("domain check: list", "error", err)
		return
	}
	for i := range domains {
		d := &domains[i]
		if !c.due(d) {
			continue
		}
		if _, err := c.Check(ctx, d); err != nil {
			slog.Warn("domain check failed", "domain", d.Domain, "error", err)
		}
	}
}

// due applies the backoff: 30 s for the first hour, 5 min for the first day, then 30 min; active domains every 10 min.
func (c *DomainChecker) due(d *entities.CustomDomain) bool {
	now := c.now()
	if d.LastCheckedAt == nil {
		return true
	}
	since := now.Sub(*d.LastCheckedAt)
	if d.Status == entities.DomainStatusActive && d.TLSReady {
		return since >= recheckActive
	}
	age := now.Sub(d.CreatedAt)
	switch {
	case age >= 24*time.Hour:
		return since >= checkEveryAfter1d
	case age >= time.Hour:
		return since >= checkEveryAfter1h
	default:
		return since >= checkEvery
	}
}

// Check runs the DNS and TLS steps for one domain and stores the outcome.
func (c *DomainChecker) Check(ctx context.Context, d *entities.CustomDomain) (*entities.CustomDomain, error) {
	expected, err := c.expectedIPs(ctx)
	if err != nil {
		return d, fmt.Errorf("resolve dashboard domain %s: %w", c.dashboardDomain, err)
	}
	status, tlsReady, msg := c.evaluate(ctx, d, expected)
	now := c.now()
	refresh := false
	if status == entities.DomainStatusVerified {
		var capped bool
		refresh, capped = c.planRefresh(d, now)
		if capped {
			msg = cappedMessage
		}
	} else { // leaving verified (or never in it) resets the schedule
		c.mu.Lock()
		delete(c.refreshSt, d.ID)
		c.mu.Unlock()
	}
	if err := c.repo.UpdateDomainCheck(ctx, d.ID, status, tlsReady, msg, now); err != nil {
		if refresh {
			c.mu.Lock()
			c.refreshing[d.AppID] = false
			c.mu.Unlock()
		}
		return d, err
	}
	out := *d
	out.Status, out.TLSReady, out.StatusMessage, out.LastCheckedAt = status, tlsReady, msg, &now
	if refresh {
		go c.refresh(d.AppID, d.ID)
	}
	return &out, nil
}

// planRefresh advances the domain's refresh schedule for a verified result: refresh on entering verified,
// then 10 min, 30 min and 2 h later; after the 4th refresh it reports capped instead. A domain first seen
// already verified (after an API restart) is seeded at refresh 1 without refreshing. When refresh is
// true the app is marked in flight; the caller must run c.refresh or clear the flag.
func (c *DomainChecker) planRefresh(prev *entities.CustomDomain, now time.Time) (refresh, capped bool) {
	if c.refresher == nil {
		return false, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	st, seen := c.refreshSt[prev.ID]
	switch {
	case !seen && prev.Status == entities.DomainStatusVerified:
		c.refreshSt[prev.ID] = &refreshState{refreshes: 1, lastRefresh: now}
		return false, false
	case !seen || prev.Status != entities.DomainStatusVerified:
		st = &refreshState{}
		c.refreshSt[prev.ID] = st
	}
	if st.refreshes >= maxRefreshes {
		return false, true
	}
	if st.refreshes > 0 && now.Sub(st.lastRefresh) < refreshGaps[st.refreshes-1] {
		return false, false
	}
	if c.refreshing[prev.AppID] {
		return false, false // another domain of this app is refreshing it; try again on the next check
	}
	st.refreshes++
	st.lastRefresh = now
	c.refreshing[prev.AppID] = true
	return true, false
}

// refresh redeploys the app's routes outside the request path and clears the app's in-flight flag.
func (c *DomainChecker) refresh(appID, domainID string) {
	defer func() {
		c.mu.Lock()
		c.refreshing[appID] = false
		c.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
	defer cancel()
	if err := c.refresher.RefreshRoutes(ctx, appID); err != nil {
		slog.Warn("domain check: route refresh failed", "app", appID, "domain_id", domainID, "error", err)
	}
}

func (c *DomainChecker) evaluate(ctx context.Context, d *entities.CustomDomain, expected []net.IP) (entities.DomainStatus, bool, string) {
	domain := d.Domain
	dctx, cancel := context.WithTimeout(ctx, c.stepTimeout)
	addrs, err := c.resolver.LookupIPAddr(dctx, domain)
	cancel()
	var dnsErr *net.DNSError
	if err != nil && !(errors.As(err, &dnsErr) && dnsErr.IsNotFound) {
		// Transient resolver trouble must not demote a domain: keep its state, report the problem.
		return d.Status, d.TLSReady, "DNS check failed: " + err.Error()
	}
	if err != nil || len(addrs) == 0 {
		return entities.DomainStatusPending, false,
			fmt.Sprintf("DNS: no A/AAAA record for %s; create an A record to %s", domain, expected[0])
	}
	if !intersects(addrs, expected) {
		return entities.DomainStatusPending, false,
			fmt.Sprintf("DNS: %s points to %s; expected %s", domain, joinIPs(addrs), expected[0])
	}
	tctx, cancel := context.WithTimeout(ctx, c.stepTimeout)
	valid, perr := c.prober.Probe(tctx, c.traefikAddr, domain)
	cancel()
	switch {
	case perr != nil:
		return entities.DomainStatusVerified, false, "TLS check failed: " + perr.Error()
	case !valid:
		return entities.DomainStatusVerified, false, "DNS ok; waiting for the certificate"
	default:
		return entities.DomainStatusActive, true, ""
	}
}

// expectedIPs resolves the dashboard domain (cached 10 min). Its addresses are this server's public addresses.
func (c *DomainChecker) expectedIPs(ctx context.Context) ([]net.IP, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.expected) > 0 && c.now().Sub(c.expectedAt) < expectedIPsTTL {
		return c.expected, nil
	}
	rctx, cancel := context.WithTimeout(ctx, c.stepTimeout)
	defer cancel()
	addrs, err := c.resolver.LookupIPAddr(rctx, c.dashboardDomain)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no address")
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		ips = append(ips, a.IP)
	}
	sort.SliceStable(ips, func(i, j int) bool { // IPv4 first so messages show the A record
		return ips[i].To4() != nil && ips[j].To4() == nil
	})
	c.expected, c.expectedAt = ips, c.now()
	return ips, nil
}

func intersects(addrs []net.IPAddr, expected []net.IP) bool {
	for _, a := range addrs {
		for _, e := range expected {
			if a.IP.Equal(e) {
				return true
			}
		}
	}
	return false
}

func joinIPs(addrs []net.IPAddr) string {
	s := make([]string, 0, len(addrs))
	for _, a := range addrs {
		s = append(s, a.IP.String())
	}
	return strings.Join(s, ", ")
}

// TraefikTLSProber dials Traefik with SNI and full chain verification for the domain.
// Roots nil means the system roots (Let's Encrypt). The probe only observes the certificate Traefik serves;
// getting Traefik to request one is the route refresh's job.
// A handshake that fails only because the chain does not verify (Traefik's default certificate
// while ACME is pending) is reported as (false, nil); any other failure is an error.
type TraefikTLSProber struct{ Roots *x509.CertPool }

func (p *TraefikTLSProber) Probe(ctx context.Context, addr, serverName string) (bool, error) {
	d := &tls.Dialer{Config: &tls.Config{ServerName: serverName, RootCAs: p.Roots, MinVersion: tls.VersionTLS12}}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err == nil {
		conn.Close()
		return true, nil
	}
	var certErr *tls.CertificateVerificationError
	var unknownCA x509.UnknownAuthorityError
	var hostErr x509.HostnameError
	var invalid x509.CertificateInvalidError
	if errors.As(err, &certErr) || errors.As(err, &unknownCA) || errors.As(err, &hostErr) || errors.As(err, &invalid) {
		return false, nil
	}
	return false, err
}
