package main

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
)

//go:embed web
var webFS embed.FS

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, ErrAmbiguous):
		code = http.StatusConflict
	case errors.Is(err, ErrInvalid):
		code = http.StatusBadRequest
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func readPatch(w http.ResponseWriter, r *http.Request) (Patch, bool) {
	var p Patch
	// Requiring JSON forces a CORS preflight, so other websites cannot write to localhost.
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type must be application/json"})
		return p, false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return p, false
	}
	return p, true
}

// localOnly rejects requests whose Host is not loopback or the -addr host, which blocks DNS rebinding.
func localOnly(addr string, next http.Handler) http.Handler {
	own, _, _ := net.SplitHostPort(addr)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			h = strings.Trim(r.Host, "[]")
		}
		h = strings.ToLower(h)
		if h != "localhost" && h != "127.0.0.1" && h != "::1" && !strings.HasSuffix(h, ".localhost") && (own == "" || h != strings.ToLower(own)) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "host not allowed: " + r.Host})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func serve(st *Store, addr string) error {
	h, err := handler(st, addr)
	if err != nil {
		return err
	}
	log.Printf("kainban web on http://%s (board: %s)", addr, st.path)
	return http.ListenAndServe(addr, h)
}

func handler(st *Store, addr string) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/tasks", func(w http.ResponseWriter, r *http.Request) {
		ts, err := st.List(r.URL.Query().Get("q"), r.URL.Query().Get("status"))
		if err != nil {
			writeErr(w, err)
			return
		}
		if ts == nil {
			ts = []Task{}
		}
		writeJSON(w, http.StatusOK, ts)
	})
	mux.HandleFunc("POST /api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if p, ok := readPatch(w, r); ok {
			if t, err := st.Create(p); err != nil {
				writeErr(w, err)
			} else {
				writeJSON(w, http.StatusCreated, t)
			}
		}
	})
	// {ref} is a numeric id ("12", "#12") or unique title text.
	mux.HandleFunc("GET /api/tasks/{ref}", func(w http.ResponseWriter, r *http.Request) {
		if t, err := st.Get(r.PathValue("ref")); err != nil {
			writeErr(w, err)
		} else {
			writeJSON(w, http.StatusOK, t)
		}
	})
	mux.HandleFunc("PATCH /api/tasks/{ref}", func(w http.ResponseWriter, r *http.Request) {
		if p, ok := readPatch(w, r); ok {
			if t, err := st.Update(r.PathValue("ref"), p); err != nil {
				writeErr(w, err)
			} else {
				writeJSON(w, http.StatusOK, t)
			}
		}
	})
	mux.HandleFunc("DELETE /api/tasks/{ref}", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Delete(r.PathValue("ref")); err != nil {
			writeErr(w, err)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	})
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServerFS(sub))
	return localOnly(addr, mux), nil
}
