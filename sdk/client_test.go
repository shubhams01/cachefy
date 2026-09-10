package cachefy

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shubhams01/cachefy/internal/cache"
	cachehttp "github.com/shubhams01/cachefy/internal/http"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func newTestServer(t *testing.T) *Client {
	t.Helper()

	c := cache.New(100)

	handler := cachehttp.NewHandler(c)

	server := httptest.NewServer(handler)

	t.Cleanup(func() {
		server.Close()
		c.Close()
	})

	return NewClient(server.URL)
}

func TestClientSetGet(t *testing.T) {
	client := newTestServer(t)

	ctx := context.Background()

	err := client.Set(
		ctx,
		"user",
		"shubham",
		time.Minute,
	)

	if err != nil {
		t.Fatal(err)
	}

	value, err := Get[string](
		ctx,
		client,
		"user",
	)

	if err != nil {
		t.Fatal(err)
	}

	if value != "shubham" {
		t.Fatalf(
			"expected shubham, got %s",
			value,
		)
	}
}

func TestClientTypedValue(t *testing.T) {
	client := newTestServer(t)

	ctx := context.Background()

	user := User{
		ID:   123,
		Name: "Shubham",
	}

	err := Set(
		ctx,
		client,
		"user:123",
		user,
		time.Minute,
	)

	if err != nil {
		t.Fatal(err)
	}

	result, err := Get[User](
		ctx,
		client,
		"user:123",
	)

	if err != nil {
		t.Fatal(err)
	}

	if result.ID != user.ID {
		t.Fatalf(
			"expected ID %d, got %d",
			user.ID,
			result.ID,
		)
	}

	if result.Name != user.Name {
		t.Fatalf(
			"expected name %s, got %s",
			user.Name,
			result.Name,
		)
	}
}

func TestClientSpecialCharactersInKey(t *testing.T) {
	client := newTestServer(t)

	ctx := context.Background()

	key := "user/profile:123?test=true"

	err := client.Set(
		ctx,
		key,
		"cachefy",
		time.Minute,
	)

	if err != nil {
		t.Fatal(err)
	}

	value, err := Get[string](
		ctx,
		client,
		key,
	)

	if err != nil {
		t.Fatal(err)
	}

	if value != "cachefy" {
		t.Fatalf(
			"expected cachefy, got %s",
			value,
		)
	}
}

func TestClientGetMissing(t *testing.T) {
	client := newTestServer(t)

	_, err := client.Get(
		context.Background(),
		"missing",
	)

	if err != ErrKeyNotFound {
		t.Fatalf(
			"expected ErrKeyNotFound, got %v",
			err,
		)
	}
}

func TestClientDelete(t *testing.T) {
	client := newTestServer(t)

	ctx := context.Background()

	err := client.Set(
		ctx,
		"user",
		"shubham",
		0,
	)

	if err != nil {
		t.Fatal(err)
	}

	err = client.Delete(ctx, "user")

	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Get(ctx, "user")

	if err != ErrKeyNotFound {
		t.Fatalf(
			"expected ErrKeyNotFound, got %v",
			err,
		)
	}
}

func TestClientExists(t *testing.T) {
	client := newTestServer(t)

	ctx := context.Background()

	exists, err := client.Exists(
		ctx,
		"user",
	)

	if err != nil {
		t.Fatal(err)
	}

	if exists {
		t.Fatal("key should not exist")
	}

	client.Set(
		ctx,
		"user",
		"shubham",
		0,
	)

	exists, err = client.Exists(
		ctx,
		"user",
	)

	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal("key should exist")
	}
}

func TestClientSetMany(t *testing.T) {
	client := newTestServer(t)

	ctx := context.Background()

	items := map[string]any{
		"user:1": "shubham",
		"user:2": "cachefy",
		"user:3": 123,
	}

	err := client.SetMany(
		ctx,
		items,
		time.Minute,
	)

	if err != nil {
		t.Fatal(err)
	}

	result, err := client.GetMany(
		ctx,
		[]string{
			"user:1",
			"user:2",
			"user:3",
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 3 {
		t.Fatalf(
			"expected 3 items, got %d",
			len(result),
		)
	}
}

func TestClientDeleteMany(t *testing.T) {
	client := newTestServer(t)

	ctx := context.Background()

	err := client.SetMany(
		ctx,
		map[string]any{
			"a": "A",
			"b": "B",
			"c": "C",
		},
		0,
	)

	if err != nil {
		t.Fatal(err)
	}

	err = client.DeleteMany(
		ctx,
		[]string{"a", "b", "c"},
	)

	if err != nil {
		t.Fatal(err)
	}

	result, err := client.GetMany(
		ctx,
		[]string{"a", "b", "c"},
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatalf(
			"expected 0 items, got %d",
			len(result),
		)
	}
}
