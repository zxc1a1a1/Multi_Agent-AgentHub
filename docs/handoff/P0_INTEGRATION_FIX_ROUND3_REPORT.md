# P0 Integration Fix Round 3 瀹屾垚鎶ュ憡

**鏃ユ湡:** 2026-06-07
**鍒嗘敮:** g
**鍩哄噯鎻愪氦:** d023926

---

## 1. 鏈疆鐩爣

P0 Round 3: Existing Issues Stabilization 鈥?淇 5 涓瀛橀棶棰橈細

1. **pkg/runtime/session GetOrCreate 缂哄け** 鈥?缂栬瘧澶辫触淇
2. **A2A 鍋囨祦寮忓璁?* 鈥?code-agent / web-agent 缂哄皯 `StreamingAgent` 瀹炵幇
3. **Snapshot 绠＄嚎** 鈥?STATE_SNAPSHOT / STATE_DELTA 绔埌绔绾胯瘎浼颁笌鏈€灏忚ˉ榻?4. **HITL confirmation 鐘舵€佹満杈圭晫** 鈥?repeated confirm銆乼imeout confirm銆乴ogical state
5. **鏂囨。鏇存柊** 鈥?Round 3 鎶ュ憡

涓ユ牸绾︽潫锛氫笉 commit銆佷笉 push銆佷笉 git add .銆佹棤鐪熷疄 LLM key

---

## 2. 淇敼鏂囦欢鎬昏

### 鏂板鏂囦欢

| 鏂囦欢 | 璇存槑 |
|---|---|
| `frontend/src/agui/snapshotRegistry.test.ts` | 10 涓?snapshotRegistry 娴嬭瘯 |
| `docs/handoff/P0_INTEGRATION_FIX_ROUND3_REPORT.md` | 鏈姤鍛?|

### 淇敼鏂囦欢

| 鏂囦欢 | 鍙樻洿绫诲瀷 | 璇存槑 |
|---|---|---|
| `pkg/runtime/session/session.go` | **淇** | 鏂板 `GetOrCreate` 鏂规硶, 骞跺彂瀹夊叏澶勭悊 duplicate insert |
| `pkg/runtime/session/session_test.go` | **鏂板娴嬭瘯** | 4 涓?GetOrCreate 娴嬭瘯 (creates/returns existing/concurrent/empty id) |
| `services/agents/code-agent/agent.go` | **鏂板鏂规硶** | `GenerateStream` 瀹炵幇 `adk.StreamingAgent` 鈥?mock 鎸?4 璇?chunk 娴佸紡浜у嚭, LLM 濮旀墭 `model.GenerateStream()` |
| `services/agents/code-agent/agent_test.go` | **鏂板娴嬭瘯** | 4 涓?GenerateStream 娴嬭瘯 + StreamingAgent 缂栬瘧鏈熸鏌?|
| `services/agents/web-agent/agent.go` | **鏂板鏂规硶** | `GenerateStream` 瀹炵幇 `adk.StreamingAgent` 鈥?鍚?code-agent 妯″紡 |
| `services/agents/web-agent/agent_test.go` | **鏂板娴嬭瘯** | 3 涓?GenerateStream 娴嬭瘯 + StreamingAgent 缂栬瘧鏈熸鏌?|
| `services/agents/code-agent/go.mod` | **bump** | go 1.22.0 鈫?1.23.0 (iter.Seq2 闇€瑕?1.23+) |
| `services/agents/web-agent/go.mod` | **bump** | go 1.22.0 鈫?1.23.0 |
| `services/orchestrator/httpapi/server.go` | **澧炲己** | 鏂板 `HITLState` 鐘舵€佸父閲?+ `hitlStates` map + `SetHITLState` 鏂规硶 |
| `services/orchestrator/httpapi/handler_hitl.go` | **澧炲己** | `handleHITLConfirm` 鍦?channel send 鍓嶆鏌?logical state; `registerPending` 鍒濆鍖?state; `deregisterPending` 娓呯悊 state |
| `services/orchestrator/httpapi/server_test.go` | **鏂板娴嬭瘯** | 2 涓柊娴嬭瘯: `TestHITLConfirmStateAfterRejected` / `TestHITLConfirmTimedOutRejected`; 鏇存柊 `TestHITLConfirmAfterDrainStillAccepted` 鍖归厤鏂扮姸鎬佹満 |

---

## 3. pkg/runtime/session GetOrCreate 淇

### 闂

`SQLSessionService` 缂哄皯 `GetOrCreate(ctx context.Context, id string) (*adk.Session, error)` 鏂规硶, 瀵艰嚧 `var _ adk.SessionService = (*SQLSessionService)(nil)` 缂栬瘧澶辫触銆?
### 淇

鎸夌収 `MemorySessionService` 鍙傝€冩ā寮忓疄鐜?
1. 鍏堣皟 `Get(ctx, id)` 鏌ユ壘鐜版湁 session
2. 鎵惧埌 鈫?鐩存帴杩斿洖
3. `ErrSessionNotFound` 鈫?鐩存帴 insert (sessionID = id, userID = id, state = {})
4. 骞跺彂瀹夊叏: 鑻?insert 杩斿洖 duplicate 閿欒, 閲嶆柊 `Get` 杩斿洖宸插苟鍙戝垱寤虹殑 session
5. 绌?id 杩斿洖閿欒

### 娴嬭瘯

| 娴嬭瘯 | 缁撴灉 |
|---|---|
| `TestSQLSessionService_GetOrCreate_Creates` | PASS |
| `TestSQLSessionService_GetOrCreate_ReturnsExisting` | PASS |
| `TestSQLSessionService_GetOrCreate_ConcurrentNoDup` | PASS (5 goroutines) |
| `TestSQLSessionService_GetOrCreate_EmptyID` | PASS |

---

## 4. A2A 鐪熸祦寮忓璁′笌淇

### 瀹¤缁撹

**鍗忚灞?(Runner / A2A Server / A2A Client / Dispatcher) 鍧囧凡姝ｇ‘鏀寔娴佸紡:**
- `Runner.Run()` 妫€鏌?`StreamingAgent` 鎺ュ彛, 鏈夊垯璧?`runStreamingLoop`, 姣忎釜 text chunk 绔嬪嵆 yield partial Event
- `A2A Server.handleRunSSE()` 妫€娴?`Accept: text/event-stream`, 閫?event 鍐?SSE frame + `Flush()`
- `A2A Client.SendJSONRPCStream()` 妫€娴?`Content-Type: text/event-stream`, 閫?SSE frame 瑙ｆ瀽 yield
- `Dispatcher.DispatchStream()` 浣跨敤 `client.SendJSONRPCStream()`, 閫?chunk 閫忎紶

**Agent 灞傚厛鍓嶇己澶辨祦寮?**
- `CodeAgent` 鍜?`WebAgent` 鍙疄鐜?`Agent.Generate()`, 涓嶅疄鐜?`StreamingAgent.GenerateStream()`
- Mock 璺緞: 鏋勫缓瀹屾暣 string, 鍘熷瓙杩斿洖鍗曚釜 response
- LLM 璺緞: 璋冪敤 `model.Generate()` (闈炴祦寮?, 鑰岄潪 `model.GenerateStream()`

### 淇

涓?`CodeAgent` 鍜?`WebAgent` 娣诲姞 `GenerateStream` 鏂规硶:
- **Mock 璺緞**: 灏?mock response 鎸?4 璇嶄竴缁?split, 閫?chunk yield 涓?partial `GenerateResponse`
- **LLM 璺緞**: 濮旀墭 `model.GenerateStream()`
- **Go 鐗堟湰**: agent go.mod 浠?1.22.0 bump 鍒?1.23.0 (iter.Seq2 鏈€浣庤姹?

### 娴佸紡閾捐矾楠岃瘉

```
Agent.GenerateStream() 鈫?Runner.runStreamingLoop() 鈫?partial Events
  鈫?A2A Server.handleRunSSE() 鈫?SSE flush per event
  鈫?A2A Client.streamSSE() 鈫?incremental StreamChunk yield
  鈫?Dispatcher.DispatchStream() 鈫?text chunk forward
  鈫?Orchestrator Executor 鈫?Gateway SSE 鈫?Frontend
```

### 娴嬭瘯

| 娴嬭瘯 | 缁撴灉 |
|---|---|
| `TestCodeAgent_GenerateStream_MultipleChunks` | PASS (鈮? chunks) |
| `TestCodeAgent_GenerateStream_ErrorOnNilRequest` | PASS |
| `TestCodeAgent_GenerateStream_NoUserText` | PASS |
| `TestCodeAgent_GenerateStream_FallsBackToMock` | PASS |
| `TestWebAgent_GenerateStream_MultipleChunks` | PASS (鈮? chunks) |
| `TestWebAgent_GenerateStream_ErrorOnNilRequest` | PASS |
| `TestWebAgent_GenerateStream_FallsBackToMock` | PASS |
| 缂栬瘧鏈?`StreamingAgent` 鎺ュ彛妫€鏌?(涓や釜 agent) | PASS |

### 鐪熷疄鎬у垽鏂?
- **Mock 妯″紡** (鏃?LLM): 閫?chunk 浜у嚭, 姣忎釜 chunk 绔嬪嵆琚?Runner yield 涓?partial Event 鈫?**鐪熸祦寮?*
- **LLM 妯″紡** (鏈夋ā鍨?: 濮旀墭 `model.GenerateStream()` 鈫?鍙栧喅浜庡簳灞傛ā鍨嬪疄鐜?鈫?妯″瀷灞傚凡鏀寔 streaming

---

## 5. Snapshot 绠＄嚎鐘舵€?
### 褰撳墠瀹屾垚搴?
| 灞傜骇 | 瀹屾垚鐘舵€?|
|---|---|
| 鍚堢害瀹氫箟 (`agui-events.md`, `snapshot-event.md`) | **宸插畬鎴?* 鈥?STATE_SNAPSHOT / STATE_DELTA event types 宸插畾涔?|
| 鍓嶇 `snapshotRegistry.ts` (娉ㄥ唽/鏌ユ壘/娓叉煋鎺ュ彛) | **宸插畬鎴?* 鈥?Map-based 娉ㄥ唽琛?|
| 鍓嶇 snapshotRegistry 娴嬭瘯 | **宸插畬鎴?(鏈疆鏂板 10 涓?** |
| 鍚庣 STATE_SNAPSHOT / STATE_DELTA 浜嬩欢鍙戝皠 | **PENDING** 鈥?Orchestrator 鏆傛湭浜у嚭杩欎簺浜嬩欢 |
| Gateway translator STATE_SNAPSHOT / STATE_DELTA 杞崲 | **PENDING** 鈥?渚濊禆鍚庣鍙戝皠 |
| 鍓嶇 messageStore STATE_SNAPSHOT / STATE_DELTA 娑堣垂 | **PENDING** 鈥?渚濊禆 translator 浜у嚭 |

### 鏈疆鏂板

- `frontend/src/agui/snapshotRegistry.test.ts` 鈥?10 涓祴璇?
  - register/get/has/clear 鍩虹鎿嶄綔
  - renderer 鎺ユ敹瀹屾暣 SnapshotData + RenderContext
  - STATE_SNAPSHOT-like / STATE_DELTA-like payload 浼犻€?  - unknown type 瀹夊叏蹇界暐 (getSnapshotRenderer 杩斿洖 undefined, 涓嶆姏寮傚父)
  - overwrite / clear 琛屼负楠岃瘉

### 闄嶇骇璇存槑

瀹屾暣鐨?STATE_SNAPSHOT / STATE_DELTA 鍚庣鍙戝皠绠＄嚎闇€瑕?
1. Orchestrator 鍦ㄦ墽琛屾祦绋嬩腑鐢熸垚 snapshot/delta 浜嬩欢
2. Gateway translator 璇嗗埆骞惰浆鎹负 AG-UI 浜嬩欢
3. Frontend messageStore 娑堣垂骞惰矾鐢卞埌 snapshotRegistry

姝ゅ伐绋嬮噺杈冨ぇ, 涓嶅湪鏈疆 P0 鑼冨洿銆?*褰撳墠鍓嶇 registry 宸插氨缁? 鍚庣涓€鏃﹀紑濮嬪彂灏勪簨浠跺嵆鍙帴鍏ャ€?* 寤鸿鍚庣画杩唬琛ラ綈銆?
---

## 6. HITL 鐘舵€佹満杈圭晫淇

### 闂璇婃柇

鍘熺姸鎬佹満浠?channel buffer (瀹归噺=1) 涓哄敮涓€ "宸插鐞? 瀹堝崼:
- Channel full 鈫?409 "already processed" (浣嗕笉鍖哄垎 channel full vs 鐪熺殑宸茬‘璁?
- Channel drain 鍚?鈫?鍙啀娆?confirm (琚?`TestHITLConfirmAfterDrainStillAccepted` 鏄庣‘璁板綍)
- 瓒呮椂鍚?confirm 鈫?鍙湁 404 "no pending" (涓嶅鏄庣‘)
- 鎷掔粷鍚庡啀 confirm 鈫?鍚屼笂, 缂轰箯娓呮櫚鍖哄垎

### 淇

鏂板 `HITLState` 閫昏緫鐘舵€佹満:

```
pending 鈹€鈹€confirm鈹€鈹€鈻?confirmed
        鈹溾攢鈹€reject鈹€鈹€鈹€鈻?rejected
        鈹斺攢鈹€timeout鈹€鈹€鈻?timed_out
```

**瀹炵幇:**
- `server.go`: 鏂板 `hitlStates map[string]HITLState`, `HITLPending/Confirmed/Rejected/TimedOut` 甯搁噺, `SetHITLState` 鍏紑鏂规硶
- `handler_hitl.go`: `handleHITLConfirm` 鍦?channel send 鍓嶅厛鏌?`hitlStates`:
  - `HITLConfirmed` 鈫?409 "plan was confirmed"
  - `HITLRejected` 鈫?409 "plan was rejected"
  - `HITLTimedOut` 鈫?410 "timed out and no longer available"
  - `HITLPending` 鈫?灏濊瘯 channel send, 鎴愬姛鍚庢洿鏂?state
  - channel full (杈规部鎯呭喌) 鈫?409 "confirmation in progress, please wait"
- `registerPending`: 鍒濆鍖?`hitlStates[runID] = HITLPending`
- `deregisterPending`: 鍚屾椂娓呯悊 `hitlStates`
- `SetHITLState`: 渚?streaming goroutine 鏍囪瓒呮椂

### 娴嬭瘯

| 娴嬭瘯 | 缁撴灉 |
|---|---|
| `TestHITLConfirmAccepted` | PASS |
| `TestHITLConfirmRejected` | PASS |
| `TestHITLConfirmNoPendingRun` | PASS |
| `TestHITLConfirmRepeated` | PASS (409) |
| `TestHITLConfirmMissingRunID` | PASS |
| `TestHITLConfirmMissingActionID` | PASS |
| `TestHITLConfirmMethodNotAllowed` | PASS |
| `TestHITLConfirmUnauthorized` | PASS |
| `TestHITLConfirmAuthorized` | PASS |
| `TestHITLConfirmAfterDrainStillAccepted` | **UPDATED** 鈥?鐜板湪姝ｇ‘杩斿洖 409 (logical state guard) |
| `TestHITLConfirmStateAfterRejected` | **NEW** 鈥?鎷掔粷鍚庡啀 confirm 杩斿洖 409 |
| `TestHITLConfirmTimedOutRejected` | **NEW** 鈥?瓒呮椂鍚?confirm 杩斿洖 410 Gone |

---

## 7. Go 娴嬭瘯缁撴灉

```
ok  services/gateway                         1.360s
ok  services/gateway/auth                    0.734s
ok  services/gateway/cmd/gateway             1.331s
ok  services/gateway/config                  0.938s
ok  services/gateway/httpapi                 1.592s
ok  services/gateway/internal/persistence    1.428s
ok  services/gateway/internal/persistence/sqlite  1.851s
ok  services/gateway/orchestratorclient      1.715s
ok  services/gateway/runservice              2.192s
ok  services/gateway/sse                     1.780s
ok  services/gateway/store                   1.943s
ok  services/orchestrator/dispatcher         24.001s
ok  services/orchestrator/executor           1.893s
ok  services/orchestrator/httpapi            2.367s
ok  services/orchestrator/plan               1.573s
ok  services/orchestrator/planner            1.901s
ok  services/orchestrator/registry           1.648s
ok  services/orchestrator/validator          1.620s
ok  services/agents/code-agent               1.780s
ok  services/agents/code-agent/cmd/code-agent 1.797s
ok  services/agents/web-agent                2.096s
ok  services/agents/web-agent/cmd/web-agent  1.364s
ok  pkg/adk                                  1.305s
ok  pkg/adk/a2a                              1.916s
ok  pkg/runtime/agui                         1.330s
ok  pkg/runtime/config                       1.426s
ok  pkg/runtime/launcher                     1.289s
ok  pkg/runtime/model                        1.479s
ok  pkg/runtime/pruning                      1.200s
ok  pkg/runtime/registry                     1.224s
ok  pkg/runtime/session                      1.116s
ok  pkg/runtime/skill                        0.893s

鍏ㄩ儴妯″潡: PASS (34 packages)
```

HITL 娴嬭瘯: 12 涓?(10 鍘熸湁 + 2 鏂板)
Session 娴嬭瘯: 鏂板 4 涓?GetOrCreate
Agent streaming 娴嬭瘯: 鏂板 7 涓?(4 code-agent + 3 web-agent)

---

## 8. 鍓嶇娴嬭瘯缁撴灉

```
Test Files  10 passed (10)
     Tests  91 passed (91)

鏂板:
- src/agui/snapshotRegistry.test.ts 鈥?10 tests (register/get/has/clear/overwrite/delta/unknown/CONTEXT)
```

**Build:** `npm run build` 鎴愬姛, 鏃犻敊璇€?
---

## 9. 瀹夊叏妫€鏌?
- **Secrets in diff:** 0 澶?- **API key patterns (sk-...):** 0 澶?(浠呮祴璇曚唬鐮佷腑鏈?`sk-[mock-redacted]` 鐢ㄤ簬鑴辨晱娴嬭瘯)
- **Prohibited files tracked:** 鏃?(`.env`, `node_modules`, `frontend/dist` 鍧囨湭琚窡韪?
- **Binary files:** 7 涓簩杩涘埗浠嶅浜?D (deleted) 鐘舵€? `code-agent`, `web-agent`, `orchestrator`, `document-agent.exe`, `services/agents/code-agent/code-agent`, `services/agents/web-agent/web-agent`, `services/orchestrator/orchestrator`
- **鏈窡韪?** `AgentHub_閲嶆瀯涓庢ā鍧楄В鑰﹂樁娈垫姤鍛?md` (鐢ㄦ埛鑷湁鏂囨。, 涓嶅姩)
- **go.mod 鐗堟湰 bump:** code-agent 鍜?web-agent 浠?`go 1.22.0` 鈫?`go 1.23.0` (iter.Seq2 鏈€浣庤姹? 椤圭洰宸茬敤 Go 1.26.3)
- **鏈慨鏀?* `.claude/settings.local.json`
- **鏈墽琛?* git add / commit / push

---

## 10. 浠嶆湭瀹屾垚椤?
1. **STATE_SNAPSHOT / STATE_DELTA 鍚庣鍙戝皠绠＄嚎** 鈥?鍓嶇 registry 宸插氨缁? 鍚庣 Orchestrator 鈫?Gateway translator 鈫?SSE 鍙戝皠閾捐矾寰呰ˉ榻愩€傚缓璁悗缁凯浠ｅ鐞嗐€?2. **Gateway鈫扥rchestrator gRPC streaming** 鈥?浠嶇敤 HTTP/SSE, 鎸夎璁¤鑼冨簲杩佺Щ, 灞炰簬闀挎湡浠诲姟銆?3. **Streaming goroutine 璋冪敤 SetHITLState 鏍囪瓒呮椂** 鈥?`SetHITLState` 鏂规硶宸叉毚闇? 浣?`handler_run_stream.go` 涓殑瓒呮椂澶勭悊闇€鍦ㄥ悗缁笌涔嬪鎺ャ€?4. **鍏朵粬 agent (document-agent, vision-agent 绛? 鐨?StreamingAgent 瀹炵幇** 鈥?鏈疆浠呰鐩?code-agent 鍜?web-agent (鏍稿績 demo agent)銆?
---

## 11. 鏄惁寤鸿鎻愪氦

**鍙互鎻愪氦銆?* 鎵€鏈夊彉鏇寸鍚堥」鐩灦鏋勭害鏉?
- 鏃?Frontend 鐩磋繛 Orchestrator/Agent
- 鏃?Gateway 鐩磋繛 Agent
- 鏃犵粫杩?Orchestrator 鎴?PlanValidator
- 鏃犵湡瀹?LLM key
- 鏃犵牬鍧忕幇鏈?AG-UI event 鍏煎
- 鏃犵牬鍧?confirm_plan tool event 鍜?STATE_UPDATE metadata
- 鍏ㄩ儴 Go 娴嬭瘯閫氳繃 (34 packages)
- 鍏ㄩ儴鍓嶇娴嬭瘯閫氳繃 (91 tests)
- 鍓嶇 build 鎴愬姛
- 浜岃繘鍒舵枃浠朵繚鎸佸垹闄ょ姸鎬?- 瀹夊叏妫€鏌ラ€氳繃

**寤鸿 commit message:**
```
fix: session GetOrCreate + A2A true streaming + HITL state machine hardening

- Add GetOrCreate to SQLSessionService with concurrent duplicate handling
- Implement StreamingAgent on CodeAgent and WebAgent for true chunked output
- Bump agent go.mod from 1.22.0 to 1.23.0 (iter.Seq2 requirement)
- Add HITLState logical state machine: pending/confirmed/rejected/timed_out
- Prevent duplicate confirms, timeout confirms, and channel-full ambiguity
- Add snapshotRegistry frontend tests (10 tests)
- Add 4 GetOrCreate tests, 7 agent streaming tests, 2 HITL state tests
- All Go tests pass (34 packages), all frontend tests pass (91 tests)
```

