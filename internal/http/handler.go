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

	mux.HandleFunc("POST /cache/batch", h.setMany)
	mux.HandleFunc("POST /cache/batch/get", h.getMany)
	mux.HandleFunc("POST /cache/batch/delete", h.deleteMany)

	return mux
}

type setRequest struct {
	Value json.RawMessage `json:"value"`
	TTL   int64           `json:"ttl"`
}

type response struct {
	Value json.RawMessage `json:"value,omitempty"`
	Error string          `json:"error,omitempty"`
}

type batchSetItem struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
	TTL   int64           `json:"ttl"`
}

type batchSetRequest struct {
	Items []batchSetItem `json:"items"`
}

type batchGetRequest struct {
	Keys []string `json:"keys"`
}

type batchGetResponse struct {
	Items map[string]json.RawMessage `json:"items"`
}

func (h *Handler) health(
	w http.ResponseWriter,
	r *http.Request,
) {
	writeJSON(
		w,
		http.StatusOK,
		map[string]string{
			"status": "ok",
		},
	)
}

func (h *Handler) get(
	w http.ResponseWriter,
	r *http.Request,
) {
	key := r.PathValue("key")

	if key == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"key is required",
		)
		return
	}

	value, err := h.cache.Get(key)

	if err != nil {
		if err == cache.ErrKeyNotFound {
			writeError(
				w,
				http.StatusNotFound,
				"key not found",
			)
			return
		}

		writeError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)

		return
	}

	writeJSON(
		w,
		http.StatusOK,
		response{
			Value: value,
		},
	)
}

func (h *Handler) set(
	w http.ResponseWriter,
	r *http.Request,
) {
	key := r.PathValue("key")

	if strings.TrimSpace(key) == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"key cannot be empty",
		)
		return
	}

	var request setRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	ttl := time.Duration(request.TTL) * time.Second

	if err := h.cache.Set(
		key,
		request.Value,
		ttl,
	); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	key := r.PathValue("key")

	if key == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"key is required",
		)
		return
	}

	if err := h.cache.Delete(key); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setMany(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request batchSetRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	for _, item := range request.Items {
		ttl := time.Duration(item.TTL) * time.Second

		if err := h.cache.Set(
			item.Key,
			item.Value,
			ttl,
		); err != nil {
			writeError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getMany(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request batchGetRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	items := make(
		map[string]json.RawMessage,
		len(request.Keys),
	)

	for _, key := range request.Keys {
		value, err := h.cache.Get(key)

		if err != nil {
			if err == cache.ErrKeyNotFound {
				continue
			}

			writeError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
			return
		}

		items[key] = json.RawMessage(value)
	}

	writeJSON(
		w,
		http.StatusOK,
		batchGetResponse{
			Items: items,
		},
	)
}

func (h *Handler) deleteMany(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request batchGetRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	for _, key := range request.Keys {
		if err := h.cache.Delete(key); err != nil {
			writeError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(
		w,
		status,
		response{
			Error: message,
		},
	)
}
