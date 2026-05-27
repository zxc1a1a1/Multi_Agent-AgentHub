package model

import (
	"context"
	"iter"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/registry"
)

type fakeProvider struct{}

func (fakeProvider) Name() string {
	return "fake"
}

func (fakeProvider) CreateModel(context.Context, registry.ModelConfig) (adk.Model, error) {
	return fakeModel{}, nil
}

type fakeModel struct{}

func (fakeModel) Generate(context.Context, *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	return &adk.GenerateResponse{}, nil
}

func (fakeModel) GenerateStream(context.Context, *adk.GenerateRequest) iter.Seq2[*adk.GenerateResponse, error] {
	return func(func(*adk.GenerateResponse, error) bool) {}
}

var _ Provider = fakeProvider{}
var _ registry.ModelProvider = fakeProvider{}

var testModelSeq uint64

func TestProvider_InterfaceCompatibility(t *testing.T) {
	p := fakeProvider{}
	if p.Name() != "fake" {
		t.Fatalf("unexpected provider name: %s", p.Name())
	}
	m, err := p.CreateModel(context.Background(), registry.ModelConfig{Name: "m", Provider: "fake"})
	if err != nil {
		t.Fatalf("create model: %v", err)
	}
	if m == nil {
		t.Fatalf("expected non-nil model")
	}
}

func TestProvider_InitRegistration(t *testing.T) {
	providers := registry.ListModelProviders()
	for _, want := range []string{anthropicProviderName, openAIProviderName, proxyProviderName} {
		if !containsString(providers, want) {
			t.Fatalf("provider %q is not registered, providers=%v", want, providers)
		}
	}
}

func TestProvider_CreateAndRegisterModelViaRegistry(t *testing.T) {
	t.Setenv("TEST_ANTHROPIC_TOKEN", "test-token")
	t.Setenv("TEST_OPENAI_TOKEN", "test-token")
	t.Setenv("TEST_PROXY_TOKEN", "fake-token")

	cases := []registry.ModelConfig{
		{
			Name:      uniqueTestModelName("anthropic"),
			Provider:  anthropicProviderName,
			APIKeyEnv: "TEST_ANTHROPIC_TOKEN",
			Model:     "claude-test",
		},
		{
			Name:      uniqueTestModelName("openai"),
			Provider:  openAIProviderName,
			APIKeyEnv: "TEST_OPENAI_TOKEN",
			Model:     "gpt-test",
		},
		{
			Name:      uniqueTestModelName("proxy"),
			Provider:  proxyProviderName,
			APIKeyEnv: "TEST_PROXY_TOKEN",
			Model:     "proxy-model",
			BaseURL:   "http://127.0.0.1:18090/proxy",
		},
	}

	for _, cfg := range cases {
		cfg := cfg
		t.Run(cfg.Provider, func(t *testing.T) {
			model, err := registry.CreateAndRegisterModel(context.Background(), cfg)
			if err != nil {
				t.Fatalf("create and register model: %v", err)
			}
			if model == nil {
				t.Fatalf("expected non-nil model")
			}
			if _, ok := model.(adk.Model); !ok {
				t.Fatalf("registered model does not implement adk.Model: %T", model)
			}
		})
	}
}

func uniqueTestModelName(prefix string) string {
	next := atomic.AddUint64(&testModelSeq, 1)
	return prefix + "-test-" + strconv.FormatUint(next, 10)
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
