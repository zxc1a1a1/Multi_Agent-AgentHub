// Package registry provides health checking for agent endpoints.
package registry

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

// HealthChecker periodically probes agent /health endpoints and maintains a
// healthy/unhealthy view. It is safe for concurrent use.
type HealthChecker struct {
	mu       sync.RWMutex
	healthy  map[string]bool    // agent name → healthy
	endpoints map[string]string  // agent name → URL

	interval   time.Duration
	httpClient *http.Client
	stopCh     chan struct{}
	doneCh     chan struct{}
}

// HealthCheckerConfig configures the health checker.
type HealthCheckerConfig struct {
	Interval   time.Duration // probe interval (default 30s)
	HTTPClient *http.Client  // optional custom client
}

// NewHealthChecker creates a health checker for the given agent endpoints.
// Call Start() to begin probing, and Stop() to shut down.
func NewHealthChecker(endpoints []AgentEndpoint, cfg HealthCheckerConfig) *HealthChecker {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	hc := &HealthChecker{
		healthy:    make(map[string]bool, len(endpoints)),
		endpoints:  make(map[string]string, len(endpoints)),
		interval:   cfg.Interval,
		httpClient: cfg.HTTPClient,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}

	for _, ep := range endpoints {
		hc.endpoints[ep.Name] = ep.URL
		hc.healthy[ep.Name] = true // start optimistic
	}

	return hc
}

// Start begins periodic health probes in a background goroutine.
func (h *HealthChecker) Start() {
	go h.loop()
}

// Stop shuts down the health checker and waits for the goroutine to exit.
func (h *HealthChecker) Stop() {
	close(h.stopCh)
	<-h.doneCh
}

// IsHealthy reports whether the named agent is currently healthy.
func (h *HealthChecker) IsHealthy(name string) bool {
	if h == nil {
		return false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthy[name]
}

// HealthyEndpoints returns only the agent endpoints currently considered healthy.
func (h *HealthChecker) HealthyEndpoints() []AgentEndpoint {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []AgentEndpoint
	for name, url := range h.endpoints {
		if h.healthy[name] {
			result = append(result, AgentEndpoint{Name: name, URL: url})
		}
	}
	return result
}

// loop runs the periodic health probe.
func (h *HealthChecker) loop() {
	defer close(h.doneCh)

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	// Probe immediately on start.
	h.probeAll()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.probeAll()
		}
	}
}

// probeAll checks all registered endpoints.
func (h *HealthChecker) probeAll() {
	for name, url := range h.endpoints {
		healthy := h.probeOne(url)
		h.mu.Lock()
		prev := h.healthy[name]
		h.healthy[name] = healthy
		h.mu.Unlock()
		if prev != healthy {
			if healthy {
				log.Printf("[health] agent %q recovered", name)
			} else {
				log.Printf("[health] agent %q is unhealthy", name)
			}
		}
	}
}

// probeOne checks a single /health endpoint with a GET request.
func (h *HealthChecker) probeOne(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
