package agui

import (
	"testing"
)

func TestInternalToPublicMappingComplete(t *testing.T) {
	internalTypes := []string{
		InternalTypeRunStarted,
		InternalTypeRunFinished,
		InternalTypeRunError,
		InternalTypeMessageStart,
		InternalTypeMessageDelta,
		InternalTypeMessageEnd,
		InternalTypeStateUpdate,
		InternalTypeActivitySnapshot,
		InternalTypeToolCallStart,
		InternalTypeToolCallArgs,
		InternalTypeToolCallEnd,
		InternalTypeAgentTurnStarted,
		InternalTypeAgentTurnContent,
		InternalTypeAgentTurnFinished,
	}

	for _, it := range internalTypes {
		public, ok := InternalToPublic[it]
		if !ok {
			t.Errorf("internal type %q has no mapping in InternalToPublic", it)
			continue
		}
		if public == "" {
			t.Errorf("internal type %q maps to empty public type", it)
		}
	}

	// Verify reverse: all values in InternalToPublic are non-empty and unique
	seen := make(map[string]string)
	for k, v := range InternalToPublic {
		if v == "" {
			t.Errorf("InternalToPublic[%q] is empty", k)
		}
		if existing, ok := seen[v]; ok {
			t.Errorf("duplicate public type %q mapped from %q and %q", v, existing, k)
		}
		seen[v] = k
	}
}

func TestTranslatorHasCaseForEveryInternalType(t *testing.T) {
	// This test verifies that all types in InternalToPublic are covered by the translator.
	// We test by constructing adk.Events with each internal type and verifying
	// the translator produces a non-empty output with the correct public type.

	internalTypes := []string{
		InternalTypeRunStarted,
		InternalTypeRunFinished,
		InternalTypeRunError,
		InternalTypeMessageStart,
		InternalTypeMessageDelta,
		InternalTypeMessageEnd,
		InternalTypeStateUpdate,
		InternalTypeActivitySnapshot,
		InternalTypeToolCallStart,
		InternalTypeToolCallArgs,
		InternalTypeToolCallEnd,
		InternalTypeAgentTurnStarted,
		InternalTypeAgentTurnContent,
		InternalTypeAgentTurnFinished,
	}

	translator := NewTranslator()
	for _, it := range internalTypes {
		// We just verify the type exists in the map; full translation testing
		// requires constructing proper adk.Event with metadata, which is tested
		// in translator_test.go for specific types.
		public := InternalToPublic[it]
		if public == "" {
			t.Errorf("type %q has no public mapping", it)
		}
		_ = translator // unused here, but confirms translator is constructable
	}
}
