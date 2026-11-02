package diffagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestUnifiedDiffMultipleHunksAndContext(t *testing.T) {
	original := "line1\nline2\nline3\nline4\nline5\n"
	modified := "line1\nLINE2\nline3\nLINE4\nline5\n"
	diff, err := unifiedDiff(original, modified, "main.go", 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"--- a/main.go", "+++ b/main.go", "@@ -1,5 +1,5 @@",
		"-line2", "+LINE2", "-line4", "+LINE4", " line3",
	} {
		if !strings.Contains(diff, want) {
			t.Fatalf("diff missing %q:\n%s", want, diff)
		}
	}
	if strings.Count(diff, "\n@@") != 1 {
		t.Fatalf("nearby changes should merge into one hunk:\n%s", diff)
	}
}

func TestUnifiedDiffInsertAndDelete(t *testing.T) {
	inserted, err := unifiedDiff("a\nb\n", "x\na\nb\n", "f.txt", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(inserted, "@@ -1,2 +1,3 @@") || !strings.Contains(inserted, "+x") {
		t.Fatalf("insertion diff wrong:\n%s", inserted)
	}

	deleted, err := unifiedDiff("a\nb\n", "", "f.txt", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(deleted, "-a") || !strings.Contains(deleted, "-b") || !strings.Contains(deleted, "@@ -1,2 +1,0 @@") {
		t.Fatalf("deletion diff wrong:\n%s", deleted)
	}
}

func TestUnifiedDiffNoChanges(t *testing.T) {
	diff, err := unifiedDiff("same\ntext\n", "same\ntext\n", "f.txt", 3)
	if err != nil {
		t.Fatal(err)
	}
	if diff != "```diff\n```" {
		t.Fatalf("expected empty diff, got %q", diff)
	}
}

func TestGenerateDiffToolProducesMultiLineDiff(t *testing.T) {
	args, err := json.Marshal(map[string]any{
		"original": "one\ntwo\nthree\nfour\nfive\n",
		"modified": "one\nTWO\nthree\nFOUR\nfive\n",
		"filePath": "pkg/file.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := (GenerateDiffTool{}).Execute(context.Background(), args)
	if err != nil || result == nil || result.IsError {
		t.Fatalf("tool failed: %#v err=%v", result, err)
	}
	if !strings.Contains(result.Content, "-two") || !strings.Contains(result.Content, "+TWO") ||
		!strings.Contains(result.Content, "-four") || !strings.Contains(result.Content, "+FOUR") {
		t.Fatalf("tool did not produce a real diff:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "@@ -1,1 +1,1 @@") {
		t.Fatalf("tool still returns the one-line template:\n%s", result.Content)
	}
}
