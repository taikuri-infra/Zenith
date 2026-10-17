//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestDeployWithDatabaseAndDomain: an image app with an attached Postgres keeps a row, gets DATABASE_URL
// as a managed env var, and serves a custom domain over TLS that verifies against the install CA.
func TestDeployWithDatabaseAndDomain(t *testing.T) {
	c := api(t)
	st := deployProbe(t, c, "alpha-1")
	saveState(t, st) // TestReboot / TestBackup can use the app even if the domain step fails

	st.Domain = "myapp." + cfg.Domain
	var d domainInfo
	raw, err := call(t, c, "POST", "/api/v1/apps/"+st.AppID+"/domains", map[string]string{"domain": st.Domain}, &d)
	if err != nil || d.ID == "" {
		t.Fatalf("add domain %s: %v %s", st.Domain, err, raw)
	}
	st.DomainID = d.ID
	logf(t, "domain added: %s", domainState(d))
	saveState(t, st)
	waitDomainActive(t, c, st.AppID, st.DomainID, domainWait)

	// The http client trusts only the system roots and the install CA, so a 200 here means the chain verified.
	waitFor(t, "GET https://"+st.Domain+"/", 60*time.Second, func() (bool, string) {
		code, body := getURL(t, "https://"+st.Domain+"/")
		return code == http.StatusOK && strings.TrimSpace(body) == "ok", tail(body, 300)
	})
}
