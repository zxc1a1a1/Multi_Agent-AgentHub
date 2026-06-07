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

export const DEFAULT_AGENT_NAME: AgentName = 'auto'

export const AGENT_OPTIONS: AgentOption[] = [
  {
    name: 'auto',
    displayName: 'Auto (Smart)',
    description: 'Automatically selects the best agent for your request',
    outputModes: ['text', 'code', 'webpage', 'html', 'artifact_ref'],
  },
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
  {
    name: 'document-agent',
    displayName: 'Document Agent',
    description: 'Generates structured documentation, API docs, READMEs, and technical manuals',
    outputModes: ['text', 'markdown', 'artifact_ref'],
  },
  {
    name: 'vision-agent',
    displayName: 'Vision Agent',
    description: 'Analyzes images, extracts text via OCR, and audits visual content',
    outputModes: ['text', 'structured_json', 'artifact_ref'],
  },
  {
    name: 'context-agent',
    displayName: 'Context Agent',
    description: 'Compresses conversation context, generates summaries, and extracts memories',
    outputModes: ['text', 'structured_json', 'summary'],
  },
  {
    name: 'test-agent',
    displayName: 'Test Agent',
    description: 'Analyzes test logs, assesses coverage, and performs root cause analysis',
    outputModes: ['text', 'structured_json', 'analysis_report'],
  },
  {
    name: 'review-agent',
    displayName: 'Review Agent',
    description: 'Reviews code, requirements, and assesses project risks',
    outputModes: ['text', 'structured_json', 'review_report'],
  },
  {
    name: 'security-agent',
    displayName: 'Security Agent',
    description: 'Scans code for vulnerabilities, checks dependencies, audits configs, and detects secrets',
    outputModes: ['text', 'structured_json', 'security_report'],
  },
  {
    name: 'deploy-agent',
    displayName: 'Deploy Agent',
    description: 'Generates deployment plans, checks environment health, and creates rollback strategies',
    outputModes: ['text', 'structured_json', 'deploy_plan'],
  },
  {
    name: 'diff-agent',
    displayName: 'Diff Agent',
    description: 'Generates unified diffs, explains changes, analyzes impact, and resolves merge conflicts',
    outputModes: ['text', 'code', 'diff', 'artifact_ref'],
  },
]

const knownDisplayNameMap: Record<string, string> = {
  'auto': 'Auto (Smart)',
  'code-agent': 'Code Agent',
  'web-agent': 'Web Agent',
  'document-agent': 'Document Agent',
  'vision-agent': 'Vision Agent',
  'context-agent': 'Context Agent',
  'test-agent': 'Test Agent',
  'review-agent': 'Review Agent',
  'security-agent': 'Security Agent',
  'deploy-agent': 'Deploy Agent',
  'diff-agent': 'Diff Agent',
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
  // Always start with Auto as the first option.
  const autoOption = AGENT_OPTIONS[0]
  const options: AgentOption[] = [{ ...autoOption, outputModes: [...autoOption.outputModes] }]
  const seen = new Set<string>(['auto'])

  if (!Array.isArray(summaries) || summaries.length === 0) {
    // No server summaries; append remaining AGENT_OPTIONS after auto.
    for (let i = 1; i < AGENT_OPTIONS.length; i++) {
      const option = AGENT_OPTIONS[i]
      options.push({ ...option, outputModes: [...option.outputModes] })
    }
    return options
  }

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

  return options
}

export function isSupportedAgentName(value: string | undefined | null): value is AgentName {
  if (!value) return false
  const normalized = normalizeText(value)
  if (!normalized) return false
  return normalized === 'auto' || normalized in knownDisplayNameMap
}

export function isConcreteAgentName(value: string | undefined | null): value is AgentName {
  if (!value) return false
  const normalized = normalizeText(value)
  if (!normalized || normalized === 'auto') return false
  return normalized in knownDisplayNameMap
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
