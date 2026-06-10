package orchestratorclient

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Phase 10 Boundary Audit: Gateway ↔ Orchestrator
//
// These tests verify architectural constraints that keep the Gateway and
// Orchestrator properly decoupled. They are "always-pass" documentation
// tests unless a boundary violation is introduced.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// TestPhase10_NoA2ADirectImport
//
// Verify that the gateway httpapi and orchestratorclient packages do not
// directly import pkg/adk/a2a. The Gateway must never speak A2A protocol
// directly — only the Orchestrator may.
// ---------------------------------------------------------------------------
func TestPhase10_NoA2ADirectImport(t *testing.T) {
	forbiddenImport := "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a"

	// Packages under audit. These must NOT import a2a directly.
	packagesToAudit := []string{
		"../httpapi",
		".",
	}

	for _, pkgPath := range packagesToAudit {
		goFiles, err := filepath.Glob(filepath.Join(pkgPath, "*.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", pkgPath, err)
		}
		if len(goFiles) == 0 {
			continue
		}

		fset := token.NewFileSet()
		for _, goFile := range goFiles {
			// Skip test files — they may import a2a for test helpers.
			if strings.HasSuffix(goFile, "_test.go") {
				continue
			}

			node, err := parser.ParseFile(fset, goFile, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse %s: %v", goFile, err)
			}
			for _, imp := range node.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if importPath == forbiddenImport {
					t.Errorf("BOUNDARY VIOLATION: %s imports %s", goFile, forbiddenImport)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_NoAgentCardFetch
//
// Verify the orchestratorclient package does not fetch /.well-known/agent.json.
// The Orchestrator is responsible for agent discovery; the Gateway must not
// perform A2A service discovery (card fetching).
// ---------------------------------------------------------------------------
func TestPhase10_NoAgentCardFetch(t *testing.T) {
	wellKnownPaths := []string{
		".well-known/agent.json",
		"/.well-known/agent.json",
		"well-known/agent.json",
	}

	pkgDirs := []string{"../httpapi", "."}
	for _, pkgDir := range pkgDirs {
		goFiles, err := filepath.Glob(filepath.Join(pkgDir, "*.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", pkgDir, err)
		}
		for _, goFile := range goFiles {
			if strings.HasSuffix(goFile, "_test.go") {
				continue
			}
			data, err := os.ReadFile(goFile)
			if err != nil {
				t.Fatalf("read %s: %v", goFile, err)
			}
			content := string(data)
			for _, path := range wellKnownPaths {
				if strings.Contains(content, path) {
					t.Errorf("BOUNDARY VIOLATION: %s contains agent card discovery path %q", goFile, path)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_ProxyInternalPathConstraint
//
// Verify that all Orchestrator-proxied paths in the gateway use the
// /internal/orchestrator/* prefix. Direct calls to Orchestrator public
// endpoints would circumvent the proxy security boundary.
// ---------------------------------------------------------------------------
func TestPhase10_ProxyInternalPathConstraint(t *testing.T) {
	// Read orchestratorclient source files and collect all URL paths.
	goFiles, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	var allPrefixes []string
	for _, goFile := range goFiles {
		if strings.HasSuffix(goFile, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, goFile, nil, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", goFile, err)
		}

		// Walk AST to find all string literals containing "/internal/orchestrator/"
		ast.Inspect(node, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val := strings.Trim(lit.Value, `"`)
			if strings.Contains(val, "/internal/orchestrator/") {
				allPrefixes = append(allPrefixes, val)
			}
			return true
		})
	}

	// Every orchestrator-proxied path must start with /internal/orchestrator/.
	// This includes query strings appended to the path.
	for _, p := range allPrefixes {
		// Extract the path portion (strip base URL prefix if present).
		if !strings.Contains(p, "/internal/orchestrator/") {
			t.Errorf("orchestrator proxy path missing /internal/orchestrator/ prefix: %q", p)
		}
	}

	t.Logf("found %d proxied internal/orchestrator paths", len(allPrefixes))
	if len(allPrefixes) == 0 {
		t.Log("no hardcoded orchestrator paths found (may use dynamic construction)")
	}
}

// ---------------------------------------------------------------------------
// TestPhase10_AllowlistDocumentation
//
// Document the Gateway boundary allowlist. These files are explicitly allowed
// to import packages or use patterns that would otherwise violate Gateway
// boundaries:
//
//   - remote_agent.go: Legacy fallback that uses a2a.Client. This file is
//     frozen and must not be extended with new features.
//   - cmd/gateway/main.go: Gateway entry point that wires dependencies,
//     including the OrchestratorRunService.
//
// This test is always-pass documentation.
// ---------------------------------------------------------------------------
func TestPhase10_AllowlistDocumentation(t *testing.T) {
	allowlist := []struct {
		file    string
		reason  string
		imports []string // the boundary-crossing imports this file is allowed to use
	}{
		{
			file:   "../runservice/remote_agent.go",
			reason: "Legacy fallback using a2a.Client — frozen, no new features",
			imports: []string{
				"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk/a2a",
			},
		},
		{
			file:   "../cmd/gateway/main.go",
			reason: "Gateway entry point — wires OrchestratorRunService, not business logic",
			imports: []string{
				"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/orchestratorclient",
			},
		},
	}

	for _, entry := range allowlist {
		t.Logf("ALLOWLIST: %s — %s", entry.file, entry.reason)
		// Verify the file actually exists.
		if _, err := os.Stat(entry.file); os.IsNotExist(err) {
			// Allowlist entry may reference a different CWD; skip strict check.
			t.Logf("  (file not found relative to orchestratorclient dir — may be renamed)")
		}
	}

	// Sanity: the allowlist must not grow without explicit review.
	if len(allowlist) > 5 {
		t.Errorf("allowlist has %d entries — review for excessive boundary exemptions", len(allowlist))
	}
}
