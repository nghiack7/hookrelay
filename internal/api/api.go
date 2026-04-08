package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nghiack7/hookrelay/internal/sse"
	"github.com/nghiack7/hookrelay/internal/store"
)

type API struct {
	store *store.Store
	hub   *sse.Hub
}

func New(s *store.Store, hub *sse.Hub) *API {
	return &API{store: s, hub: hub}
}

func (a *API) Register(mux *http.ServeMux) {
	// Webhook capture endpoint (all common methods)
	mux.HandleFunc("POST /hook/{endpointID}", a.handleWebhook)
	mux.HandleFunc("PUT /hook/{endpointID}", a.handleWebhook)
	mux.HandleFunc("PATCH /hook/{endpointID}", a.handleWebhook)
	mux.HandleFunc("DELETE /hook/{endpointID}", a.handleWebhook)
	mux.HandleFunc("GET /hook/{endpointID}", a.handleWebhook)

	// Management API
	mux.HandleFunc("POST /api/endpoints", a.createEndpoint)
	mux.HandleFunc("GET /api/endpoints", a.listEndpoints)
	mux.HandleFunc("DELETE /api/endpoints/{id}", a.deleteEndpoint)
	mux.HandleFunc("GET /api/endpoints/{id}/requests", a.listRequests)
	mux.HandleFunc("GET /api/endpoints/{id}/stream", a.streamRequests)
	mux.HandleFunc("POST /api/requests/{id}/replay", a.replayRequest)

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Quick start - generate endpoint without API key
	mux.HandleFunc("POST /api/quick", a.quickStart)
}

func (a *API) handleWebhook(w http.ResponseWriter, r *http.Request) {
	endpointID := r.PathValue("endpointID")

	ep, err := a.store.GetEndpoint(endpointID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "endpoint not found"})
		return
	}
	_ = ep

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}

	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}

	req, err := a.store.SaveRequest(endpointID, r.Method, r.URL.Path, string(body), r.URL.RawQuery, r.RemoteAddr, headers)
	if err != nil {
		slog.Error("save request", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
		return
	}

	// Broadcast to SSE subscribers
	a.hub.Broadcast(endpointID, req)

	writeJSON(w, http.StatusOK, map[string]string{"status": "received", "id": req.ID})
}

func (a *API) createEndpoint(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "X-API-Key header required"})
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	ep, err := a.store.CreateEndpoint(apiKey, body.Name)
	if err != nil {
		slog.Error("create endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create"})
		return
	}

	writeJSON(w, http.StatusCreated, ep)
}

func (a *API) listEndpoints(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "X-API-Key header required"})
		return
	}

	endpoints, err := a.store.ListEndpoints(apiKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list"})
		return
	}
	if endpoints == nil {
		endpoints = []store.Endpoint{}
	}
	writeJSON(w, http.StatusOK, endpoints)
}

func (a *API) deleteEndpoint(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "X-API-Key header required"})
		return
	}

	id := r.PathValue("id")
	if err := a.store.DeleteEndpoint(id, apiKey); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) listRequests(w http.ResponseWriter, r *http.Request) {
	endpointID := r.PathValue("id")

	requests, err := a.store.ListRequests(endpointID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list"})
		return
	}
	if requests == nil {
		requests = []store.Request{}
	}
	writeJSON(w, http.StatusOK, requests)
}

func (a *API) streamRequests(w http.ResponseWriter, r *http.Request) {
	endpointID := r.PathValue("id")
	a.hub.ServeHTTP(w, r, endpointID)
}

func (a *API) replayRequest(w http.ResponseWriter, r *http.Request) {
	reqID := r.PathValue("id")

	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url field required"})
		return
	}

	original, err := a.store.GetRequest(reqID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "request not found"})
		return
	}

	// Replay the request
	replayReq, err := http.NewRequest(original.Method, body.URL, strings.NewReader(original.Body))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid replay URL"})
		return
	}
	for k, v := range original.Headers {
		if strings.HasPrefix(strings.ToLower(k), "x-") || strings.ToLower(k) == "content-type" {
			replayReq.Header.Set(k, v)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(replayReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "replay failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      resp.StatusCode,
		"body":        string(respBody),
		"original_id": original.ID,
	})
}

func (a *API) quickStart(w http.ResponseWriter, r *http.Request) {
	apiKey := generateAPIKey()

	ep, err := a.store.CreateEndpoint(apiKey, "My First Endpoint")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"endpoint": ep,
		"api_key":  apiKey,
		"hook_url": "/hook/" + ep.ID,
	})
}

func generateAPIKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "hk_" + hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
