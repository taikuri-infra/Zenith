//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	cliapi "github.com/dotechhq/zenith/cli/internal/api"
)

// Bounded waits (spec §2 and the task brief).
const (
	pollEvery      = 5 * time.Second
	appRunningWait = 300 * time.Second
	dbReadyWait    = 300 * time.Second
	envInjectWait  = 180 * time.Second
	domainWait     = 180 * time.Second
	rebootCmdWait  = 10 * time.Minute
	healthWait     = 300 * time.Second
	upgradeWait    = 15 * time.Minute
	backupWait     = 15 * time.Minute
	restoreWait    = 20 * time.Minute
	zenWait        = 5 * time.Minute
	copyWait       = 5 * time.Minute

	probeAppName = "probe"
	probeDBName  = "pg"
	probePort    = 8080
)

var (
	cfg     Config
	zenHome string // HOME for the zen binary: zen login writes a throwaway config here
	logMu   sync.Mutex
	logFile *os.File

	httpOnce sync.Once
	httpC    *http.Client
	httpErr  error

	adminMu  sync.Mutex
	adminAPI *cliapi.Client
)

func TestMain(m *testing.M) {
	flag.Parse()
	if f := flag.Lookup("test.list"); f != nil && f.Value.String() != "" {
		os.Exit(m.Run()) // -list only prints names; it needs no server
	}
	c, err := Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: bad configuration:", err)
		os.Exit(2)
	}
	cfg = c
	if cfg.Artifacts == "" {
		if cfg.Artifacts, err = os.MkdirTemp("", "zenith-e2e-artifacts-"); err != nil {
			fmt.Fprintln(os.Stderr, "e2e:", err)
			os.Exit(2)
		}
	}
	if err := os.MkdirAll(cfg.Artifacts, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "e2e: artifacts dir:", err)
		os.Exit(2)
	}
	if logFile, err = os.OpenFile(filepath.Join(cfg.Artifacts, "suite.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "e2e: suite log:", err)
		os.Exit(2)
	}
	if _, err := httpClient(); err != nil {
		fmt.Fprintln(os.Stderr, "e2e:", err)
		os.Exit(2)
	}
	if zenHome, err = os.MkdirTemp("", "zenith-e2e-home-"); err != nil {
		fmt.Fprintln(os.Stderr, "e2e:", err)
		os.Exit(2)
	}
	writeLog("suite", fmt.Sprintf("endpoint=%s domain=%s zen=%s ca=%q candidate=%q artifacts=%s",
		cfg.Endpoint, cfg.Domain, cfg.Zen, cfg.CAFile, cfg.CandidateVersion, cfg.Artifacts))
	code := m.Run()
	_ = os.RemoveAll(zenHome)
	_ = logFile.Close()
	os.Exit(code)
}

func writeLog(who, msg string) {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		fmt.Fprintf(logFile, "%s [%s] %s\n", time.Now().UTC().Format(time.RFC3339), who, msg)
	}
}

// logf writes to the test log and to <Artifacts>/suite.log.
func logf(t *testing.T, format string, args ...any) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	t.Log(msg)
	writeLog(t.Name(), msg)
}

// tail keeps the end of long outputs readable in the test log (suite.log keeps everything).
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

// sslipIP returns the IPv4 an sslip.io name resolves to ("1.2.3.4.sslip.io" or "x.1.2.3.4.sslip.io"), or "".
// The suite's own HTTP client dials that IP directly (the answer public DNS would give), so its requests to
// dotted sslip.io names do not need DNS. Only those: the zen binary, the hooks, and the dashed form
// (1-2-3-4.sslip.io) still resolve through DNS.
func sslipIP(host string) string {
	h := strings.TrimSuffix(strings.ToLower(host), ".")
	if !strings.HasSuffix(h, ".sslip.io") {
		return ""
	}
	labels := strings.Split(strings.TrimSuffix(h, ".sslip.io"), ".")
	if len(labels) < 4 {
		return ""
	}
	ip := net.ParseIP(strings.Join(labels[len(labels)-4:], "."))
	if ip == nil || ip.To4() == nil {
		return ""
	}
	return ip.String()
}

// httpClient trusts the system roots plus cfg.CAFile, so every HTTPS call also checks the chain against the install CA.
func httpClient() (*http.Client, error) {
	httpOnce.Do(func() {
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if cfg.CAFile != "" {
			pem, err := os.ReadFile(cfg.CAFile)
			if err != nil {
				httpErr = fmt.Errorf("ZENITH_E2E_CA_FILE: %w", err)
				return
			}
			if !pool.AppendCertsFromPEM(pem) {
				httpErr = fmt.Errorf("ZENITH_E2E_CA_FILE %s: no PEM certificate found", cfg.CAFile)
				return
			}
		}
		dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			if host, port, err := net.SplitHostPort(addr); err == nil {
				if ip := sslipIP(host); ip != "" {
					addr = net.JoinHostPort(ip, port)
				}
			}
			return dialer.DialContext(ctx, network, addr)
		}
		httpC = &http.Client{Transport: tr, Timeout: 30 * time.Second}
	})
	return httpC, httpErr
}

func mustHTTP(t *testing.T) *http.Client {
	t.Helper()
	c, err := httpClient()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// newAPI logs in with email/password and returns a client over the CA-trusting transport.
func newAPI(t *testing.T, email, password string) (*cliapi.Client, error) {
	t.Helper()
	c := cliapi.NewClient(cfg.Endpoint, "")
	c.HTTPClient = mustHTTP(t)
	logf(t, "POST %s/api/v1/auth/login as %s", cfg.Endpoint, email)
	if _, err := c.LoginWithPassword(email, password); err != nil {
		return nil, err
	}
	return c, nil
}

// api returns the admin client, logged in once per process.
func api(t *testing.T) *cliapi.Client {
	t.Helper()
	adminMu.Lock()
	defer adminMu.Unlock()
	if adminAPI == nil {
		c, err := newAPI(t, cfg.AdminEmail, cfg.AdminPassword)
		if err != nil {
			t.Fatalf("admin login: %v", err)
		}
		adminAPI = c
	}
	return adminAPI
}

// resetAPI forgets the admin session (after a restore the server has other users).
func resetAPI() {
	adminMu.Lock()
	adminAPI = nil
	adminMu.Unlock()
}

// apiLogin waits until the API accepts the login (after a reboot, restore or upgrade).
func apiLogin(t *testing.T, email, password string, timeout time.Duration) *cliapi.Client {
	t.Helper()
	var c *cliapi.Client
	waitFor(t, "login as "+email, timeout, func() (bool, string) {
		var err error
		c, err = newAPI(t, email, password)
		if err != nil {
			return false, err.Error()
		}
		return true, "ok"
	})
	return c
}

// call does one API request; out (may be nil) receives the decoded JSON. The raw body is returned for logging.
func call(t *testing.T, c *cliapi.Client, method, path string, body, out any) (string, error) {
	t.Helper()
	logf(t, "%s %s%s", method, cfg.Endpoint, path)
	var raw json.RawMessage
	if err := c.DoRaw(method, path, body, &raw); err != nil {
		return "", err
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return string(raw), fmt.Errorf("decode %s %s: %w; body: %s", method, path, err, raw)
		}
	}
	return string(raw), nil
}

// waitFor polls fn every 5 s until it reports done or timeout passes; the failure carries the last state.
func waitFor(t *testing.T, name string, timeout time.Duration, fn func() (bool, string)) {
	t.Helper()
	start := time.Now()
	deadline := start.Add(timeout)
	last := "(never polled)"
	for {
		done, state := fn()
		last = state
		logf(t, "wait %s (%ds/%ds): %s", name, int(time.Since(start).Seconds()), int(timeout.Seconds()), tail(state, 400))
		if done {
			return
		}
		if time.Now().Add(pollEvery).After(deadline) {
			t.Fatalf("%s: not done after %s; last state: %s", name, timeout, last)
		}
		time.Sleep(pollEvery)
	}
}

func do(t *testing.T, req *http.Request) (int, string) {
	t.Helper()
	resp, err := mustHTTP(t).Do(req)
	if err != nil {
		logf(t, "%s %s: %v", req.Method, req.URL, err)
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	logf(t, "%s %s → %d %s", req.Method, req.URL, resp.StatusCode, tail(strings.TrimSpace(string(b)), 200))
	return resp.StatusCode, string(b)
}

// getURL is an unauthenticated GET; status 0 means the request did not complete (the body is the error).
func getURL(t *testing.T, url string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return do(t, req)
}

func postJSON(t *testing.T, url, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	return do(t, req)
}

// runCmd runs argv with a timeout and returns the combined output; the whole output goes to suite.log.
func runCmd(t *testing.T, timeout time.Duration, env []string, argv ...string) (string, error) {
	t.Helper()
	if len(argv) == 0 {
		t.Fatal("runCmd: empty command")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if env != nil {
		cmd.Env = env
	}
	logf(t, "run: %s", strings.Join(argv, " "))
	start := time.Now()
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("timed out after %s: %w", timeout, err)
	}
	writeLog(t.Name(), fmt.Sprintf("output of %q (%s, err=%v):\n%s", strings.Join(argv, " "), time.Since(start).Round(time.Second), err, out))
	logf(t, "exit after %s, err=%v, output:\n%s", time.Since(start).Round(time.Second), err, tail(string(out), 3000))
	return string(out), err
}

// zen runs the candidate zen locally with a throwaway HOME, the install CA and the admin password.
func zen(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return zenAs(t, cfg.AdminPassword, args...)
}

func zenAs(t *testing.T, password string, args ...string) (string, error) {
	t.Helper()
	env := make([]string, 0, len(os.Environ())+4)
	for _, kv := range os.Environ() {
		k := strings.SplitN(kv, "=", 2)[0]
		if k == "HOME" || k == "SSL_CERT_FILE" || k == "ZEN_PASSWORD" || k == "XDG_CONFIG_HOME" {
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "HOME="+zenHome, "ZEN_PASSWORD="+password, "NO_COLOR=1")
	if cfg.CAFile != "" {
		env = append(env, "SSL_CERT_FILE="+cfg.CAFile)
	}
	return runCmd(t, zenWait, env, append([]string{cfg.Zen}, args...)...)
}

// zenLogin logs the local zen in as the admin.
func zenLogin(t *testing.T) {
	t.Helper()
	out, err := zen(t, "login", "--endpoint", cfg.Endpoint, "--email", cfg.AdminEmail)
	if err != nil || !strings.Contains(out, "Logged in") {
		t.Fatalf("zen login: %v\n%s", err, out)
	}
}

// serverExec runs a command on the server (cwd /opt/zenith) through ZENITH_E2E_SERVER_EXEC.
func serverExec(t *testing.T, timeout time.Duration, args ...string) (string, error) {
	t.Helper()
	if len(cfg.ServerExec) == 0 {
		t.Fatal("ZENITH_E2E_SERVER_EXEC is not set")
	}
	return runCmd(t, timeout, nil, append(append([]string{}, cfg.ServerExec...), args...)...)
}

// waitHealth waits for GET <endpoint>/api/v1/version 200.
func waitHealth(t *testing.T, timeout time.Duration) {
	t.Helper()
	waitFor(t, "/api/v1/version", timeout, func() (bool, string) {
		code, body := getURL(t, cfg.Endpoint+"/api/v1/version")
		return code == http.StatusOK, fmt.Sprintf("%d %s", code, body)
	})
}

// --- state shared between tests -------------------------------------------------

// state is what TestDeployWithDatabaseAndDomain leaves for TestReboot and TestBackup.
type state struct {
	AppID    string `json:"app_id"`
	AppName  string `json:"app_name"`
	AppURL   string `json:"app_url"` // https://<app host>
	DBID     string `json:"db_id"`
	DBName   string `json:"db_name"`
	Marker   string `json:"marker"` // the row written through the probe
	Domain   string `json:"domain,omitempty"`
	DomainID string `json:"domain_id,omitempty"`
}

// requireDeployState loads state.json and fails unless TestDeployWithDatabaseAndDomain left the app and its database.
func requireDeployState(t *testing.T) *state {
	t.Helper()
	st := loadState(t)
	if st.AppID == "" || st.AppURL == "" || st.DBID == "" || st.Marker == "" {
		t.Fatalf("state %s lacks the probe app or its database: %+v", statePath(), *st)
	}
	return st
}

func statePath() string { return filepath.Join(cfg.Artifacts, "state.json") }

func saveState(t *testing.T, st *state) {
	t.Helper()
	b, _ := json.MarshalIndent(st, "", "  ")
	if err := os.WriteFile(statePath(), b, 0o644); err != nil {
		t.Fatalf("save state: %v", err)
	}
	logf(t, "state saved to %s: %s", statePath(), b)
}

func loadState(t *testing.T) *state {
	t.Helper()
	b, err := os.ReadFile(statePath())
	if err != nil {
		t.Fatalf("no state from TestDeployWithDatabaseAndDomain (%v): run it first in this artifacts dir", err)
	}
	var st state
	if err := json.Unmarshal(b, &st); err != nil {
		t.Fatalf("state %s: %v", statePath(), err)
	}
	logf(t, "state loaded from %s: %s", statePath(), b)
	return &st
}

// --- API shapes (services/api handlers) -----------------------------------------

type appInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	URL    string `json:"url"`
}

type dbInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Engine string `json:"engine"`
	Status string `json:"status"`
}

type domainInfo struct {
	ID            string  `json:"id"`
	Domain        string  `json:"domain"`
	Status        string  `json:"status"`
	TLSReady      bool    `json:"tls_ready"`
	StatusMessage string  `json:"status_message"`
	LastCheckedAt *string `json:"last_checked_at"`
}

type envVar struct {
	Key    string `json:"key"`
	Source string `json:"source"`
}

func listApps(t *testing.T, c *cliapi.Client) ([]appInfo, string, error) {
	t.Helper()
	var resp struct {
		Items []appInfo `json:"items"`
	}
	raw, err := call(t, c, "GET", "/api/v1/apps", nil, &resp)
	return resp.Items, raw, err
}

func findApp(t *testing.T, c *cliapi.Client, pred func(appInfo) bool) (*appInfo, string) {
	t.Helper()
	apps, raw, err := listApps(t, c)
	if err != nil {
		return nil, err.Error()
	}
	for i := range apps {
		if pred(apps[i]) {
			return &apps[i], fmt.Sprintf("%s status=%s url=%s", apps[i].Name, apps[i].Status, apps[i].URL)
		}
	}
	return nil, "app not listed: " + raw
}

// waitAppRunning waits for the app to be running and its generated URL to answer "ok"; it returns the app.
func waitAppRunning(t *testing.T, c *cliapi.Client, pred func(appInfo) bool, timeout time.Duration) *appInfo {
	t.Helper()
	var app *appInfo
	waitFor(t, "app running and answering ok", timeout, func() (bool, string) {
		a, st := findApp(t, c, pred)
		if a == nil {
			return false, st
		}
		app = a
		if a.Status != "running" || a.URL == "" {
			return false, st
		}
		code, body := getURL(t, a.URL+"/")
		return code == http.StatusOK && strings.TrimSpace(body) == "ok", fmt.Sprintf("%s; GET / → %d %s", st, code, tail(body, 200))
	})
	return app
}

func getDB(t *testing.T, c *cliapi.Client, appID, dbID string) (*dbInfo, string) {
	t.Helper()
	var db dbInfo
	raw, err := call(t, c, "GET", "/api/v1/apps/"+appID+"/databases/"+dbID, nil, &db)
	if err != nil {
		return nil, err.Error()
	}
	return &db, raw
}

func waitDBReady(t *testing.T, c *cliapi.Client, appID, dbID string, timeout time.Duration) {
	t.Helper()
	waitFor(t, "database "+dbID+" ready", timeout, func() (bool, string) {
		db, raw := getDB(t, c, appID, dbID)
		if db == nil {
			return false, raw
		}
		return db.Status == "ready", "status=" + db.Status + " " + tail(raw, 300)
	})
}

// probe is GET <app URL><path> against the probe app.
func probe(t *testing.T, st *state, path string) (int, string) {
	t.Helper()
	return getURL(t, st.AppURL+path)
}

// waitProbeRead waits until the probe's /read lists marker.
func waitProbeRead(t *testing.T, st *state, marker string, timeout time.Duration) {
	t.Helper()
	waitFor(t, "probe /read has "+marker, timeout, func() (bool, string) {
		code, body := probe(t, st, "/read")
		if code != http.StatusOK {
			return false, fmt.Sprintf("%d %s", code, body)
		}
		var vals []string
		if err := json.Unmarshal([]byte(body), &vals); err != nil {
			return false, fmt.Sprintf("decode %q: %v", body, err)
		}
		for _, v := range vals {
			if v == marker {
				return true, body
			}
		}
		return false, body
	})
}

func listDomains(t *testing.T, c *cliapi.Client, appID string) ([]domainInfo, string, error) {
	t.Helper()
	var ds []domainInfo
	raw, err := call(t, c, "GET", "/api/v1/apps/"+appID+"/domains", nil, &ds)
	return ds, raw, err
}

func domainState(d domainInfo) string {
	checked := "never"
	if d.LastCheckedAt != nil {
		checked = *d.LastCheckedAt
	}
	return fmt.Sprintf("%s status=%s tls_ready=%v message=%q last_checked_at=%s", d.Domain, d.Status, d.TLSReady, d.StatusMessage, checked)
}

// waitDomainActive waits for status active and tls_ready; every sixth poll asks for a check now (POST …/verify).
func waitDomainActive(t *testing.T, c *cliapi.Client, appID, domainID string, timeout time.Duration) {
	t.Helper()
	polls := 0
	waitFor(t, "domain "+domainID+" active with tls", timeout, func() (bool, string) {
		defer func() { polls++ }()
		var d *domainInfo
		if polls%6 == 0 {
			var v domainInfo
			raw, err := call(t, c, "POST", "/api/v1/apps/"+appID+"/domains/"+domainID+"/verify", nil, &v)
			if err == nil {
				d = &v
			} else {
				logf(t, "verify: %v %s", err, raw)
			}
		}
		if d == nil {
			ds, raw, err := listDomains(t, c, appID)
			if err != nil {
				return false, err.Error()
			}
			for i := range ds {
				if ds[i].ID == domainID {
					d = &ds[i]
				}
			}
			if d == nil {
				return false, "domain not listed: " + raw
			}
		}
		return d.Status == "active" && d.TLSReady, domainState(*d)
	})
}

// zenStatus logs zen in and checks that zen status lists every name in want.
func zenStatus(t *testing.T, want ...string) {
	t.Helper()
	zenLogin(t)
	out, err := zen(t, "status")
	if err != nil {
		t.Fatalf("zen status: %v\n%s", err, out)
	}
	for _, w := range want {
		if !hasRow(out, w) {
			t.Fatalf("zen status has no table row named %q:\n%s", w, out)
		}
	}
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// hasRow reports whether a line of a zen table starts with the cell name (NAME is the first column).
func hasRow(out, name string) bool {
	for _, line := range strings.Split(ansi.ReplaceAllString(out, ""), "\n") {
		if f := strings.Fields(line); len(f) > 0 && f[0] == name {
			return true
		}
	}
	return false
}

// --- the deploy flow shared by TestDeployWithDatabaseAndDomain and TestUpgrade --

// deployProbe deploys the probe app with an attached Postgres and writes marker through it.
func deployProbe(t *testing.T, c *cliapi.Client, marker string) *state {
	t.Helper()
	if cfg.ProbeImage == "" {
		t.Fatal("ZENITH_E2E_PROBE_IMAGE is not set")
	}
	// zen deploy deploys to an existing app (POST /api/v1/deploy looks it up by name), so the app is created
	// through the API first; creating an image app deploys it.
	var created appInfo
	raw, err := call(t, c, "POST", "/api/v1/apps", map[string]any{
		"name": probeAppName, "deploy_source": "image", "image_url": cfg.ProbeImage, "port": probePort,
	}, &created)
	if err != nil {
		t.Fatalf("create app %s: %v %s", probeAppName, err, raw)
	}
	if created.ID == "" {
		t.Fatalf("create app: no id in %s", raw)
	}
	byID := func(a appInfo) bool { return a.ID == created.ID }
	app := waitAppRunning(t, c, byID, appRunningWait)

	// The CLI path: redeploy the same image with zen deploy.
	zenLogin(t)
	// zen deploy exits 0 when its own 3-minute wait runs out, so its output is checked too.
	if out, err := zen(t, "deploy", "--image", cfg.ProbeImage, "--name", probeAppName); err != nil || !strings.Contains(out, "Deployed successfully") {
		t.Fatalf("zen deploy: %v\n%s", err, out)
	}
	app = waitAppRunning(t, c, byID, appRunningWait)
	st := &state{AppID: app.ID, AppName: app.Name, AppURL: strings.TrimRight(app.URL, "/"), DBName: probeDBName, Marker: marker}

	var db dbInfo
	raw, err = call(t, c, "POST", "/api/v1/apps/"+st.AppID+"/databases", map[string]string{"name": probeDBName, "engine": "postgresql"}, &db)
	if err != nil || db.ID == "" {
		t.Fatalf("create database: %v %s", err, raw)
	}
	st.DBID = db.ID
	logf(t, "database %s created, status=%s", db.ID, db.Status)
	waitDBReady(t, c, st.AppID, st.DBID, dbReadyWait)

	// Attaching the database injects DATABASE_URL and redeploys the app.
	waitFor(t, "probe /env lists DATABASE_URL", envInjectWait, func() (bool, string) {
		code, body := probe(t, st, "/env")
		if code != http.StatusOK {
			return false, fmt.Sprintf("%d %s", code, body)
		}
		var names []string
		if err := json.Unmarshal([]byte(body), &names); err != nil {
			return false, fmt.Sprintf("decode %q: %v", body, err)
		}
		for _, n := range names {
			if n == "DATABASE_URL" {
				return true, body
			}
		}
		return false, body
	})
	waitFor(t, "probe /write?v="+marker, 60*time.Second, func() (bool, string) {
		code, body := probe(t, st, "/write?v="+marker)
		return code == http.StatusOK && strings.TrimSpace(body) == "written", fmt.Sprintf("%d %s", code, body)
	})
	waitProbeRead(t, st, marker, 60*time.Second)

	var env struct {
		Items []envVar `json:"items"`
	}
	raw, err = call(t, c, "GET", "/api/v1/apps/"+st.AppID+"/env-v2", nil, &env)
	if err != nil {
		t.Fatalf("env-v2: %v", err)
	}
	found := false
	for _, v := range env.Items {
		if v.Key == "DATABASE_URL" {
			found = true
			if v.Source != "managed_service" {
				t.Fatalf("DATABASE_URL source = %q, want managed_service; body: %s", v.Source, raw)
			}
		}
	}
	if !found {
		t.Fatalf("env-v2 does not list DATABASE_URL: %s", raw)
	}
	return st
}

// errNotNewer reports whether zen upgrade refused because the target is not newer than the install.
func errNotNewer(out string, err error) bool {
	return err != nil && strings.Contains(out, "not newer")
}
