package diffagent

import (
	"fmt"
	"strings"
)

type diffOp struct {
	kind byte // ' ', '-', '+'
	text string
	oldPos int
	newPos int
}

const maxDiffCells = 4_000_000

// unifiedDiff builds a real unified diff using a longest-common-subsequence line
// diff. Inputs larger than maxDiffCells return an error rather than risking an
// unbounded allocation.
func unifiedDiff(original, modified, path string, contextLines int) (string, error) {
	oldLines := splitLines(original)
	newLines := splitLines(modified)
	if len(oldLines)*len(newLines) > maxDiffCells {
		return "", fmt.Errorf("input too large for diff")
	}
	if contextLines <= 0 {
		contextLines = 3
	}
	if contextLines > 10 {
		contextLines = 10
	}
	ops := diffOps(oldLines, newLines)
	numberOps(ops)
	hunks := buildHunks(ops, contextLines)
	if len(hunks) == 0 {
		return "```diff\n```", nil
	}
	if strings.TrimSpace(path) == "" {
		path = "file"
	}
	var out strings.Builder
	out.WriteString("```diff\n")
	fmt.Fprintf(&out, "--- a/%s\n+++ b/%s\n", path, path)
	for _, hunk := range hunks {
		oldCount, newCount := 0, 0
		for _, op := range hunk {
			if op.kind != '+' {
				oldCount++
			}
			if op.kind != '-' {
				newCount++
			}
		}
		oldStart := hunk[0].oldPos
		newStart := hunk[0].newPos
		fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
		for _, op := range hunk {
			out.WriteByte(op.kind)
			out.WriteString(op.text)
			out.WriteByte('\n')
		}
	}
	out.WriteString("```")
	return out.String(), nil
}

func splitLines(text string) []string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.TrimSuffix(normalized, "\n")
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "\n")
}

func diffOps(oldLines, newLines []string) []diffOp {
	n, m := len(oldLines), len(newLines)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	ops := make([]diffOp, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case oldLines[i] == newLines[j]:
			ops = append(ops, diffOp{kind: ' ', text: oldLines[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{kind: '-', text: oldLines[i]})
			i++
		default:
			ops = append(ops, diffOp{kind: '+', text: newLines[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{kind: '-', text: oldLines[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{kind: '+', text: newLines[j]})
	}
	return ops
}

func numberOps(ops []diffOp) {
	oldLine, newLine := 1, 1
	for i := range ops {
		ops[i].oldPos = oldLine
		ops[i].newPos = newLine
		switch ops[i].kind {
		case ' ':
			oldLine++
			newLine++
		case '-':
			oldLine++
		case '+':
			newLine++
		}
	}
}

func buildHunks(ops []diffOp, contextLines int) [][]diffOp {
	changeIndexes := make([]int, 0)
	for i, op := range ops {
		if op.kind != ' ' {
			changeIndexes = append(changeIndexes, i)
		}
	}
	if len(changeIndexes) == 0 {
		return nil
	}
	type span struct{ start, end int }
	spans := make([]span, 0)
	for _, index := range changeIndexes {
		start := max(0, index-contextLines)
		end := min(len(ops)-1, index+contextLines)
		if len(spans) > 0 && start <= spans[len(spans)-1].end+1 {
			spans[len(spans)-1].end = end
			continue
		}
		spans = append(spans, span{start: start, end: end})
	}
	hunks := make([][]diffOp, 0, len(spans))
	for _, s := range spans {
		hunks = append(hunks, ops[s.start:s.end+1])
	}
	return hunks
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
