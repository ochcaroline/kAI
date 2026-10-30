package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestServer(t *testing.T) {
	st, err := NewStore(filepath.Join(t.TempDir(), "b.json"))
	if err != nil {
		t.Fatal(err)
	}
	h, err := handler(st, "kai:7777")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, method, host, ctype, body string
		want                            int
	}{
		{"create", "POST", "127.0.0.1:7777", "application/json", `{"title":"a"}`, http.StatusCreated},
		{"no json", "POST", "127.0.0.1:7777", "text/plain", `{"title":"a"}`, http.StatusUnsupportedMediaType},
		{"unknown field", "POST", "127.0.0.1:7777", "application/json", `{"nope":1}`, http.StatusBadRequest},
		{"too big", "POST", "127.0.0.1:7777", "application/json", `{"title":"` + strings.Repeat("x", 2<<20) + `"}`, http.StatusBadRequest},
		{"rebinding read", "GET", "evil.com:7777", "", "", http.StatusForbidden},
		{"rebinding delete", "DELETE", "evil.com", "", "", http.StatusForbidden},
		{"localhost", "GET", "localhost:7777", "", "", http.StatusOK},
		{"ipv6", "GET", "[::1]:7777", "", "", http.StatusOK},
		{"dot localhost", "GET", "kai.localhost:7777", "", "", http.StatusOK},
		{"addr host", "GET", "KAI:7777", "", "", http.StatusOK},
		{"ui", "GET", "127.0.0.1:7777", "", "", http.StatusOK},
	} {
		path := "/api/tasks"
		if c.method == "DELETE" {
			path += "/1"
		}
		if c.name == "ui" {
			path = "/"
		}
		r := httptest.NewRequest(c.method, path, strings.NewReader(c.body))
		r.Host = c.host
		if c.ctype != "" {
			r.Header.Set("Content-Type", c.ctype)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s: got %d, want %d (%s)", c.name, w.Code, c.want, w.Body)
		}
	}
	if _, err := st.Get("1"); err != nil {
		t.Errorf("rebinding delete must not remove the task: %v", err)
	}
}
