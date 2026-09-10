package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/shubhams01/cachefy/internal/cache"
)

type Handler struct {
	cache *cache.Cache
}

func NewHandler(c *cache.Cache) http.Handler {
	h := &Handler{
		cache: c,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /cache/{key}", h.get)
	mux.HandleFunc("PUT /cache/{key}", h.set)
	mux.HandleFunc("DELETE /cache/{key}", h.delete)

	return mux
}

type setRequest struct {
	Value string `json:"value"`
	TTL   int64  `json:"ttl"`
}

type response struct {
	Value string `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	value, err := h.cache.Get(key)

	if err != nil {
		if err == cache.ErrKeyNotFound {
			writeError(w, http.StatusNotFound, "key not found")
			return
		}

		if err == cache.ErrCacheClosed {
			writeError(w, http.StatusServiceUnavailable, "cache is closed")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, response{
		Value: string(value),
	})
}

func (h *Handler) set(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	if strings.TrimSpace(key) == "" {
		writeError(w, http.StatusBadRequest, "key cannot be empty")
		return
	}

	var request setRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var ttl time.Duration

	if request.TTL > 0 {
		ttl = time.Duration(request.TTL) * time.Second
	}

	if err := h.cache.Set(key, []byte(request.Value), ttl); err != nil {
		if err == cache.ErrCacheClosed {
			writeError(w, http.StatusServiceUnavailable, "cache is closed")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	if err := h.cache.Delete(key); err != nil {
		if err == cache.ErrCacheClosed {
			writeError(w, http.StatusServiceUnavailable, "cache is closed")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, response{
		Error: message,
	})
}
