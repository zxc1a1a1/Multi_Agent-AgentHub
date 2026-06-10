package registry

import (
	"context"
	"testing"
)

func TestDispatchResolverAgentInfosFiltersUnavailableDynamicAgents(t *testing.T) {
	static, err := NewStaticAgentRegistry([]AgentEndpoint{{Name: "static-agent", URL: "http://static", Description: "static", CapabilityIDs: []string{"static_cap"}}})
	if err != nil {
		t.Fatalf("NewStaticAgentRegistry: %v", err)
	}
	dyn := newTestDynamicRegistry(t)
	ctx := context.Background()

	healthyURL := fakeAgentServer(t, "healthy-agent").URL
	if _, err := dyn.Register(ctx, RegisterAgentRequest{URL: healthyURL}); err != nil {
		t.Fatalf("register healthy: %v", err)
	}
	if _, err := dyn.Check(ctx, "healthy-agent"); err != nil {
		t.Fatalf("check healthy: %v", err)
	}

	disabledURL := fakeAgentServer(t, "disabled-agent").URL
	if _, err := dyn.Register(ctx, RegisterAgentRequest{URL: disabledURL}); err != nil {
		t.Fatalf("register disabled: %v", err)
	}
	disabled := false
	if _, err := dyn.Update(ctx, "disabled-agent", UpdateAgentRequest{Enabled: &disabled}); err != nil {
		t.Fatalf("disable dynamic: %v", err)
	}

	unhealthyURL := fakeAgentServerWithError(t).URL
	// Store an unhealthy dynamic agent directly because Register requires a valid card.
	if err := dyn.store.Upsert(ctx, RegisteredAgent{Name: "unhealthy-agent", BaseURL: unhealthyURL, Source: AgentSourceDynamic, Enabled: true, Healthy: false, Capabilities: []string{"bad_cap"}}); err != nil {
		t.Fatalf("upsert unhealthy: %v", err)
	}

	infos, err := NewDispatchResolver(static, dyn).AgentInfos(ctx)
	if err != nil {
		t.Fatalf("AgentInfos: %v", err)
	}
	seen := map[string]bool{}
	for _, info := range infos {
		seen[info.Name] = true
	}
	if !seen["static-agent"] || !seen["healthy-agent"] {
		t.Fatalf("expected static and healthy dynamic agents, got %#v", seen)
	}
	if seen["disabled-agent"] {
		t.Fatal("disabled dynamic agent must not be visible to planner AgentInfos")
	}
	if seen["unhealthy-agent"] {
		t.Fatal("unhealthy dynamic agent must not be visible to planner AgentInfos")
	}
}

func TestDispatchResolverAgentInfosExcludesUnhealthyStaticAgents(t *testing.T) {
	static, err := NewStaticAgentRegistry([]AgentEndpoint{
		{Name: "healthy-static", URL: "http://healthy-static:8080", Description: "healthy", CapabilityIDs: []string{"cap1"}},
		{Name: "unhealthy-static", URL: "http://unhealthy-static:8080", Description: "unhealthy", CapabilityIDs: []string{"cap2"}},
	})
	if err != nil {
		t.Fatalf("NewStaticAgentRegistry: %v", err)
	}

	// Attach a HealthChecker with one agent marked unhealthy.
	hc := &HealthChecker{
		healthy: map[string]bool{
			"healthy-static":   true,
			"unhealthy-static": false,
		},
		endpoints: map[string]string{
			"healthy-static":   "http://healthy-static:8080",
			"unhealthy-static": "http://unhealthy-static:8080",
		},
	}
	static.SetHealthChecker(hc)

	dyn := newTestDynamicRegistry(t)
	infos, err := NewDispatchResolver(static, dyn).AgentInfos(context.Background())
	if err != nil {
		t.Fatalf("AgentInfos: %v", err)
	}
	seen := map[string]bool{}
	for _, info := range infos {
		seen[info.Name] = true
	}
	if !seen["healthy-static"] {
		t.Fatal("healthy static agent must be visible to AgentInfos")
	}
	if seen["unhealthy-static"] {
		t.Fatal("unhealthy static agent must not be visible to AgentInfos")
	}

	// Names and ResolveURL must also respect static health for consistency.
	names, err := NewDispatchResolver(static, dyn).Names(context.Background())
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["healthy-static"] {
		t.Error("Names must include healthy static agent")
	}
	if nameSet["unhealthy-static"] {
		t.Error("Names must exclude unhealthy static agent")
	}

	url, ok, err := NewDispatchResolver(static, dyn).ResolveURL(context.Background(), "healthy-static")
	if err != nil || !ok || url != "http://healthy-static:8080" {
		t.Fatalf("ResolveURL healthy-static: ok=%v err=%v url=%q", ok, err, url)
	}
	_, ok, _ = NewDispatchResolver(static, dyn).ResolveURL(context.Background(), "unhealthy-static")
	if ok {
		t.Fatal("ResolveURL must not resolve unhealthy static agent")
	}
}
