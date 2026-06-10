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
