/**
 * R3-01 交互组件协议类型。
 *
 * 事实源：`docs/contracts/rich-artifact-output.md` §11（`ui.component` schemaVersion 1）。
 *
 * `props` 是来自子 Agent 的不可信数据：只能作为纯数据交给白名单渲染器，
 * 禁止 `eval` / `new Function` / `Function` 构造器 / 由 props 提供 URL 的动态
 * `import()` / `dangerouslySetInnerHTML`。校验与 registry 见 `frontend/src/agui/skills.ts`。
 */

export type UIComponentName = 'html_sandbox' | 'json_form' | 'chart' | 'code_editor'

export type UIComponentCapability = 'read_only' | 'user_input' | 'sandboxed_script'

/** 当前冻结的组件 schema 版本；严格等于 1 才合法。 */
export type UIComponentSchemaVersion = 1

/** props 只允许 JSON 纯数据对象。 */
export type UIComponentProps = Record<string, unknown>

/**
 * `ui.component` AG-UI 事件（SSE 载荷）。
 * 除本节协议字段外，事件可携带标准 AG-UI 游标字段用于幂等去重。
 */
export interface UIComponentAGUIEvent {
  type: 'ui.component'
  component: UIComponentName
  props: UIComponentProps
  schemaVersion: UIComponentSchemaVersion
  artifactId: string
  capabilities: UIComponentCapability[]
  runId?: string
  messageId?: string
  taskId?: string
  /** SSE eventId；优先作为去重键。 */
  id?: string
  seq?: number
  subIndex?: number
  traceId?: string
}

/** 校验拒绝原因码；对外只暴露原因码，不回显原始 payload。 */
export type UIComponentRejectionReason =
  | 'event_shape_invalid'
  | 'component_not_allowed'
  | 'props_invalid'
  | 'props_too_large'
  | 'schema_version_unsupported'
  | 'artifact_id_invalid'
  | 'capabilities_invalid'

export type UIComponentValidation =
  | {
      ok: true
      component: UIComponentName
      props: UIComponentProps
      schemaVersion: UIComponentSchemaVersion
      artifactId: string
      capabilities: UIComponentCapability[]
      /** registry 中登记的渲染器标识；null 表示协议已接受但渲染器尚未实现（R3-01）。 */
      renderer: string | null
    }
  | {
      ok: false
      reason: UIComponentRejectionReason
      /** 清洗后的组件名 token，仅用于降级卡片展示。 */
      component?: string
    }

/** 降级卡片展示用的白名单参数（不含 props 原文）。 */
export interface UIComponentFallbackArgs {
  reason: UIComponentRejectionReason
  component: string
  artifactId: string
  schemaVersion: string
}

/**
 * 合法且已注册组件对应的技能卡片。
 * R3-01 未注册任何渲染器，因此该卡片由 `MessageBubble` 的 default 分支降级为
 * 文本卡片；R3-02/R3-04/R3-05 接入渲染器时在此类型上补字段，不改事件协议。
 */
export interface UIComponentCardData {
  type: 'ui_component'
  component: UIComponentName
  props: UIComponentProps
  schemaVersion: UIComponentSchemaVersion
  artifactId: string
  capabilities: UIComponentCapability[]
  runId?: string
  taskId?: string
  messageId?: string
  eventId?: string
  seq?: number
  subIndex?: number
}
