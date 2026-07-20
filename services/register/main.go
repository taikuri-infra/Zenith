// Command register is the FreeZenith subdomain-registration service.
//
// It is the ONLY place the Cloudflare token lives. A customer installer POSTs
// its public IP; this service creates <slug>.apps.freezenith.com -> <ip> and
// returns just the hostname. The customer's box never sees the token.
//
// Security posture (see README): callers must present INSTALL_TOKEN; the target
// IP must be public; X-Forwarded-For is ignored unless the peer is a configured
// trusted proxy (CF-Connecting-IP is honored there). Proof-of-possession of the
// IP (an HTTP-01-style challenge) is still TODO and REQUIRED before this mints
// certs-worthy subdomains for fully untrusted callers at scale.
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultBase = "apps.freezenith.com"
	defaultZone = "freezenith.com"
)

func main() {
	token := os.Getenv("CLOUDFLARE_DNS_TOKEN")
	if token == "" {
		log.Fatal("CLOUDFLARE_DNS_TOKEN is required")
	}
	base := envOr("BASE_DOMAIN", defaultBase)
	zoneName := envOr("ZONE_NAME", defaultZone)
	port := envOr("PORT", "8080")
	installToken := os.Getenv("INSTALL_TOKEN")
	if installToken == "" {
		log.Println("WARNING: INSTALL_TOKEN is unset — /register and /release are DISABLED (fail closed). Set it before serving customers.")
	}
	trusted := parseCIDRs(os.Getenv("TRUSTED_PROXIES"))

	cf := &cfClient{token: token, http: &http.Client{Timeout: 15 * time.Second}}
	zoneID, err := cf.findZoneID(zoneName)
	if err != nil {
		log.Fatalf("resolve zone %q: %v", zoneName, err)
	}

	s := &server{cf: cf, zoneID: zoneID, base: base, installToken: installToken, trusted: trusted, lim: newLimiter(10, time.Hour)}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/register", s.register)
	mux.HandleFunc("/release", s.release)

	log.Printf("subdomain-registration service listening on :%s (base=%s, zone=%s, auth=%v)", port, base, zoneName, installToken != "")
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// ---- HTTP server ----------------------------------------------------------

type server struct {
	cf           *cfClient
	zoneID       string
	base         string
	installToken string
	trusted      []*net.IPNet
	lim          *limiter
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// authorized fails closed: no configured token means registration is disabled.
func (s *server) authorized(r *http.Request) bool {
	if s.installToken == "" {
		return false
	}
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.installToken)) == 1
}

func (s *server) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if !s.authorized(r) {
		writeErr(w, http.StatusUnauthorized, "missing or invalid install token")
		return
	}
	src := s.sourceIP(r)
	if !s.lim.allow(src) {
		writeErr(w, http.StatusTooManyRequests, "rate limit exceeded, try again later")
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	ip := strings.TrimSpace(req.IP)
	if ip == "" {
		ip = src
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || !isPublicIP(parsed) {
		writeErr(w, http.StatusBadRequest, "a valid, public IP is required")
		return
	}

	for i := 0; i < 8; i++ {
		host := generateSlug() + "." + s.base
		exists, err := s.cf.recordExists(s.zoneID, host)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "dns lookup failed")
			return
		}
		if exists {
			continue
		}
		if err := s.cf.createA(s.zoneID, host, ip); err != nil {
			writeErr(w, http.StatusBadGateway, "could not create dns record")
			return
		}
		log.Printf("registered %s -> %s (src %s)", host, ip, src)
		writeJSON(w, http.StatusOK, map[string]string{"hostname": host})
		return
	}
	writeErr(w, http.StatusConflict, "could not allocate a free subdomain")
}

func (s *server) release(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if !s.authorized(r) {
		writeErr(w, http.StatusUnauthorized, "missing or invalid install token")
		return
	}
	if !s.lim.allow(s.sourceIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "rate limit exceeded, try again later")
		return
	}
	var req struct {
		Hostname string `json:"hostname"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	host := strings.TrimSpace(strings.ToLower(req.Hostname))
	// Only ever delete a well-formed subdomain inside the base we operate.
	if host == "" || !strings.HasSuffix(host, "."+s.base) || !validHostname(host) {
		writeErr(w, http.StatusBadRequest, "hostname must be a "+s.base+" subdomain")
		return
	}
	if err := s.cf.deleteA(s.zoneID, host); err != nil {
		writeErr(w, http.StatusBadGateway, "could not delete dns record")
		return
	}
	log.Printf("released %s", host)
	writeJSON(w, http.StatusOK, map[string]string{"status": "released"})
}

// sourceIP returns the caller IP. X-Forwarded-For is inherently spoofable, so it
// is ignored; CF-Connecting-IP is honored ONLY when the immediate peer is a
// configured trusted proxy. Otherwise the TCP peer address is used.
func (s *server) sourceIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer != nil && ipInAny(peer, s.trusted) {
		if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" && net.ParseIP(cf) != nil {
			return cf
		}
	}
	return host
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// ---- validation helpers ---------------------------------------------------

// isPublicIP rejects loopback, private, link-local, multicast, and unspecified
// addresses — a subdomain must never point at an internal target (SSRF/phishing).
func isPublicIP(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified())
}

// validHostname allows only DNS-safe characters, so a hostname can never inject
// into the Cloudflare API query string.
func validHostname(h string) bool {
	for _, r := range h {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.'
		if !ok {
			return false
		}
	}
	return true
}

func parseCIDRs(s string) []*net.IPNet {
	var out []*net.IPNet
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, n, err := net.ParseCIDR(part); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func ipInAny(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ---- slug -----------------------------------------------------------------

var slugAdjectives = []string{"amber", "brave", "clever", "dawn", "eager", "fair", "gentle", "happy", "ivory", "jolly", "keen", "lively", "misty", "noble", "olive", "proud", "quiet", "rapid", "swift", "teal", "urban", "vivid", "warm", "zesty"}
var slugAnimals = []string{"falcon", "otter", "lynx", "heron", "bison", "koala", "raven", "gecko", "panda", "tiger", "moose", "crane", "ibex", "puma", "wren", "seal", "fox", "hare", "owl", "stag"}

func generateSlug() string {
	const hex = "0123456789abcdef"
	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = hex[randInt(len(hex))]
	}
	return fmt.Sprintf("%s-%s-%s", slugAdjectives[randInt(len(slugAdjectives))], slugAnimals[randInt(len(slugAnimals))], string(suffix))
}

func randInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

// ---- rate limiter (per source IP, fixed window) ---------------------------

type limiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{hits: map[string][]time.Time{}, max: max, window: window}
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.max {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}

// ---- minimal Cloudflare client --------------------------------------------

type cfClient struct {
	token string
	http  *http.Client
}

type cfResp struct {
	Success bool              `json:"success"`
	Errors  []json.RawMessage `json:"errors"`
	Result  json.RawMessage   `json:"result"`
}

func (c *cfClient) do(method, path string, body, out any) error {
	var rdr *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = strings.NewReader(string(b))
	} else {
		rdr = strings.NewReader("")
	}
	req, err := http.NewRequest(method, "https://api.cloudflare.com/client/v4"+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var cr cfResp
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return err
	}
	if !cr.Success {
		return fmt.Errorf("cloudflare error: %s", string(join(cr.Errors)))
	}
	if out != nil {
		return json.Unmarshal(cr.Result, out)
	}
	return nil
}

func join(msgs []json.RawMessage) []byte {
	parts := make([]string, len(msgs))
	for i, m := range msgs {
		parts[i] = string(m)
	}
	return []byte(strings.Join(parts, "; "))
}

func (c *cfClient) findZoneID(name string) (string, error) {
	var zones []struct {
		ID string `json:"id"`
	}
	if err := c.do("GET", "/zones?name="+url.QueryEscape(name), nil, &zones); err != nil {
		return "", err
	}
	if len(zones) == 0 {
		return "", fmt.Errorf("no zone named %q", name)
	}
	return zones[0].ID, nil
}

type dnsRecord struct {
	ID string `json:"id"`
}

func (c *cfClient) findRecord(zoneID, name string) (*dnsRecord, error) {
	var recs []dnsRecord
	q := fmt.Sprintf("/zones/%s/dns_records?type=A&name=%s", url.PathEscape(zoneID), url.QueryEscape(name))
	if err := c.do("GET", q, nil, &recs); err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return &recs[0], nil
}

func (c *cfClient) recordExists(zoneID, name string) (bool, error) {
	rec, err := c.findRecord(zoneID, name)
	return rec != nil, err
}

func (c *cfClient) createA(zoneID, name, ip string) error {
	return c.do("POST", "/zones/"+url.PathEscape(zoneID)+"/dns_records", map[string]any{
		"type": "A", "name": name, "content": ip, "ttl": 120, "proxied": false,
	}, nil)
}

func (c *cfClient) deleteA(zoneID, name string) error {
	rec, err := c.findRecord(zoneID, name)
	if err != nil {
		return err
	}
	if rec == nil {
		return nil
	}
	return c.do("DELETE", "/zones/"+url.PathEscape(zoneID)+"/dns_records/"+url.PathEscape(rec.ID), nil, nil)
}
