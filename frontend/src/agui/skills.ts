// Registered runtime capabilities for the frontend.
// Maps to tool names the Gateway may emit via TOOL_CALL_START/TOOL_CALL_ARGS/TOOL_CALL_END.
// Adding new capabilities here does not register a new component — the message store
// matches toolName and artifact.type to decide which preview block to create.
export const frontendSkills: string[] = [
  'code_preview',
  'web_preview',
  'markdown_render',
  'terminal_output',
  'diff_preview',
  'deploy_status',
  'chart_render',
  'file_download',
  'image_preview',
]
