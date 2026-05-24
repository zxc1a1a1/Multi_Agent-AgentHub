package a2a

import (
	"context"
	"errors"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
)

func TestGetOrCreateClientCachesByAgentURL(t *testing.T) {
	c := &Client{
		clients: make(map[string]*a2aclient.Client),
	}

	createCalls := 0
	c.create = func(ctx context.Context, endpoints []*a2a.AgentInterface) (*a2aclient.Client, error) {
		createCalls++
		if len(endpoints) != 1 {
			t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
		}
		if endpoints[0].URL != "http://agent-a:8081" {
			t.Fatalf("unexpected endpoint url: %s", endpoints[0].URL)
		}
		return &a2aclient.Client{}, nil
	}

	first, err := c.getOrCreateClient(context.Background(), "http://agent-a:8081")
	if err != nil {
		t.Fatalf("first getOrCreateClient returned error: %v", err)
	}
	second, err := c.getOrCreateClient(context.Background(), "http://agent-a:8081")
	if err != nil {
		t.Fatalf("second getOrCreateClient returned error: %v", err)
	}

	if createCalls != 1 {
		t.Fatalf("expected create to be called once, got %d", createCalls)
	}
	if first != second {
		t.Fatalf("expected cached client instance to be reused")
	}
}

func TestGetOrCreateClientCreatesPerDifferentAgentURL(t *testing.T) {
	c := &Client{
		clients: make(map[string]*a2aclient.Client),
	}

	createCalls := 0
	c.create = func(ctx context.Context, endpoints []*a2a.AgentInterface) (*a2aclient.Client, error) {
		createCalls++
		return &a2aclient.Client{}, nil
	}

	if _, err := c.getOrCreateClient(context.Background(), "http://agent-a:8081"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := c.getOrCreateClient(context.Background(), "http://agent-b:8081"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createCalls != 2 {
		t.Fatalf("expected create to be called twice, got %d", createCalls)
	}
}

func TestGetOrCreateClientCreateErrorNotCached(t *testing.T) {
	c := &Client{
		clients: make(map[string]*a2aclient.Client),
	}

	createCalls := 0
	c.create = func(ctx context.Context, endpoints []*a2a.AgentInterface) (*a2aclient.Client, error) {
		createCalls++
		return nil, errors.New("dial failed")
	}

	_, err := c.getOrCreateClient(context.Background(), "http://agent-a:8081")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if createCalls != 1 {
		t.Fatalf("expected one create call, got %d", createCalls)
	}
	if len(c.clients) != 0 {
		t.Fatalf("expected failed create not to be cached")
	}
}
