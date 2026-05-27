package registry

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

type fakeModel struct{}

func (fakeModel) Generate(context.Context, *adk.GenerateRequest) (*adk.GenerateResponse, error) {
	return &adk.GenerateResponse{}, nil
}

func (fakeModel) GenerateStream(context.Context, *adk.GenerateRequest) iter.Seq2[*adk.GenerateResponse, error] {
	return func(func(*adk.GenerateResponse, error) bool) {}
}

type fakeModelProvider struct {
	name   string
	create func(ctx context.Context, cfg ModelConfig) (adk.Model, error)
}

func (p fakeModelProvider) Name() string {
	return p.name
}

func (p fakeModelProvider) CreateModel(ctx context.Context, cfg ModelConfig) (adk.Model, error) {
	if p.create != nil {
		return p.create(ctx, cfg)
	}
	return fakeModel{}, nil
}

func TestRegisterModelProvider(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	if err := RegisterModelProvider(fakeModelProvider{name: "mock"}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	got := ListModelProviders()
	if !reflect.DeepEqual(got, []string{"mock"}) {
		t.Fatalf("provider list mismatch, got %v", got)
	}
}

func TestRegisterModelProvider_Invalid(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	if err := RegisterModelProvider(nil); err == nil {
		t.Fatalf("expected nil provider error")
	}
	if err := RegisterModelProvider(fakeModelProvider{name: ""}); err == nil {
		t.Fatalf("expected empty provider name error")
	}
}

func TestRegisterModelProvider_Duplicate(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	if err := RegisterModelProvider(fakeModelProvider{name: "mock"}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	if err := RegisterModelProvider(fakeModelProvider{name: "mock"}); err == nil {
		t.Fatalf("expected duplicate provider error")
	}
}

func TestRegisterModelInstance(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	if err := RegisterModelInstance("main", fakeModel{}); err != nil {
		t.Fatalf("register model instance: %v", err)
	}
	if got := ListModelInstances(); !reflect.DeepEqual(got, []string{"main"}) {
		t.Fatalf("instance list mismatch, got %v", got)
	}
}

func TestRegisterModelInstance_Invalid(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	if err := RegisterModelInstance("", fakeModel{}); err == nil {
		t.Fatalf("expected empty name error")
	}
	if err := RegisterModelInstance("main", nil); err == nil {
		t.Fatalf("expected nil model error")
	}
}

func TestResolveModel(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	want := fakeModel{}
	if err := RegisterModelInstance("main", want); err != nil {
		t.Fatalf("register model: %v", err)
	}

	got, err := ResolveModel(context.Background(), "main")
	if err != nil {
		t.Fatalf("resolve model: %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil model")
	}
}

func TestResolveModel_NotFound(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	if _, err := ResolveModel(context.Background(), "missing"); err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestCreateAndRegisterModel(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	if err := RegisterModelProvider(fakeModelProvider{name: "mock"}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	cfg := ModelConfig{
		Name:     "main",
		Provider: "mock",
		Model:    "model-1",
	}
	created, err := CreateAndRegisterModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("create and register model: %v", err)
	}
	if created == nil {
		t.Fatalf("expected created model")
	}

	resolved, err := ResolveModel(context.Background(), "main")
	if err != nil {
		t.Fatalf("resolve created model: %v", err)
	}
	if resolved == nil {
		t.Fatalf("expected resolved model")
	}
}

func TestCreateAndRegisterModel_ProviderNotFound(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	_, err := CreateAndRegisterModel(context.Background(), ModelConfig{
		Name:     "main",
		Provider: "missing",
	})
	if err == nil {
		t.Fatalf("expected provider-not-found error")
	}
}

func TestCreateAndRegisterModel_ProviderError(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	expectedErr := errors.New("create failed")
	if err := RegisterModelProvider(fakeModelProvider{
		name: "mock",
		create: func(context.Context, ModelConfig) (adk.Model, error) {
			return nil, expectedErr
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	_, err := CreateAndRegisterModel(context.Background(), ModelConfig{
		Name:     "main",
		Provider: "mock",
	})
	if err == nil {
		t.Fatalf("expected provider create error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped provider error, got %v", err)
	}
	if !strings.Contains(err.Error(), "mock") {
		t.Fatalf("expected provider name in error, got %v", err)
	}
}

func TestListModelProviders_ReturnsCopy(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	if err := RegisterModelProvider(fakeModelProvider{name: "mock"}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	got := ListModelProviders()
	got[0] = "mutated"

	again := ListModelProviders()
	if !reflect.DeepEqual(again, []string{"mock"}) {
		t.Fatalf("provider list should remain unchanged, got %v", again)
	}
}

func TestListModelInstances_ReturnsCopy(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	if err := RegisterModelInstance("main", fakeModel{}); err != nil {
		t.Fatalf("register model: %v", err)
	}

	got := ListModelInstances()
	got[0] = "mutated"

	again := ListModelInstances()
	if !reflect.DeepEqual(again, []string{"main"}) {
		t.Fatalf("instance list should remain unchanged, got %v", again)
	}
}

func TestModelRegistry_ConcurrentAccess(t *testing.T) {
	resetModelRegistryForTest()
	t.Cleanup(resetModelRegistryForTest)

	const n = 50
	var wg sync.WaitGroup
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := fmt.Sprintf("provider-%d", i)
			if err := RegisterModelProvider(fakeModelProvider{name: name}); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent provider register failed: %v", err)
	}

	errCh = make(chan error, n)
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := CreateAndRegisterModel(context.Background(), ModelConfig{
				Name:     fmt.Sprintf("model-%d", i),
				Provider: fmt.Sprintf("provider-%d", i),
				Model:    "fake",
			})
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent model create/register failed: %v", err)
	}

	errCh = make(chan error, n)
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ResolveModel(context.Background(), fmt.Sprintf("model-%d", i)); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent resolve failed: %v", err)
	}
}
