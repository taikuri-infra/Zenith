package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func tokenJSON(access, refresh string, expiresIn int) string {
	b, _ := json.Marshal(map[string]any{"access_token": access, "refresh_token": refresh, "token_type": "Bearer", "expires_in": expiresIn})
	return string(b)
}

func TestLoginWithPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" || r.Method != "POST" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["email"] != "a@b.c" || body["password"] != "pw" {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(tokenJSON("acc", "ref", 3600)))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "")
	before := time.Now()
	ts, err := c.LoginWithPassword("a@b.c", "pw")
	if err != nil {
		t.Fatal(err)
	}
	if ts.AccessToken != "acc" || ts.RefreshToken != "ref" || ts.ExpiresAt.Before(before.Add(59*time.Minute)) {
		t.Fatalf("ts = %+v", ts)
	}
	if c.Token != "acc" {
		t.Fatal("login must set the client's token")
	}
}

func TestLoginWithPassword_MFARequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			w.Write([]byte(`{"mfa_required":true,"mfa_token":"mfa-1"}`))
		case "/api/v1/auth/login/mfa":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["mfa_token"] != "mfa-1" || body["code"] != "123456" {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(tokenJSON("acc2", "ref2", 3600)))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "")
	_, err := c.LoginWithPassword("a@b.c", "pw")
	var mfa *ErrMFARequired
	if !errors.As(err, &mfa) || mfa.MFAToken != "mfa-1" {
		t.Fatalf("err = %v", err)
	}
	ts, err := c.LoginMFA(mfa.MFAToken, "123456")
	if err != nil || ts.AccessToken != "acc2" {
		t.Fatalf("ts = %+v err = %v", ts, err)
	}
}

func TestClient_RefreshesBeforeExpiry(t *testing.T) {
	var refreshes int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			atomic.AddInt32(&refreshes, 1)
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["refresh_token"] != "ref" {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(tokenJSON("fresh", "ref2", 3600)))
		case "/api/v1/projects":
			if r.Header.Get("Authorization") != "Bearer fresh" {
				t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
			}
			w.Write([]byte(`{"items":[]}`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "stale")
	var saved TokenSet
	c.OnTokens = func(ts TokenSet) { saved = ts }
	c.SetSession(TokenSet{AccessToken: "stale", RefreshToken: "ref", ExpiresAt: time.Now().Add(30 * time.Second)})
	if _, err := c.ListProjects(); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&refreshes) != 1 || saved.AccessToken != "fresh" || saved.RefreshToken != "ref2" {
		t.Fatalf("refreshes = %d saved = %+v", refreshes, saved)
	}
}

func TestClient_RefreshesAndRetriesOn401(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			w.Write([]byte(tokenJSON("fresh", "ref", 3600)))
		case "/api/v1/projects":
			if atomic.AddInt32(&calls, 1) == 1 {
				w.WriteHeader(401)
				return
			}
			w.Write([]byte(`{"items":[]}`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "")
	c.SetSession(TokenSet{AccessToken: "old", RefreshToken: "ref", ExpiresAt: time.Now().Add(time.Hour)})
	if _, err := c.ListProjects(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestClient_RefreshFailureIsReportedOnce(t *testing.T) {
	var refreshes, calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			atomic.AddInt32(&refreshes, 1)
			w.WriteHeader(401)
		case "/api/v1/projects":
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(401)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "")
	c.SetSession(TokenSet{AccessToken: "old", RefreshToken: "ref", ExpiresAt: time.Now().Add(time.Hour)})
	_, err := c.ListProjects()
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("err = %v, want ErrSessionExpired", err)
	}
	if refreshes != 1 || calls != 1 {
		t.Fatalf("refreshes = %d calls = %d (must not loop)", refreshes, calls)
	}
}

func TestClient_NoRefreshWithoutRefreshToken(t *testing.T) {
	var refreshes int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/refresh" {
			atomic.AddInt32(&refreshes, 1)
		}
		w.WriteHeader(401)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "pasted-token") // old config: token only
	if _, err := c.ListProjects(); err == nil {
		t.Fatal("expected an error")
	}
	if refreshes != 0 {
		t.Fatal("must not call refresh without a refresh token")
	}
}

func TestLogout(t *testing.T) {
	for _, tc := range []struct{ refresh, wantBody string }{
		{"ref", `{"refresh_token":"ref"}`},
		{"", `{"refresh_token":""}`},
	} {
		var hit bool
		var body string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hit = r.URL.Path == "/api/v1/auth/logout" && r.Method == "POST" && r.Header.Get("Authorization") == "Bearer acc"
			b, _ := io.ReadAll(r.Body)
			body = string(b)
		}))
		c := NewClient(srv.URL, "acc")
		c.RefreshToken = tc.refresh
		if err := c.Logout(); err != nil || !hit {
			t.Fatalf("err = %v hit = %v", err, hit)
		}
		if body != tc.wantBody {
			t.Errorf("body = %s, want %s", body, tc.wantBody)
		}
		srv.Close()
	}
}
