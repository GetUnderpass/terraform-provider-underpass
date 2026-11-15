package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeAPI answers like the coordinator's /api/v1 (server/internal/api),
// recording what it was sent.
func fakeAPI(t *testing.T) (*Client, map[string]string) {
	t.Helper()
	got := map[string]string{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/keys", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got["create key"] = string(b)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"key":"tskey-secret","authKey":{"id":"k1","prefix":"tskey-ab","reusable":true,"tags":["tag:ci"],"valid":true,"expiresAt":"2027-02-01T00:00:00Z"}}`)
	})
	mux.HandleFunc("GET /api/v1/keys", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"keys":[{"id":"k1","prefix":"tskey-ab","valid":true}]}`)
	})
	mux.HandleFunc("DELETE /api/v1/keys/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"error":"auth key not found"}`)
	})
	mux.HandleFunc("GET /api/v1/policy", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"custom":true,"rules":[{"id":"r1","src":["*"],"dst":["tag:web"],"proto":"any","ports":["*"]}],"ssh":[],"remote":[],"clipboard":[],"groups":{},"tests":[],"results":[],"version":"x","builtin":["shared-clipboard"]}`)
	})
	mux.HandleFunc("PATCH /api/v1/settings", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got["settings"] = string(b)
		io.WriteString(w, `{}`)
	})
	mux.HandleFunc("PUT /api/v1/policy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":"1 test would fail; the policy wasn't saved","results":[]}`)
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tsapi-test" {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `{"error":"invalid or expired API token"}`)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL+"/", "tsapi-test"), got
}

func TestAuthKeys(t *testing.T) {
	c, got := fakeAPI(t)
	ctx := context.Background()
	k, secret, err := c.CreateAuthKey(ctx, NewAuthKey{Reusable: true, Tags: []string{"tag:ci"}, ExpirySeconds: 3600})
	if err != nil || secret != "tskey-secret" || k.ID != "k1" || len(k.Tags) != 1 {
		t.Fatalf("create: %+v %q %v", k, secret, err)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(got["create key"]), &sent)
	if sent["reusable"] != true || sent["expirySeconds"] != float64(3600) {
		t.Errorf("sent %s", got["create key"])
	}
	if _, err := c.AuthKey(ctx, "k1"); err != nil {
		t.Errorf("read: %v", err)
	}
	if _, err := c.AuthKey(ctx, "gone"); !IsNotFound(err) {
		t.Errorf("a missing key: %v, want not found", err)
	}
	// Revoking a key that's already gone isn't an error.
	if err := c.RevokeAuthKey(ctx, "gone"); err != nil {
		t.Errorf("revoke a gone key: %v", err)
	}
}

func TestPolicyKeepsOnlyWhatsWritten(t *testing.T) {
	c, _ := fakeAPI(t)
	p, custom, err := c.Policy(context.Background())
	if err != nil || !custom {
		t.Fatalf("%v %v", custom, err)
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(p, &m)
	for _, k := range []string{"results", "version", "builtin", "custom"} {
		if _, ok := m[k]; ok {
			t.Errorf("read-only %q kept in the policy", k)
		}
	}
	if _, ok := m["rules"]; !ok {
		t.Error("rules dropped")
	}
}

func TestSettingsSendOnlyWhatsSet(t *testing.T) {
	c, got := fakeAPI(t)
	on := true
	if err := c.UpdateSettings(context.Background(), Settings{DeviceApproval: &on}); err != nil {
		t.Fatal(err)
	}
	if got["settings"] != `{"deviceApproval":true}` {
		t.Errorf("sent %s, want only deviceApproval", got["settings"])
	}
}

func TestErrorsCarryTheAPIMessage(t *testing.T) {
	c, _ := fakeAPI(t)
	err := c.SetPolicy(context.Background(), json.RawMessage(`{}`))
	var e *Error
	if err == nil || !errorAs(err, &e) || e.Status != 400 || e.Message != "1 test would fail; the policy wasn't saved" {
		t.Errorf("got %v", err)
	}
	bad := New(c.base[:len(c.base)-len("/api/v1")], "tsapi-wrong")
	if _, err := bad.Settings(context.Background()); err == nil || !errorAs(err, &e) || e.Status != 401 {
		t.Errorf("wrong token: %v", err)
	}
}

func errorAs(err error, target **Error) bool {
	e, ok := err.(*Error)
	if ok {
		*target = e
	}
	return ok
}
