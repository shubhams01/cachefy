package http
package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shubhams01/cachefy/internal/cache"
)

func newTestHandler() http.Handler {
	c := cache.New(100)

	return NewHandler(c)
}

func TestHealth(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}
}

func TestSetAndGet(t *testing.T) {
	handler := newTestHandler()

	body := bytes.NewBufferString(`{
		"value": "cachefy",
		"ttl": 60
	}`)

	setRequest := httptest.NewRequest(
		http.MethodPut,
		"/cache/name",
		body,
	)

	setRecorder := httptest.NewRecorder()

	handler.ServeHTTP(setRecorder, setRequest)

	if setRecorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			setRecorder.Code,
		)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/cache/name",
		nil,
	)

	getRecorder := httptest.NewRecorder()

	handler.ServeHTTP(getRecorder, getRequest)

	if getRecorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			getRecorder.Code,
		)
	}

	expected := `{"value":"cachefy"}`

	if getRecorder.Body.String() != expected+"\n" {
		t.Fatalf(
			"expected %q, got %q",
			expected+"\n",
			getRecorder.Body.String(),
		)
	}
}

func TestGetMissing(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(
		http.MethodGet,
		"/cache/missing",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}

func TestDelete(t *testing.T) {
	handler := newTestHandler()

	body := bytes.NewBufferString(`{
		"value": "cachefy"
	}`)

	setRequest := httptest.NewRequest(
		http.MethodPut,
		"/cache/name",
		body,
	)

	handler.ServeHTTP(
		httptest.NewRecorder(),
		setRequest,
	)

	deleteRequest := httptest.NewRequest(
		http.MethodDelete,
		"/cache/name",
		nil,
	)

	deleteRecorder := httptest.NewRecorder()

	handler.ServeHTTP(deleteRecorder, deleteRequest)

	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			deleteRecorder.Code,
		)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/cache/name",
		nil,
	)

	getRecorder := httptest.NewRecorder()

	handler.ServeHTTP(getRecorder, getRequest)

	if getRecorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			getRecorder.Code,
		)
	}
}

func TestInvalidJSON(t *testing.T) {
	handler := newTestHandler()

	body := bytes.NewBufferString(`invalid`)

	req := httptest.NewRequest(
		http.MethodPut,
		"/cache/name",
		body,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}