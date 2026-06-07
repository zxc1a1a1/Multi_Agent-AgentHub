package planner

import (
	"context"
	"strings"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// RulePlanner implements Planner using keyword-based rules.
// Mixed multi-agent keywords produce ordered_parallel; conversational intents
// produce a conversational plan with orchestrator self-response.
//
// Deprecated: RulePlanner is a transitional fallback only.
// Do not add new routing rules. It will be removed after LLMPlanner
// validation and repair are stable.
type RulePlanner struct {
	availableAgents []string
}

// NewRulePlanner creates a RulePlanner backed by the given agent names.
func NewRulePlanner(availableAgents []string) *RulePlanner {
	return &RulePlanner{
		availableAgents: availableAgents,
	}
}

// Plan generates an OrchestrationPlan using keyword rules.
// Conversational intents produce a conversational plan (no dispatch).
// Multi-agent keyword matches produce ordered_parallel plans;
// single-agent matches produce a single-task plan.
func (p *RulePlanner) Plan(ctx context.Context, input PlannerInput) (*plan.OrchestrationPlan, error) {
	msg := strings.ToLower(input.UserMessage)

	// 1. Conversational intent detection — return orchestrator self-response.
	// Skip conversational detection when an explicit (non-auto) agentName is set,
	// because the user explicitly chose a concrete agent.
	hasExplicitAgent := strings.TrimSpace(input.AgentName) != "" &&
		strings.TrimSpace(input.AgentName) != "auto"
	if !hasExplicitAgent && p.isConversational(msg) {
		return &plan.OrchestrationPlan{
			Version:        "v1",
			PlanID:         plan.NewPlanID(),
			RunID:          input.RunID,
			ConversationID: input.ConversationID,
			PlanningMode:   input.PlanningMode,
			Strategy:       plan.StrategyConversational,
			IntentSummary:  summarize(input.UserMessage),
			TraceID:        input.TraceID,
			Aggregation:    plan.Aggregation{Required: false, Mode: "none"},
			Fallback:       plan.Fallback{Enabled: false},
			Validation:     plan.Validation{Validated: false},
		}, nil
	}

	// 2. Multi-agent keyword matching.
	matched := p.matchedAgents(msg)

	// 2a. Code-analysis dependency pattern: code-agent + test/review/security.
	// These need a sequential DAG plan so code runs first and analysis agents
	// receive the generated code as upstream input via {{deps.task_code-agent.output}}.
	if len(matched) >= 2 && containsStr(matched, "code-agent") && hasAnalysisAgent(matched) {
		return p.buildCodeAnalysisPlan(input, matched), nil
	}

	if len(matched) >= 2 {
		tasks := make([]plan.TaskPlan, 0, len(matched))
		for i, agentName := range matched {
			tasks = append(tasks, plan.TaskPlan{
				TaskID:          "task_" + agentName,
				AgentName:       agentName,
				CapabilityIDs:   staticCapabilityIDs(agentName),
				TaskContent:     input.UserMessage,
				ExpectedOutputs: staticExpectedOutputs(agentName),
				DependsOn:       []string{},
				Priority:        i + 1,
				TimeoutMs:       120000,
				RiskLevel:       "low",
			})
		}
		return &plan.OrchestrationPlan{
			Version:        "v1",
			PlanID:         plan.NewPlanID(),
			RunID:          input.RunID,
			ConversationID: input.ConversationID,
			PlanningMode:   input.PlanningMode,
			Strategy:       plan.StrategyOrderedParallel,
			IntentSummary:  summarize(input.UserMessage),
			TraceID:        input.TraceID,
			Tasks:          tasks,
			Aggregation:    plan.Aggregation{Required: true, Mode: "summary"},
			Fallback:       plan.Fallback{Enabled: true, Reason: "rule_multi_agent"},
			Validation:     plan.Validation{Validated: false},
		}, nil
	}

	// 3. Single agent dispatch.
	agentName := p.determineAgent(input)

	return &plan.OrchestrationPlan{
		Version:        "v1",
		PlanID:         plan.NewPlanID(),
		RunID:          input.RunID,
		ConversationID: input.ConversationID,
		PlanningMode:   input.PlanningMode,
		Strategy:       plan.StrategySingle,
		IntentSummary:  summarize(input.UserMessage),
		TraceID:        input.TraceID,
		Tasks: []plan.TaskPlan{
			{
				TaskID:          "task_001",
				AgentName:       agentName,
				CapabilityIDs:   staticCapabilityIDs(agentName),
				TaskContent:     input.UserMessage,
				ExpectedOutputs: staticExpectedOutputs(agentName),
				DependsOn:       []string{},
				Priority:        1,
				TimeoutMs:       120000,
				RiskLevel:       "low",
			},
		},
		Aggregation: plan.Aggregation{Required: false, Mode: "none"},
		Fallback:    plan.Fallback{Enabled: true, Reason: "rule_default"},
		Validation:  plan.Validation{Validated: false},
	}, nil
}

// conversationalKeywords are multi-word phrases or Chinese phrases matched with
// substring containment. Single short English words use word-boundary matching
// to avoid false positives (e.g., "hi" inside "something").
var conversationalPhrases = []string{
	"你好", "你是谁", "你能做什么", "你能干嘛", "帮我介绍", "介绍一下",
	"hello", "who are you", "what can you", "what do you",
	"帮助", "使用说明", "功能", "自我介绍",
	"what is agenthub", "what agents", "可用agent", "有什么能力",
	"how are you", "good morning", "good afternoon",
}

// conversationalWords are single English words that indicate conversational intent.
// These are matched against whole words to avoid substring false positives.
var conversationalWords = []string{
	"hi", "hey", "help", "thanks", "thank",
}

// isConversational returns true when the message is purely conversational
// and should be answered by the orchestrator itself without dispatching agents.
func (p *RulePlanner) isConversational(msg string) bool {
	for _, kw := range conversationalPhrases {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	// Word-boundary matching for short English words.
	words := strings.Fields(msg)
	for _, word := range words {
		clean := strings.TrimFunc(word, func(r rune) bool {
			return r == '.' || r == ',' || r == '!' || r == '?' || r == ';' || r == ':'
		})
		clean = strings.ToLower(strings.TrimSpace(clean))
		for _, cw := range conversationalWords {
			if clean == cw {
				return true
			}
		}
	}
	return false
}

// matchedAgents returns the list of known agents whose keywords appear in msg.
func (p *RulePlanner) matchedAgents(msg string) []string {
	agentKeywords := map[string][]string{
		"web-agent":      webKeywords,
		"code-agent":     codeKeywords,
		"document-agent": {"文档", "doc", "readme", "api文档", "手册", "manual", "接口说明"},
		"vision-agent":   {"图片", "图像", "ocr", "识别", "截图", "照片"},
		"context-agent":  {"上下文", "摘要", "压缩", "总结", "记忆"},
		"test-agent":     {"测试", "test", "覆盖率", "coverage", "用例", "bug", "运行"},
		"review-agent":   {"审查", "review", "评审", "风险", "代码检查", "审计"},
		"security-agent": {"安全", "漏洞", "security", "vulnerability", "注入", "密钥"},
		"deploy-agent":   {"部署", "deploy", "发布", "上线", "回滚", "k8s", "docker"},
		"diff-agent":     {"diff", "差异", "对比", "patch", "合并冲突"},
	}

	var matched []string
	for agentName, keywords := range agentKeywords {
		if p.isKnown(agentName) && containsAny(msg, keywords) {
			matched = append(matched, agentName)
		}
	}
	// Sort by stable priority so map iteration order is deterministic.
	sortMatchedAgents(matched)
	return matched
}

var webKeywords = []string{
	"页面", "ui", "html", "react", "登录页", "前端", "前端页面",
	"page", "webpage", "css", "component", "layout", "登录页面",
}

var codeKeywords = []string{
	"go", "api", "后端", "接口", "server", "service", "后端登录",
	"golang", "handler", "endpoint", "database", "sql", "函数", "登录api",
	"代码", "编程", "程序", "写代码",
}

// determineAgent picks a single target agent based on priority:
//  1. Explicit agentName from request (mention/direct)
//  2. First element of selectedAgentNames
//  3. Keyword matching on userMessage
//  4. Default to "code-agent"
func (p *RulePlanner) determineAgent(input PlannerInput) string {
	if name := strings.TrimSpace(input.AgentName); name != "" {
		if p.isKnown(name) {
			return name
		}
	}
	if len(input.SelectedAgentNames) > 0 {
		if name := strings.TrimSpace(input.SelectedAgentNames[0]); name != "" && p.isKnown(name) {
			return name
		}
	}

	msg := strings.ToLower(input.UserMessage)

	// Check all specialized agent keywords in priority order.
	agentKeywords := map[string][]string{
		"web-agent":      webKeywords,
		"code-agent":     codeKeywords,
		"document-agent": {"文档", "doc", "readme", "api文档", "手册", "manual", "接口说明"},
		"vision-agent":   {"图片", "图像", "ocr", "识别", "截图", "照片"},
		"context-agent":  {"上下文", "摘要", "压缩", "总结", "记忆"},
		"test-agent":     {"测试", "test", "覆盖率", "coverage", "用例", "bug", "运行"},
		"review-agent":   {"审查", "review", "评审", "风险", "代码检查", "审计"},
		"security-agent": {"安全", "漏洞", "security", "vulnerability", "注入", "密钥"},
		"deploy-agent":   {"部署", "deploy", "发布", "上线", "回滚", "k8s", "docker"},
		"diff-agent":     {"diff", "差异", "对比", "patch", "合并冲突"},
	}

	for agentName, keywords := range agentKeywords {
		if p.isKnown(agentName) && containsAny(msg, keywords) {
			return agentName
		}
	}

	// Default fallback.
	if p.isKnown("code-agent") {
		return "code-agent"
	}
	if len(p.availableAgents) > 0 {
		return p.availableAgents[0]
	}
	return "code-agent"
}

func (p *RulePlanner) isKnown(name string) bool {
	for _, a := range p.availableAgents {
		if strings.EqualFold(a, name) {
			return true
		}
	}
	return false
}

func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func summarize(text string) string {
	const maxLen = 120
	cleaned := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if len(cleaned) <= maxLen {
		return cleaned
	}
	return cleaned[:maxLen-3] + "..."
}

func staticCapabilityIDs(agentName string) []string {
	switch strings.ToLower(strings.TrimSpace(agentName)) {
	case "code-agent":
		return []string{"code_generation"}
	case "web-agent":
		return []string{"web_generation"}
	default:
		return nil
	}
}

func staticExpectedOutputs(agentName string) []string {
	switch strings.ToLower(strings.TrimSpace(agentName)) {
	case "code-agent":
		return []string{"code"}
	case "web-agent":
		return []string{"webpage", "markdown"}
	default:
		return nil
	}
}

// codeAnalysisAgents lists agents that analyse code-agent output.
// When matched together with code-agent, the plan becomes sequential with
// code-agent in wave 0 and analysis agents in wave 1 depending on its output.
var codeAnalysisAgents = map[string]bool{
	"test-agent":     true,
	"review-agent":   true,
	"security-agent": true,
}

// hasAnalysisAgent reports whether matched contains at least one analysis agent
// that should depend on code-agent output.
func hasAnalysisAgent(matched []string) bool {
	for _, name := range matched {
		if codeAnalysisAgents[name] {
			return true
		}
	}
	return false
}

// containsStr reports whether slice contains target.
func containsStr(slice []string, target string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}

// codeAnalysisTaskPrompts maps analysis agents to a task prefix that explains
// what the agent should do with the upstream code.
var codeAnalysisTaskPrompts = map[string]string{
	"test-agent":     "请基于以下代码编写全面的测试用例（包括单元测试、边界输入、预期结果）：\n\n",
	"review-agent":   "请审查以下代码的质量、正确性和最佳实践（包括审查摘要、发现的问题、严重程度和改进建议）：\n\n",
	"security-agent": "请分析以下代码的安全漏洞（包括攻击面、输入验证风险、资源风险和修复建议）：\n\n",
}

// analysisAgentPriority maps analysis agents to their execution priority within wave 1.
var analysisAgentPriority = map[string]int{
	"test-agent":     2,
	"review-agent":   3,
	"security-agent": 4,
}

// codeTaskID is the stable task ID for the code-agent in dependency plans.
const codeTaskID = "task_code-agent"

// buildCodeAnalysisPlan builds a StrategySequential plan where code-agent runs
// first (wave 0) and test/review/security agents run in parallel afterwards
// (wave 1), each depending on the code-agent output.
func (p *RulePlanner) buildCodeAnalysisPlan(input PlannerInput, matched []string) *plan.OrchestrationPlan {
	tasks := make([]plan.TaskPlan, 0, len(matched))

	// Separate code-agent from analysis agents.
	var analysisNames []string
	hasCode := false
	for _, name := range matched {
		if name == "code-agent" {
			hasCode = true
		} else if codeAnalysisAgents[name] {
			analysisNames = append(analysisNames, name)
		}
	}

	// Sort analysis agents by stable priority for deterministic ordering.
	sortAnalysisAgents(analysisNames)

	// Wave 0: code-agent (no dependencies, runs first).
	if hasCode {
		tasks = append(tasks, plan.TaskPlan{
			TaskID:          codeTaskID,
			AgentName:       "code-agent",
			CapabilityIDs:   staticCapabilityIDs("code-agent"),
			TaskContent:     input.UserMessage,
			ExpectedOutputs: staticExpectedOutputs("code-agent"),
			DependsOn:       []string{},
			Priority:        1,
			TimeoutMs:       120000,
			RiskLevel:       "low",
		})
	}

	// Wave 1: analysis agents (depend on code-agent output).
	for i, name := range analysisNames {
		promptPrefix := codeAnalysisTaskPrompts[name]
		taskContent := promptPrefix + "{{deps." + codeTaskID + ".output}}"
		tasks = append(tasks, plan.TaskPlan{
			TaskID:          "task_" + name,
			AgentName:       name,
			CapabilityIDs:   staticCapabilityIDs(name),
			TaskContent:     taskContent,
			ExpectedOutputs: staticExpectedOutputs(name),
			DependsOn:       []string{codeTaskID},
			Priority:        analysisAgentPriority[name],
			TimeoutMs:       120000,
			RiskLevel:       "low",
		})

		// Also include any non-analysis, non-code agents (e.g. document-agent)
		// in wave 0 alongside code-agent so they run in parallel.
		_ = i
	}

	// Add any remaining matched agents that are neither code nor analysis
	// (e.g. document-agent, web-agent) in wave 0 alongside code-agent.
	for _, name := range matched {
		if name == "code-agent" || codeAnalysisAgents[name] {
			continue
		}
		tasks = append(tasks, plan.TaskPlan{
			TaskID:          "task_" + name,
			AgentName:       name,
			CapabilityIDs:   staticCapabilityIDs(name),
			TaskContent:     input.UserMessage,
			ExpectedOutputs: staticExpectedOutputs(name),
			DependsOn:       []string{},
			Priority:        1,
			TimeoutMs:       120000,
			RiskLevel:       "low",
		})
	}

	return &plan.OrchestrationPlan{
		Version:        "v1",
		PlanID:         plan.NewPlanID(),
		RunID:          input.RunID,
		ConversationID: input.ConversationID,
		PlanningMode:   input.PlanningMode,
		Strategy:       plan.StrategySequential,
		IntentSummary:  summarize(input.UserMessage),
		TraceID:        input.TraceID,
		Tasks:          tasks,
		Aggregation:    plan.Aggregation{Required: true, Mode: "summary"},
		Fallback:       plan.Fallback{Enabled: true, Reason: "rule_code_analysis"},
		Validation:     plan.Validation{Validated: false},
	}
}

// agentPriority defines stable ordering for multi-agent dispatch.
// Lower = dispatched first. Order mirrors the keyword map for predictability.
var agentPriority = map[string]int{
	"web-agent":      1,
	"code-agent":     2,
	"document-agent": 3,
	"vision-agent":   4,
	"context-agent":  5,
	"test-agent":     6,
	"review-agent":   7,
	"security-agent": 8,
	"deploy-agent":   9,
	"diff-agent":     10,
}

// sortMatchedAgents sorts agent names by their stable priority for deterministic output.
func sortMatchedAgents(names []string) {
	sortSliceStable(names, func(a, b string) int {
		pa := agentPriority[a]
		pb := agentPriority[b]
		if pa < pb {
			return -1
		}
		if pa > pb {
			return 1
		}
		return 0
	})
}

// sortAnalysisAgents sorts analysis agent names by their stable priority for
// deterministic task ordering within wave 1.
func sortAnalysisAgents(names []string) {
	sortSliceStable(names, func(a, b string) int {
		pa := analysisAgentPriority[a]
		pb := analysisAgentPriority[b]
		if pa < pb {
			return -1
		}
		if pa > pb {
			return 1
		}
		return 0
	})
}

func sortSliceStable(slice []string, less func(a, b string) int) {
	for i := 1; i < len(slice); i++ {
		j := i
		for j > 0 && less(slice[j], slice[j-1]) < 0 {
			slice[j], slice[j-1] = slice[j-1], slice[j]
			j--
		}
	}
}

// Ensure RulePlanner implements Planner.
var _ Planner = (*RulePlanner)(nil)
