export type AgentName = string

export type AgentOption = {
  name: AgentName
  displayName: string
  description: string
  outputModes: string[]
}

export type AgentSummary = {
  name?: string | null
  displayName?: string | null
  description?: string | null
  outputModes?: string[] | null
}

export const DEFAULT_AGENT_NAME: AgentName = 'code-agent'

export const AGENT_OPTIONS: AgentOption[] = [
  {
    name: 'code-agent',
    displayName: 'Code Agent',
    description: 'Generates and explains code',
    outputModes: ['text', 'code', 'artifact_ref'],
  },
  {
    name: 'web-agent',
    displayName: 'Web Agent',
    description: 'Generates webpages and HTML previews',
    outputModes: ['text', 'webpage', 'html', 'artifact_ref'],
  },
]

const knownDisplayNameMap: Record<string, string> = {
  'code-agent': 'Code Agent',
  'web-agent': 'Web Agent',
}

function normalizeText(value: string | null | undefined): string {
  if (typeof value !== 'string') {
    return ''
  }
  return value.trim()
}

function humanizeAgentName(name: string): string {
  const trimmed = normalizeText(name)
  if (!trimmed) {
    return 'Assistant'
  }
  return trimmed
    .split('-')
    .filter((part) => part !== '')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function normalizeOutputModes(outputModes: string[] | null | undefined): string[] {
  if (!Array.isArray(outputModes)) {
    return []
  }
  const modes = outputModes
    .map((value) => normalizeText(value))
    .filter((value) => value !== '')
  return Array.from(new Set(modes))
}

function fallbackOptionByName(name: string): AgentOption | undefined {
  return AGENT_OPTIONS.find((option) => option.name === name)
}

export function buildAgentOptionsFromSummary(summaries: AgentSummary[] | null | undefined): AgentOption[] {
  if (!Array.isArray(summaries) || summaries.length === 0) {
    return AGENT_OPTIONS.map((option) => ({ ...option, outputModes: [...option.outputModes] }))
  }

  const options: AgentOption[] = []
  const seen = new Set<string>()

  for (const summary of summaries) {
    const name = normalizeText(summary?.name || '')
    if (!name || seen.has(name)) {
      continue
    }
    seen.add(name)

    const fallbackOption = fallbackOptionByName(name)
    const displayName =
      normalizeText(summary?.displayName || '') ||
      fallbackOption?.displayName ||
      getAgentDisplayName(name)
    const description =
      normalizeText(summary?.description || '') ||
      fallbackOption?.description ||
      'Supports text responses'
    const outputModes = normalizeOutputModes(summary?.outputModes)

    options.push({
      name,
      displayName,
      description,
      outputModes: outputModes.length > 0 ? outputModes : fallbackOption?.outputModes || ['text'],
    })
  }

  if (options.length === 0) {
    return AGENT_OPTIONS.map((option) => ({ ...option, outputModes: [...option.outputModes] }))
  }
  return options
}

export function isSupportedAgentName(value: string | undefined | null): value is 'code-agent' | 'web-agent' {
  return value === 'code-agent' || value === 'web-agent'
}

export function normalizeAgentName(value: string | undefined | null): AgentName {
  const normalized = normalizeText(value || '')
  if (normalized === '') {
    return DEFAULT_AGENT_NAME
  }
  return normalized
}

export function getAgentDisplayName(value: string | undefined | null): string {
  const normalized = normalizeText(value || '')
  if (normalized === '') {
    return 'Assistant'
  }
  return knownDisplayNameMap[normalized] || humanizeAgentName(normalized)
}
