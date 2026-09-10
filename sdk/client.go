package cachefy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Option func(*Client)

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func NewClient(baseURL string, options ...Option) *Client {
	client := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	for _, option := range options {
		option(client)
	}

	return client
}

type setRequest struct {
	Value json.RawMessage `json:"value"`
	TTL   int64           `json:"ttl"`
}

type getResponse struct {
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

func (c *Client) Set(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	body := setRequest{
		Value: data,
		TTL:   ttlSeconds(ttl),
	}

	return c.doJSON(
		ctx,
		http.MethodPut,
		c.keyURL(key),
		body,
		http.StatusNoContent,
		nil,
	)
}

func (c *Client) Get(
	ctx context.Context,
	key string,
) (json.RawMessage, error) {
	var result getResponse

	err := c.doJSON(
		ctx,
		http.MethodGet,
		c.keyURL(key),
		nil,
		http.StatusOK,
		&result,
	)

	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return nil, ErrKeyNotFound
		}

		return nil, err
	}

	return result.Value, nil
}

func Get[T any](
	ctx context.Context,
	c *Client,
	key string,
) (T, error) {
	var zero T

	data, err := c.Get(ctx, key)

	if err != nil {
		return zero, err
	}

	var value T

	if err := json.Unmarshal(data, &value); err != nil {
		return zero, fmt.Errorf("unmarshal cached value: %w", err)
	}

	return value, nil
}

func Set[T any](
	ctx context.Context,
	c *Client,
	key string,
	value T,
	ttl time.Duration,
) error {
	return c.Set(ctx, key, value, ttl)
}

func (c *Client) Delete(
	ctx context.Context,
	key string,
) error {
	return c.doJSON(
		ctx,
		http.MethodDelete,
		c.keyURL(key),
		nil,
		http.StatusNoContent,
		nil,
	)
}

func (c *Client) Exists(
	ctx context.Context,
	key string,
) (bool, error) {
	_, err := c.Get(ctx, key)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, ErrKeyNotFound) {
		return false, nil
	}

	return false, err
}

func (c *Client) SetMany(
	ctx context.Context,
	items map[string]any,
	ttl time.Duration,
) error {
	request := batchSetRequest{
		Items: make([]batchSetItem, 0, len(items)),
	}

	for key, value := range items {
		data, err := json.Marshal(value)

		if err != nil {
			return fmt.Errorf(
				"marshal value for key %q: %w",
				key,
				err,
			)
		}

		request.Items = append(
			request.Items,
			batchSetItem{
				Key:   key,
				Value: data,
				TTL:   ttlSeconds(ttl),
			},
		)
	}

	return c.doJSON(
		ctx,
		http.MethodPost,
		c.baseURL+"/cache/batch",
		request,
		http.StatusNoContent,
		nil,
	)
}

func (c *Client) GetMany(
	ctx context.Context,
	keys []string,
) (map[string]json.RawMessage, error) {
	request := batchGetRequest{
		Keys: keys,
	}

	var response batchGetResponse

	err := c.doJSON(
		ctx,
		http.MethodPost,
		c.baseURL+"/cache/batch/get",
		request,
		http.StatusOK,
		&response,
	)

	if err != nil {
		return nil, err
	}

	return response.Items, nil
}

func (c *Client) DeleteMany(
	ctx context.Context,
	keys []string,
) error {
	request := batchGetRequest{
		Keys: keys,
	}

	return c.doJSON(
		ctx,
		http.MethodPost,
		c.baseURL+"/cache/batch/delete",
		request,
		http.StatusNoContent,
		nil,
	)
}

func (c *Client) Health(ctx context.Context) error {
	return c.doJSON(
		ctx,
		http.MethodGet,
		c.baseURL+"/health",
		nil,
		http.StatusOK,
		nil,
	)
}

func (c *Client) keyURL(key string) string {
	return fmt.Sprintf(
		"%s/cache/%s",
		c.baseURL,
		url.PathEscape(key),
	)
}

func (c *Client) doJSON(
	ctx context.Context,
	method string,
	targetURL string,
	body any,
	expectedStatus int,
	result any,
) error {
	var reader io.Reader

	if body != nil {
		data, err := json.Marshal(body)

		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}

		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		targetURL,
		reader,
	)

	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return c.parseError(resp)
	}

	if result == nil {
		return nil
	}

	if err := json.NewDecoder(
		resp.Body,
	).Decode(result); err != nil {
		return fmt.Errorf(
			"decode response: %w",
			err,
		)
	}

	return nil
}

func (c *Client) parseError(
	resp *http.Response,
) error {
	body, err := io.ReadAll(resp.Body)

	if err == nil {
		var result getResponse

		if json.Unmarshal(body, &result) == nil &&
			result.Error != "" {
			if resp.StatusCode == http.StatusNotFound {
				return ErrKeyNotFound
			}

			return errors.New(result.Error)
		}
	}

	return fmt.Errorf(
		"cachefy request failed with status %d",
		resp.StatusCode,
	)
}

func ttlSeconds(ttl time.Duration) int64 {
	if ttl <= 0 {
		return 0
	}

	return int64(ttl / time.Second)
}
