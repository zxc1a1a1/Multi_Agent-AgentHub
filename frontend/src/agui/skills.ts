import type {
  UIComponentCapability,
  UIComponentFallbackArgs,
  UIComponentName,
  UIComponentProps,
  UIComponentRejectionReason,
  UIComponentValidation,
} from '../types/uiComponent'

export type JsonSchema = Record<string, unknown>

export type FrontendSkillDefinition = {
  name: string
  description: string
  parameters: JsonSchema
  interactive: boolean
  renderer: string
}

// Registered runtime capabilities for the frontend. File upload/download and
// image message full-chain are intentionally not implemented in Phase 8; cards
// remain metadata/safe-url display only.
export const frontendSkillDefinitions: FrontendSkillDefinition[] = [
  { name: 'code_preview', description: 'Render code artifacts', parameters: { type: 'object' }, interactive: false, renderer: 'code_preview' },
  { name: 'web_preview', description: 'Render safe HTML preview artifacts', parameters: { type: 'object' }, interactive: false, renderer: 'web_preview' },
  { name: 'markdown_render', description: 'Render markdown content', parameters: { type: 'object' }, interactive: false, renderer: 'markdown_render' },
  { name: 'terminal_output', description: 'Render terminal output', parameters: { type: 'object' }, interactive: false, renderer: 'terminal_output' },
  { name: 'diff_preview', description: 'Render diff preview with dry-run/apply controls', parameters: { type: 'object' }, interactive: false, renderer: 'diff_preview' },
  { name: 'deploy_status', description: 'Render deployment status metadata', parameters: { type: 'object' }, interactive: false, renderer: 'deploy_status' },
  { name: 'chart_render', description: 'Render small chart data', parameters: { type: 'object' }, interactive: false, renderer: 'chart_render' },
  { name: 'file_download', description: 'Render file metadata only; no upload/download transport', parameters: { type: 'object' }, interactive: false, renderer: 'file_download' },
  { name: 'image_preview', description: 'Render safe image URL metadata only', parameters: { type: 'object' }, interactive: false, renderer: 'image_preview' },
  { name: 'confirm_action', description: 'Interactive confirmation that posts tool-result', parameters: { type: 'object' }, interactive: true, renderer: 'confirm_action' },
  { name: 'form_input', description: 'Interactive JSON-schema form that posts tool-result', parameters: { type: 'object' }, interactive: true, renderer: 'form_input' },
  { name: 'artifact_metadata', description: 'Render artifact metadata card', parameters: { type: 'object' }, interactive: false, renderer: 'artifact_card' },
]

export const frontendSkills: string[] = frontendSkillDefinitions.map((skill) => skill.name)

export function getFrontendSkillNames(): string[] {
  return [...frontendSkills]
}

export function getFrontendSkillDefinition(name: string): FrontendSkillDefinition | undefined {
  return frontendSkillDefinitions.find((skill) => skill.name === name)
}

// ---------------------------------------------------------------------------
// R3-01 交互组件协议：白名单 registry + 解析/校验。
// 事实源 docs/contracts/rich-artifact-output.md §11（ui.component schemaVersion 1）。
// 组件渲染器由 R3-02/R3-04/R3-05 接入；本阶段 renderer 为 null，组件事件一律降级为
// 文本卡片。任何路径都不得执行 props：禁止 eval / new Function / Function 构造器 /
// 由 props 提供 URL 的动态 import / dangerouslySetInnerHTML。
// ---------------------------------------------------------------------------

export const UI_COMPONENT_EVENT_TYPE = 'ui.component'

export const UI_COMPONENT_SCHEMA_VERSION = 1

/** props 序列化后的 UTF-8 字节上限（64KB）。 */
export const UI_COMPONENT_PROPS_MAX_BYTES = 64 * 1024

/** props 嵌套深度防御上限，超出视为非法。 */
export const UI_COMPONENT_PROPS_MAX_DEPTH = 64

/**
 * 既有产物 ID 形状：`art_` + 十六进制/ULID 风格后缀。
 * 生成端见 `pkg/runtime/tools/artifact.go`、`services/orchestrator/httpapi/handler_render.go`，
 * 存储列 `artifact_id VARCHAR(64)`（`docs/contracts/mysql-schema.md`）。
 */
export const UI_COMPONENT_ARTIFACT_ID_PATTERN = /^art_[A-Za-z0-9_-]{1,64}$/

export const uiComponentAllowlist: readonly UIComponentName[] = ['html_sandbox', 'json_form', 'chart', 'code_editor']

export const uiCapabilityAllowlist: readonly UIComponentCapability[] = ['read_only', 'user_input', 'sandboxed_script']

export type UIComponentDefinition = {
  component: UIComponentName
  description: string
  /** null = 协议已接受，渲染器尚未实现（R3-01）；降级为文本卡片。 */
  renderer: string | null
  interactive: boolean
  acceptsStreamingProps: false
}

// Registered interactive components. Adding an entry here is not enough to run
// any code: props stay inert data until a reviewed renderer is wired in.
export const uiComponentRegistry: UIComponentDefinition[] = [
  {
    component: 'html_sandbox',
    description: 'Sandboxed HTML artifact surface (iframe allow-scripts + CSP)',
    renderer: 'html_sandbox',
    interactive: false,
    acceptsStreamingProps: false,
  },
  {
    component: 'json_form',
    description: 'JSON-schema driven form (sanitized RJSF, tool-result submit)',
    renderer: 'json_schema_form',
    interactive: false,
    acceptsStreamingProps: false,
  },
  {
    component: 'chart',
    description: 'Declarative chart surface (renderer lands in R3-05)',
    renderer: null,
    interactive: false,
    acceptsStreamingProps: false,
  },
  {
    component: 'code_editor',
    description: 'Read-mostly code editor surface (post-R3-02 evaluation)',
    renderer: null,
    interactive: false,
    acceptsStreamingProps: false,
  },
]

export function isUIComponentName(value: unknown): value is UIComponentName {
  return typeof value === 'string' && (uiComponentAllowlist as readonly string[]).includes(value)
}

export function isUIComponentCapability(value: unknown): value is UIComponentCapability {
  return typeof value === 'string' && (uiCapabilityAllowlist as readonly string[]).includes(value)
}

export function getUIComponentDefinition(name: UIComponentName): UIComponentDefinition | undefined {
  return uiComponentRegistry.find((entry) => entry.component === name)
}

/**
 * 降级卡片展示用的 token 清洗：数字先转字符串，只保留 [A-Za-z0-9_.-]，
 * 长度 ≤64，含任何其它字符或超长时整体替换为 <redacted>。
 */
export function safeComponentToken(value: unknown): string {
  const text = typeof value === 'number' && Number.isFinite(value) ? String(value) : value
  if (typeof text !== 'string') {
    return '<redacted>'
  }
  if (!/^[A-Za-z0-9_.-]{1,64}$/.test(text)) {
    return '<redacted>'
  }
  return text
}

/**
 * 校验 props 是纯 JSON 数据对象，并返回其 UTF-8 字节长度。
 * 返回 null 表示非法（非纯对象 / 含函数、symbol、bigint / 访问器属性 / 不可序列化 / 深度超限）。
 */
function measureProps(props: unknown): { bytes: number } | null {
  if (!isPlainObject(props)) {
    return null
  }
  if (!hasOnlyJSONValues(props, 0)) {
    return null
  }
  let serialized: string
  try {
    serialized = JSON.stringify(props)
  } catch {
    return null
  }
  if (typeof serialized !== 'string') {
    return null
  }
  return { bytes: new TextEncoder().encode(serialized).length }
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return false
  }
  const proto = Object.getPrototypeOf(value)
  return proto === Object.prototype || proto === null
}

/**
 * props 里不允许出现函数 / symbol / undefined / bigint / 非法原型对象，
 * 也不允许访问器属性（getter/setter）：只读取 own data property 描述符，
 * 因此校验过程本身不会调用任何来自事件的代码。
 */
function hasOnlyJSONValues(value: unknown, depth: number): boolean {
  if (depth > UI_COMPONENT_PROPS_MAX_DEPTH) {
    return false
  }
  if (value === null) {
    return true
  }
  const type = typeof value
  if (type === 'function' || type === 'symbol' || type === 'undefined' || type === 'bigint') {
    return false
  }
  if (type !== 'object') {
    return true
  }
  if (!Array.isArray(value) && !isPlainObject(value)) {
    return false
  }
  return everyOwnDataValue(value, depth)
}

function everyOwnDataValue(value: object, depth: number): boolean {
  for (const name of Object.getOwnPropertyNames(value)) {
    const descriptor = Object.getOwnPropertyDescriptor(value, name)
    if (!descriptor || !('value' in descriptor)) {
      // 访问器属性一律非法：拒绝而不是调用它。
      return false
    }
    if (!hasOnlyJSONValues(descriptor.value, depth + 1)) {
      return false
    }
  }
  return true
}

/**
 * `ui.component` 事件的唯一入口校验。
 * 输入按 unknown 处理：调用方不得绕过本函数直接使用事件字段。
 */
export function validateUIComponentEvent(raw: unknown): UIComponentValidation {
  if (!isPlainObject(raw)) {
    return { ok: false, reason: 'event_shape_invalid', component: '<redacted>' }
  }
  const event = raw
  if (event.type !== UI_COMPONENT_EVENT_TYPE) {
    return { ok: false, reason: 'event_shape_invalid', component: safeComponentToken(event.component) }
  }

  const component = event.component
  if (!isUIComponentName(component)) {
    return { ok: false, reason: 'component_not_allowed', component: safeComponentToken(component) }
  }

  if (event.schemaVersion !== UI_COMPONENT_SCHEMA_VERSION) {
    return { ok: false, reason: 'schema_version_unsupported', component }
  }

  const measured = measureProps(event.props)
  if (!measured) {
    return { ok: false, reason: 'props_invalid', component }
  }
  if (measured.bytes > UI_COMPONENT_PROPS_MAX_BYTES) {
    return { ok: false, reason: 'props_too_large', component }
  }

  if (typeof event.artifactId !== 'string' || !UI_COMPONENT_ARTIFACT_ID_PATTERN.test(event.artifactId)) {
    return { ok: false, reason: 'artifact_id_invalid', component }
  }

  const capabilities = event.capabilities
  if (!Array.isArray(capabilities) || !capabilities.every((item) => isUIComponentCapability(item))) {
    return { ok: false, reason: 'capabilities_invalid', component }
  }

  return {
    ok: true,
    component,
    props: event.props as UIComponentProps,
    schemaVersion: UI_COMPONENT_SCHEMA_VERSION,
    artifactId: event.artifactId,
    capabilities: [...capabilities] as UIComponentCapability[],
    renderer: getUIComponentDefinition(component)?.renderer ?? null,
  }
}

/** 降级文本卡片展示的白名单参数：不含 props 原文。 */
export function describeUIComponentRejection(
  raw: unknown,
  reason: UIComponentRejectionReason,
): UIComponentFallbackArgs {
  const event = isPlainObject(raw) ? raw : {}
  return {
    reason,
    component: safeComponentToken(event.component),
    artifactId: safeComponentToken(event.artifactId),
    schemaVersion: safeComponentToken(event.schemaVersion),
  }
}
