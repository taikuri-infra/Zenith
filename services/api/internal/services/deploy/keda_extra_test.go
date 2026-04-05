package deploy

import (
	"encoding/json"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestGenerateColdStartMiddleware(t *testing.T) {
	mw := generateColdStartMiddleware("zenith-apps")

	if mw["apiVersion"] != "traefik.io/v1alpha1" {
		t.Errorf("apiVersion = %v, want traefik.io/v1alpha1", mw["apiVersion"])
	}
	if mw["kind"] != "Middleware" {
		t.Errorf("kind = %v, want Middleware", mw["kind"])
	}

	meta := mw["metadata"].(map[string]interface{})
	if meta["name"] != "cold-start-errors" {
		t.Errorf("name = %v, want cold-start-errors", meta["name"])
	}
	if meta["namespace"] != "zenith-apps" {
		t.Errorf("namespace = %v, want zenith-apps", meta["namespace"])
	}

	spec := mw["spec"].(map[string]interface{})
	errors := spec["errors"].(map[string]interface{})
	status := errors["status"].([]string)
	if len(status) != 1 || status[0] != "502-503" {
		t.Errorf("Expected status [502-503], got %v", status)
	}
}

func TestGenerateIngressRouteWithColdStart(t *testing.T) {
	app := &entities.App{
		ID:        "app-1",
		Subdomain: "coldapp",
		Port:      8080,
	}
	labels := map[string]string{"app": "coldapp"}

	ir := generateIngressRouteWithColdStart(app, "zenith-apps", labels, "freezenith.com", nil)

	if ir["kind"] != "IngressRoute" {
		t.Errorf("kind = %v, want IngressRoute", ir["kind"])
	}

	data, _ := json.Marshal(ir)
	content := string(data)

	if !containsStr(content, "cold-start-errors") {
		t.Error("Expected cold-start-errors middleware in IngressRoute")
	}
	if !containsStr(content, "coldapp.freezenith.com") {
		t.Error("Expected host coldapp.freezenith.com")
	}
}

func TestGenerateIngressRouteWithColdStart_CustomDomains(t *testing.T) {
	app := &entities.App{
		ID:        "app-2",
		Subdomain: "myapp",
		Port:      8080,
	}
	labels := map[string]string{"app": "myapp"}
	customDomains := []string{"example.com"}

	ir := generateIngressRouteWithColdStart(app, "zenith-apps", labels, "freezenith.com", customDomains)

	data, _ := json.Marshal(ir)
	content := string(data)

	if !containsStr(content, "example.com") {
		t.Error("Expected custom domain in IngressRoute")
	}

	spec := ir["spec"].(map[string]interface{})
	tls := spec["tls"].(map[string]interface{})
	if tls["secretName"] != "myapp-custom-tls" {
		t.Errorf("Expected TLS secretName 'myapp-custom-tls', got '%v'", tls["secretName"])
	}
}

func TestGenerateIngressRouteViaApisix(t *testing.T) {
	app := &entities.App{
		ID:        "app-3",
		Subdomain: "apiapp",
		Port:      8080,
	}
	labels := map[string]string{"app": "apiapp"}

	ir := generateIngressRouteViaApisix(app, "zenith-apps", labels, "freezenith.com", nil)

	if ir["kind"] != "IngressRoute" {
		t.Errorf("kind = %v, want IngressRoute", ir["kind"])
	}

	data, _ := json.Marshal(ir)
	content := string(data)

	if !containsStr(content, "apisix-gateway-bridge") {
		t.Error("Expected apisix-gateway-bridge service in IngressRoute")
	}
	if !containsStr(content, "apiapp.freezenith.com") {
		t.Error("Expected host apiapp.freezenith.com")
	}
}

func TestGenerateIngressRouteViaApisix_CustomDomains(t *testing.T) {
	app := &entities.App{
		ID:        "app-4",
		Subdomain: "customapi",
		Port:      8080,
	}
	labels := map[string]string{"app": "customapi"}

	ir := generateIngressRouteViaApisix(app, "zenith-apps", labels, "freezenith.com", []string{"api.example.com"})

	data, _ := json.Marshal(ir)
	content := string(data)

	if !containsStr(content, "api.example.com") {
		t.Error("Expected custom domain in IngressRoute")
	}

	spec := ir["spec"].(map[string]interface{})
	tls := spec["tls"].(map[string]interface{})
	if tls["secretName"] != "customapi-custom-tls" {
		t.Errorf("Expected TLS secretName 'customapi-custom-tls', got '%v'", tls["secretName"])
	}
}

func TestGenerateIngressRouteViaApisixWithColdStart(t *testing.T) {
	app := &entities.App{
		ID:        "app-5",
		Subdomain: "coldapi",
		Port:      8080,
	}
	labels := map[string]string{"app": "coldapi"}

	ir := generateIngressRouteViaApisixWithColdStart(app, "zenith-apps", labels, "freezenith.com", nil)

	if ir["kind"] != "IngressRoute" {
		t.Errorf("kind = %v, want IngressRoute", ir["kind"])
	}

	data, _ := json.Marshal(ir)
	content := string(data)

	if !containsStr(content, "apisix-gateway-bridge") {
		t.Error("Expected apisix-gateway-bridge service")
	}
	if !containsStr(content, "cold-start-errors") {
		t.Error("Expected cold-start-errors middleware")
	}
}

func TestGenerateIngressRouteViaApisixWithColdStart_CustomDomains(t *testing.T) {
	app := &entities.App{
		ID:        "app-6",
		Subdomain: "freeapi",
		Port:      8080,
	}
	labels := map[string]string{"app": "freeapi"}

	ir := generateIngressRouteViaApisixWithColdStart(app, "zenith-apps", labels, "freezenith.com", []string{"free.example.com"})

	spec := ir["spec"].(map[string]interface{})
	tls := spec["tls"].(map[string]interface{})
	if tls["secretName"] != "freeapi-custom-tls" {
		t.Errorf("Expected TLS secretName 'freeapi-custom-tls', got '%v'", tls["secretName"])
	}
}

func TestBuildHostMatchRule_SingleHost(t *testing.T) {
	rule := buildHostMatchRule("myapp.freezenith.com", nil)
	expected := "Host(`myapp.freezenith.com`)"
	if rule != expected {
		t.Errorf("Expected '%s', got '%s'", expected, rule)
	}
}

func TestBuildHostMatchRule_MultipleHosts(t *testing.T) {
	rule := buildHostMatchRule("myapp.freezenith.com", []string{"example.com", "www.example.com"})
	if !containsStr(rule, "Host(`myapp.freezenith.com`)") {
		t.Error("Expected primary host in rule")
	}
	if !containsStr(rule, "Host(`example.com`)") {
		t.Error("Expected example.com in rule")
	}
	if !containsStr(rule, "||") {
		t.Error("Expected || separator in rule")
	}
}

func TestGenerateCertificate(t *testing.T) {
	app := &entities.App{
		ID:        "app-cert",
		Subdomain: "certapp",
	}
	labels := map[string]string{"app": "certapp"}

	cert := generateCertificate(app, "zenith-apps", labels, "freezenith.com", []string{"example.com"})

	if cert["kind"] != "Certificate" {
		t.Errorf("kind = %v, want Certificate", cert["kind"])
	}
	if cert["apiVersion"] != "cert-manager.io/v1" {
		t.Errorf("apiVersion = %v, want cert-manager.io/v1", cert["apiVersion"])
	}

	meta := cert["metadata"].(map[string]interface{})
	if meta["name"] != "certapp-custom-tls" {
		t.Errorf("name = %v, want certapp-custom-tls", meta["name"])
	}

	spec := cert["spec"].(map[string]interface{})
	if spec["secretName"] != "certapp-custom-tls" {
		t.Errorf("secretName = %v, want certapp-custom-tls", spec["secretName"])
	}

	dnsNames := spec["dnsNames"].([]string)
	if len(dnsNames) != 2 {
		t.Fatalf("Expected 2 dnsNames, got %d", len(dnsNames))
	}
	if dnsNames[0] != "certapp.freezenith.com" {
		t.Errorf("Expected primary DNS name, got '%s'", dnsNames[0])
	}
	if dnsNames[1] != "example.com" {
		t.Errorf("Expected custom DNS name, got '%s'", dnsNames[1])
	}
}

func TestLogHub_Subscribe_ReplayOverflow(t *testing.T) {
	hub := NewLogHub(100)

	// Add more entries than the subscriber buffer can hold
	for i := 0; i < 20; i++ {
		hub.PublishInfo("d1", "msg")
	}

	// Subscribe with tiny buffer (3) - should skip oldest during replay
	sub := hub.Subscribe("d1", 3)
	defer sub.Close()

	// Should have received some entries (buffer size 3)
	count := 0
	for {
		select {
		case <-sub.Ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 3 {
		t.Errorf("Expected 3 replayed entries (buffer size), got %d", count)
	}
}

func TestNewLogHub_DefaultMaxHistory(t *testing.T) {
	hub := NewLogHub(0)
	if hub.maxHistory != 500 {
		t.Errorf("Expected default maxHistory 500, got %d", hub.maxHistory)
	}
}

func TestNewLogHub_NegativeMaxHistory(t *testing.T) {
	hub := NewLogHub(-10)
	if hub.maxHistory != 500 {
		t.Errorf("Expected default maxHistory 500, got %d", hub.maxHistory)
	}
}
