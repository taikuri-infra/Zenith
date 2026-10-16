package login

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/dotechhq/zenith/cli/internal/config"
)

func TestLoginWithPasswordSavesSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			json.NewEncoder(w).Encode(json.RawMessage(`{"access_token":"acc","refresh_token":"ref","token_type":"Bearer","expires_in":3600}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "config.yaml")
	var out bytes.Buffer
	err := runLoginTo(&out, loginInputs{endpoint: srv.URL, email: "a@b.c", password: "pw", configPath: path,
		promptMFA: func() (string, error) { t.Fatal("no MFA expected"); return "", nil }})
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.LoadFrom(path)
	if cfg.APIEndpoint != srv.URL || cfg.Token != "acc" || cfg.RefreshToken != "ref" || cfg.TokenExpiresAt.IsZero() {
		t.Fatalf("cfg = %+v", cfg)
	}
	if !bytes.Contains(out.Bytes(), []byte("Logged in to "+srv.URL+" as a@b.c")) {
		t.Fatalf("out = %s", out.String())
	}
}

func TestLoginMFAPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			json.NewEncoder(w).Encode(json.RawMessage(`{"mfa_required":true,"mfa_token":"m1"}`))
		case "/api/v1/auth/login/mfa":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["code"] != "000111" {
				t.Errorf("code = %q", body["code"])
			}
			json.NewEncoder(w).Encode(json.RawMessage(`{"access_token":"acc","refresh_token":"ref","token_type":"Bearer","expires_in":3600}`))
		}
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := runLoginTo(&bytes.Buffer{}, loginInputs{endpoint: srv.URL, email: "a@b.c", password: "pw", configPath: path,
		promptMFA: func() (string, error) { return "000111", nil }})
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.LoadFrom(path)
	if cfg.Token != "acc" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(json.RawMessage(`{"error":"invalid credentials"}`))
	}))
	defer srv.Close()
	err := runLoginTo(&bytes.Buffer{}, loginInputs{endpoint: srv.URL, email: "a@b.c", password: "bad", configPath: filepath.Join(t.TempDir(), "c.yaml")})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("invalid credentials")) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoginTokenOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := runLoginTo(&bytes.Buffer{}, loginInputs{endpoint: "https://z.example.com", token: "znt_id_a:znt_sk_b", configPath: path}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.LoadFrom(path)
	if cfg.Token != "znt_id_a:znt_sk_b" || cfg.RefreshToken != "" {
		t.Fatalf("cfg = %+v", cfg)
	}
}
