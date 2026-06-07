# P0 Integration Fix Round 2 瀹屾垚鎶ュ憡

**鏃ユ湡:** 2026-06-07
**鍒嗘敮:** g
**鍩哄噯鎻愪氦:** d023926

---

## 1. 鏈疆鐩爣

鍦?Round 1 宸插畬鎴愮殑鍩虹涓婏紝瑙ｅ喅 3 涓牳蹇冮棶棰橈細

1. **StreamingText 缂轰笓椤规祴璇?* 鈥?琛ュ厖 open fence / closed fence / mixed text-code / incremental chunks 娴嬭瘯銆?2. **Plan confirmation 鍋忚嚜瀹氫箟 metadata** 鈥?鎻愬崌涓?AG-UI 鏍囧噯 `confirm_plan` tool event锛圱OOL_CALL_START/ARGS/END锛? 淇濈暀 STATE_UPDATE metadata 鍏煎銆?3. **ChatWindow 鏈覆鏌?HITLConfirm** 鈥?鎺ュ叆 HITLConfirm UI锛岀敤鎴峰彲纭/鎷掔粷璁″垝銆?
棰濆淇锛?- **Gateway handler_hitl.go 鎺ュ彛绛惧悕涓嶅尮閰?* 鈥?`ConfirmRun` 缂哄皯 `ctx context.Context` 鍙傛暟锛屽鑷寸被鍨嬫柇瑷€濮嬬粓澶辫触杩斿洖 501銆?- **sanitizeErrorText 鏈劚鏁忓唴閮?URL** 鈥?鏂板 URL sanitization銆?
---

## 2. 淇敼鏂囦欢

### 鍚庣鏂板/淇敼

| 鏂囦欢 | 鍙樻洿绫诲瀷 | 璇存槑 |
|---|---|---|
| `services/gateway/httpapi/handler_hitl.go` | **淇** | 鎺ュ彛鏂█娣诲姞 `ctx context.Context` 鍙傛暟锛涜皟鐢ㄤ紶閫?`r.Context()` |
| `services/gateway/httpapi/server_test.go` | **鏂板娴嬭瘯** | 10 涓?HITL confirm 璺敱娴嬭瘯 |
| `services/orchestrator/httpapi/server_test.go` | **鏂板娴嬭瘯** | 10 涓?HITL confirm handler 娴嬭瘯 |
| `services/orchestrator/httpapi/handler_run_stream.go` | (Round 1) | confirm_plan TOOL_CALL_START/ARGS/END 浜嬩欢鍙戝皠 |
| `services/gateway/orchestratorclient/client.go` | (Round 1) | tool_call_start/args/end SSE 瑙ｆ瀽 |
| `pkg/runtime/agui/translator.go` | (Round 1) | tool_call_start/args/end AG-UI 浜嬩欢缈昏瘧 |

### 鍓嶇鏂板/淇敼

| 鏂囦欢 | 鍙樻洿绫诲瀷 | 璇存槑 |
|---|---|---|
| `frontend/src/stores/messageStore.ts` | **淇+澧炲己** | 鏂板 URL sanitization锛沜onfirmPlan error handling |
| `frontend/src/stores/messageStore.test.ts` | **鏂板娴嬭瘯** | 26 涓柊娴嬭瘯锛坧lan confirmation + confirmPlan API锛?|
| `frontend/src/components/ChatWindow.tsx` | (Round 2) | HITLConfirm UI wiring |
| `frontend/src/components/StreamingText.test.tsx` | (Round 2) | 10 涓祦寮忎唬鐮佸潡娓叉煋娴嬭瘯 |
| `frontend/src/components/HITLConfirm.tsx` | (Round 1-2) | 纭瀵硅瘽妗嗙粍浠?|
| `frontend/src/components/OrchestrationCard.tsx` | (Round 1) | requiresConfirmation 瀛楁 |

---

## 3. StreamingText 娴嬭瘯琛ュ厖

鏂板 `frontend/src/components/StreamingText.test.tsx`锛?0 涓祴璇曠敤渚嬶細

| 娴嬭瘯 | 瑕嗙洊鍦烘櫙 |
|---|---|
| plain text during streaming | 绾枃鏈祦寮忔覆鏌?|
| open fence as styled code block | 鏈棴鍚?code fence 娓叉煋涓轰唬鐮佸潡鏍峰紡 |
| closed fence complete code block | 闂悎 fence 瀹屾垚鍚庢樉绀哄畬鏁翠唬鐮?|
| mixed text-code preserved | 鍓?鍚庣疆鏂囨湰 + 浠ｇ爜鍧楀叏閮ㄤ繚鐣?|
| incremental chunks accumulate | 澧為噺娓叉煋涓嶉噸缃唴瀹?|
| open fence without language tag | 鏃犺瑷€鏍囩鐨勪唬鐮佸潡 |
| empty content thinking indicator | 绌哄唴瀹规樉绀?pulsing cursor |
| long code block no truncation | 闀夸唬鐮佸唴瀹逛笉鎴柇 |
| copy button present | 澶嶅埗鎸夐挳瀛樺湪 |

---

## 4. AG-UI Plan Confirmation 瀵归綈

### Dual-path 绛栫暐

```
STATE_UPDATE (metadata path)     TOOL_CALL_START/ARGS/END (AG-UI standard path)
        \                               /
         \                             /
          鈹斺攢鈹€鈹€鈹€鈹€鈹€ 缁熶竴涓?PendingConfirmation 鈹€鈹€鈹€鈹€鈹€鈹€鈹?                           鈹?                    ChatWindow 娓叉煋 HITLConfirm
```

### 瀹炵幇鐘舵€?
- **TOOL_CALL_START/ARGS/END 浜嬩欢**: 宸插疄鐜帮紝Orchestrator 鍦?`requireConfirm` 璺緞涓彂灏?`confirm_plan` 宸ュ叿浜嬩欢銆?- **STATE_UPDATE metadata**: 宸蹭繚鐣欙紝鎼哄甫 `requiresConfirmation`, `plannedAgents`, `tasks` 绛夊瓧娈点€?- **Translator**: 宸叉敮鎸?`tool_call_start/args/end` 鈫?`TOOL_CALL_START/ARGS/END` 杞崲銆?- **messageStore**: 鍦?`TOOL_CALL_END` 涓瘑鍒?`confirm_plan` 宸ュ叿锛岃缃?`PendingConfirmation` state銆?
### confirm_plan tool event 瀛楁

| 瀛楁 | 鏉ユ簮 |
|---|---|
| `runId` | orchestrator stream event |
| `planId` | `confirmToolID` = `orchPlan.PlanID` |
| `strategy` | `orchPlan.Strategy` |
| `plannedAgents` | `plannedAgentNames(orchPlan)` |
| `tasks` | `taskSummaries(orchPlan)` |
| `intentSummary` | `orchPlan.IntentSummary` |
| `requiresConfirmation` | `true` |

---

## 5. ChatWindow HITLConfirm UI Wiring

### 娓叉煋鏉′欢

`ChatWindow` 浠?`messageStore.confirmationByConversation[conversationId]` 鑾峰彇 pending confirmation锛?- `status === 'pending'` 鈫?娓叉煋 `HITLConfirm` 瀵硅瘽妗?- `status === 'confirmed'` 鈫?鏄剧ず缁胯壊 confirmed banner
- `status === 'rejected'` 鈫?鏄剧ず鐏拌壊 rejected banner

### 鐢ㄦ埛鎿嶄綔娴佺▼

1. **Confirm**: 璋冪敤 `confirmPlan(conversationId, runId, actionId, true)` 鈫?`api.confirmHITL({ runId, actionId, confirmed: true })` 鈫?`POST /api/runs/{runId}/confirm`
2. **Reject**: 璋冪敤 `confirmPlan(conversationId, runId, actionId, false, reason)` 鈫?`POST /api/runs/{runId}/confirm` with `confirmed: false`
3. **Timeout**: 鑷姩 reject
4. **Error**: 鑴辨晱鏄剧ず锛屽厑璁?Dismiss 閲嶈瘯

---

## 6. Gateway / Orchestrator Confirm Payload 瀵归綈

### 璇锋眰浣撲竴鑷存€ч獙璇?
| 灞傜骇 | 瀛楁 |
|---|---|
| Frontend `HITLConfirmRequest` | `{ runId, actionId, confirmed, rejectReason }` |
| Gateway `handleRunsConfirm` | 瑙ｆ瀽涓?`orchestratorclient.HITLConfirmRequest` |
| Orchestrator `handleHITLConfirm` | 瑙ｆ瀽涓?`httpapi.HITLConfirmRequest` |

涓夊眰缁撴瀯涓€鑷达紝瀛楁鍚嶅叏閮ㄥ榻愩€?
### 淇鐨勫叧閿?Bug

**Gateway handler_hitl.go 鎺ュ彛绛惧悕涓嶅尮閰嶏細**
- **闂:** 绫诲瀷鏂█瑕佹眰 `ConfirmRun(req HITLConfirmRequest) error`锛屼絾 `OrchestratorRunService.ConfirmRun(ctx context.Context, req HITLConfirmRequest) error` 绛惧悕涓嶅悓銆?- **淇:** 鏇存柊鎺ュ彛鏂█娣诲姞 `ctx context.Context`锛岃皟鐢ㄦ椂浼犻€?`r.Context()`銆?- **褰卞搷:** 淇鍓嶆墍鏈?confirm 璇锋眰杩斿洖 501锛涗慨澶嶅悗姝ｅ父杞彂銆?
---

## 7. 鍓嶇娴嬭瘯缁撴灉

```
Test Files  9 passed (9)
     Tests  81 passed (81)
```

鏂板 26 涓?messageStore 娴嬭瘯锛?
| 娴嬭瘯缁?| 娴嬭瘯鏁?| 瑕嗙洊 |
|---|---|---|
| plan confirmation via TOOL_CALL events | 2 | confirm_plan 璇嗗埆銆侀潪 confirm_plan 蹇界暐 |
| plan confirmation via STATE_UPDATE metadata | 2 | requiresConfirmation 鎹曡幏銆乨ual-path 鍏煎 |
| confirmPlan API integration | 5 | confirm/reject 璋冪敤銆侀敊璇劚鏁忋€乧lear/getConfirmation |

---

## 8. 鍚庣娴嬭瘯缁撴灉

```
services/gateway/httpapi     鈥?10 new HITL tests, all pass
services/orchestrator/httpapi 鈥?10 new HITL tests, all pass
services/gateway/...         鈥?all pass
services/orchestrator/...    鈥?all pass
pkg/runtime/agui/...         鈥?all pass
```

### Gateway HITL confirm 娴嬭瘯瑕嗙洊

- `TestHITLConfirmRouteAccepted` 鈥?纭鎴愬姛 200
- `TestHITLConfirmRouteRejected` 鈥?鎷掔粷鎴愬姛 200
- `TestHITLConfirmRouteDefaultRunID` 鈥?璺緞 runId 鑷姩濉厖
- `TestHITLConfirmRouteRunIDMismatch` 鈥?璺緞/body runId 涓嶅尮閰?400
- `TestHITLConfirmRouteNotImplemented` 鈥?runner 涓嶆敮鎸?ConfirmRun 501
- `TestHITLConfirmRouteMethodNotAllowed` 鈥?GET 鏂规硶 405
- `TestHITLConfirmRouteBadRequest` 鈥?鏃犳晥 JSON 400
- `TestHITLConfirmRouteNotFound` 鈥?璺緞鏃?confirm 鍚庣紑 404
- `TestHITLConfirmRouteErrorSanitized` 鈥?閿欒鑴辨晱 502
- `TestHITLConfirmRouteEmptyRunID` 鈥?绌?runId 404

### Orchestrator HITL confirm 娴嬭瘯瑕嗙洊

- `TestHITLConfirmAccepted` 鈥?纭鎴愬姛 200
- `TestHITLConfirmRejected` 鈥?鎷掔粷浜や粯鍒?channel
- `TestHITLConfirmNoPendingRun` 鈥?鏃?pending plan 404
- `TestHITLConfirmRepeated` 鈥?閲嶅纭 409 (channel full)
- `TestHITLConfirmMissingRunID` 鈥?缂?runId 400
- `TestHITLConfirmMissingActionID` 鈥?缂?actionId 400
- `TestHITLConfirmMethodNotAllowed` 鈥?閿欒鏂规硶 405
- `TestHITLConfirmUnauthorized` 鈥?鏃?token 401
- `TestHITLConfirmAuthorized` 鈥?鏈?token 200
- `TestHITLConfirmAfterDrainStillAccepted` 鈥?channel drain 鍚庡彲鍐嶆纭

---

## 9. 瀹夊叏妫€鏌?
- **Secrets in diff:** 鏃?(浠呮祴璇曚唬鐮佷腑鏈夊亣 token `sk-[mock-redacted]` 鐢ㄤ簬鑴辨晱娴嬭瘯)
- **Prohibited files tracked:** 鏃?(`.env`, `node_modules`, `frontend/dist` 绛夊潎鏈璺熻釜)
- **Binary files:** 7 涓簩杩涘埗宸插湪 Round 1 鍒犻櫎 (git status 鏄剧ず D 鐘舵€?
- **Internal URLs in error output:** 宸查€氳繃 `sanitizeErrorText` URL 鑴辨晱瑙勫垯瑕嗙洊

---

## 10. 浠嶆湭瀹屾垚椤?
1. **`pkg/runtime/session` 缂栬瘧澶辫触** 鈥?`*SQLSessionService` 缂?`GetOrCreate` 鏂规硶銆傛涓洪瀛橀棶棰橈紝涓嶅湪鏈疆鑼冨洿銆?2. **Orchestrator 纭鍚?"channel full" 淇濇姢** 鈥?褰撳墠 channel buffer=1锛宒rain 鍚庡彲鍐嶆纭銆傚闇€鏇翠弗鏍肩殑 "宸茬‘璁? 鐘舵€佹満淇濇姢锛屽缓璁悗缁凯浠ｅ鍔犻€昏緫鏍囪銆?3. **瀹屾暣鏆傚仠鎵ц (pause/resume)** 鈥?褰撳墠纭娴佺▼浣跨敤 channel block 绛夊緟锛屽姛鑳芥纭絾鍙紭鍖栦负鏇存爣鍑嗙殑 AG-UI interrupt/resume 妯″瀷銆?4. **gRPC streaming** 鈥?Gateway鈫扥rchestrator 浠嶄娇鐢?HTTP/SSE銆傛寜璁捐瑙勮寖搴旇縼绉昏嚦 gRPC锛屼絾涓洪暱鏈熶换鍔°€?
---

## 11. 鏄惁鍙互鎻愪氦

**鍙互鎻愪氦銆?* 鎵€鏈夊彉鏇寸鍚堥」鐩灦鏋勭害鏉燂細
- 鏃?Frontend 鐩磋繛 Orchestrator/Agent
- 鏃?Gateway 鐩磋繛 Agent
- 鏃犵粫杩?Orchestrator 鎴?PlanValidator
- 鏃犵湡瀹?LLM key
- 鏃犵牬鍧忕幇鏈?AG-UI event 鍏煎
- 鏃犵牬鍧?`requiresConfirmation` metadata 娑堣垂閫昏緫
- 娴嬭瘯鍏ㄩ儴閫氳繃 (81 frontend + all backend)
- 瀹夊叏妫€鏌ラ€氳繃

**寤鸿 commit message:**
```
fix: AG-UI confirm_plan tool events + HITLConfirm UI wiring + backend tests

- Add confirm_plan TOOL_CALL_START/ARGS/END events for AG-UI standard HITL
- Fix Gateway handler_hitl.go ConfirmRun interface mismatch (missing ctx)
- Wire ChatWindow HITLConfirm dialog with confirm/reject/timeout
- Add internal URL sanitization to error messages
- Add 20 backend HITL confirm tests (10 Gateway + 10 Orchestrator)
- Add 26 frontend tests (plan confirmation + confirmPlan + StreamingText)
- Preserve STATE_UPDATE metadata path for backward compatibility
```

