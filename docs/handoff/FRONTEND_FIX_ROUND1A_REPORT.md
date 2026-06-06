# Frontend Fix Round 1A — Repair Report

Date: 2026-06-06
Branch: `g`
Status: **All fixes implemented and verified**

## Summary

7 frontend fixes applied. No backend changes. 49/49 tests pass. TypeScript compilation clean. Production build succeeds.

## Fix 1: Auto-Orchestration (Default Agent)

**Problem:** Users had to manually select an agent. The orchestrator supports keyword-based auto-routing, but the frontend always sent `agentName: 'code-agent'`.

**Fix:**
- Changed `DEFAULT_AGENT_NAME` from `'code-agent'` to `'auto'` in `frontend/src/lib/agents.ts`
- Added `'auto'` to `AGENT_OPTIONS` as first entry with display name "Auto (Smart)"
- Added `isConcreteAgentName()` guard function to distinguish 'auto' from concrete agents ('code-agent', 'web-agent')
- In `messageStore.sendMessage()`: only add `agentName` to request body when it's a concrete agent name (not 'auto')

**Affected files:** `agents.ts`, `messageStore.ts`

## Fix 2: Loading / Streaming Perception

**Problem:** During agent response streaming (1-2 seconds for mock agents, potentially much longer for real LLMs), the UI showed only "Start a conversation" empty state, making it appear unresponsive.

**Fix:**
- Added "Thinking…" bouncing-dot indicator in `ChatWindow.tsx` that appears during `streaming` state
- Enhanced the empty-content animation in `StreamingText.tsx` from simple "Thinking..." text to animated bouncing dots
- Changed scroll behavior from always-smooth to instant-scroll during streaming (avoids scroll lag that can truncate content display)
- Added `!streaming` condition to the empty-state guard so "Start a conversation" doesn't show while the agent is working

**Affected files:** `ChatWindow.tsx`, `StreamingText.tsx`

## Fix 3: Web Preview Rendering (iframe sandbox)

**Problem:** Web preview showed raw HTML in a `<pre><code>` block with no visual rendering. Users couldn't see the actual HTML output.

**Fix:**
- Rewrote `WebPreview.tsx` to render HTML in a sandbox `<iframe>` with `srcDoc` and `sandbox="allow-scripts"` (no `allow-same-origin` to prevent cookie/token access)
- Added Preview/Source tab toggle (Eye/Code icons from lucide-react)
- Added Expand/Collapse button for full-screen preview
- Line count display
- Graceful empty state ("No HTML to preview")
- Added web-agent-only gate in `messageStore.ts`: `appendWebPreviewFromMessageContent` now checks `isConcreteAgentName(currentAgentName) && currentAgentName === 'web-agent'` before extracting HTML preview blocks

**Affected files:** `WebPreview.tsx`, `messageStore.ts`

## Fix 4: Code Block Copy Button

**Problem:** Code blocks in agent responses had no copy functionality.

**Fix:**
- Added `CodeBlockHeader` component to `StreamingText.tsx` with Copy/Check icons
- Clipboard API copy with `execCommand('copy')` fallback for older browsers
- Auto-extracts language tag from Markdown code fence for display
- "Copied" confirmation with 2-second auto-reset

**Affected files:** `StreamingText.tsx`

## Fix 5: Session Title Auto-Generation

**Problem:** Conversations showed generic titles (or conversation IDs) instead of meaningful names.

**Fix:**
- Added `updateTitle()` method to `conversationStore`
- Added `generateConversationTitle()` utility (truncates first message to 30 chars, strips newlines/collapses whitespace)
- In `messageStore.sendMessage()`: auto-generates title from first user message and calls `updateTitle()` on the conversation (only for the first message per conversation)

**Affected files:** `conversationStore.ts`, `messageStore.ts`

**Limitation:** Titles are client-side only and are lost on page refresh (Gateway does not currently support title persistence). This is a session-level improvement.

## Fix 6: Orchestration Card Collapse

**Problem:** The orchestration card ("编排分析") was always expanded and took significant vertical space, cluttering the chat view.

**Fix:**
- Rewrote `OrchestrationCard.tsx` to default-collapsed with click-to-expand
- Collapsed view shows a compact one-line status summary: "单 Agent · 1 个任务" or "多 Agent 协作 · 2 个任务"
- Expandable via ChevronRight/ChevronDown icon (only clickable when there are details)
- Detail area shows intent, planner source, planner model, and reasoning

**Affected files:** `OrchestrationCard.tsx`

## Fix 7: Long Conversation Truncation Investigation

**Problem:** Users reported conversation content appearing truncated.

**Findings:**
- No explicit client-side truncation found (no message count limit, no content length cutoff)
- Root cause suspected: smooth-scroll lag during streaming — `scrollIntoView({ behavior: 'smooth' })` during rapid text streaming can cause the viewport to not reach the bottom, making the latest content appear "truncated" when it's actually just scrolled off-screen
- **Fix applied:** Instant scroll (`behavior: 'auto'`) during streaming, smooth scroll after streaming completes

**Affected files:** `ChatWindow.tsx`

## Verification Results

```
Test Files:  7 passed (7)
Tests:       49 passed (49)
TypeScript:  Clean (no errors)
Build:       Success (vite v6.4.2, 2050 modules)
```

Test files:
- `src/agui/client.test.ts` — 6 tests
- `src/stores/agentStore.test.ts` — 2 tests
- `src/stores/messageReplay.test.ts` — 5 tests
- `src/stores/messageStore.test.ts` — 17 tests
- `src/components/WebPreview.test.tsx` — 3 tests
- `src/components/CodePreview.test.tsx` — 7 tests
- `src/components/MessageBubble.test.tsx` — 9 tests

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/lib/agents.ts` | +15/-1 — Added 'auto' agent, `isConcreteAgentName` |
| `frontend/src/stores/messageStore.ts` | +31/-1 — Auto-orchestration, title gen, web preview boundary |
| `frontend/src/stores/conversationStore.ts` | +9 — `updateTitle()` method |
| `frontend/src/components/ChatWindow.tsx` | +21/-3 — Thinking indicator, instant scroll, empty-state guard |
| `frontend/src/components/StreamingText.tsx` | +82/-2 — Code block copy, enhanced animation |
| `frontend/src/components/WebPreview.tsx` | +77/-5 — Sandbox iframe, tabs, expand |
| `frontend/src/components/OrchestrationCard.tsx` | Rewritten — Default collapsed |
| `frontend/src/components/WebPreview.test.tsx` | Updated for new component |
| `frontend/src/components/MessageBubble.test.tsx` | Updated for new header text |

## Known Limitations

1. **Session titles** are ephemeral (lost on refresh) — Gateway does not persist conversation title updates.
2. **Auto-orchestration** relies on orchestrator keyword-matching which is basic. Not tested with real LLM planning.
3. **Web Preview sandbox** allows scripts (`allow-scripts`) but not same-origin. Cross-origin iframe isolation is correct for security but limits interactivity (no Gateway API access from previewed HTML).
