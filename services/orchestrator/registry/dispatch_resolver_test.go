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
		{Name: "healthy-static", URL: "http://healthy", Description: "healthy", CapabilityIDs: []string{"ok"}},
		{Name: "unhealthy-static", URL: "http://unhealthy", Description: "bad", CapabilityIDs: []string{"bad"}},
	})
	if err != nil {
		t.Fatalf("NewStaticAgentRegistry: %v", err)
	}
	hc := NewHealthChecker(static.List(), HealthCheckerConfig{})
	hc.healthy["healthy-static"] = true
	hc.healthy["unhealthy-static"] = false
	static.SetHealthChecker(hc)

	infos, err := NewDispatchResolver(static, nil).AgentInfos(context.Background())
	if err != nil {
		t.Fatalf("AgentInfos: %v", err)
	}
	seen := map[string]bool{}
	for _, info := range infos {
		seen[info.Name] = true
	}
	if !seen["healthy-static"] {
		t.Fatalf("healthy static not visible: %#v", seen)
	}
	if seen["unhealthy-static"] {
		t.Fatalf("unhealthy static must not be visible to planner AgentInfos: %#v", seen)
	}
}
