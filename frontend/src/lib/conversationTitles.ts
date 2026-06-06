/**
 * Lightweight localStorage persistence for conversation titles.
 *
 * Key: agenthub.conversationTitles.v1
 * Value: JSON-serialized Record<conversationId, title>
 *
 * All reads and writes handle missing storage, private mode errors,
 * and corrupted JSON gracefully.
 */

const STORAGE_KEY = 'agenthub.conversationTitles.v1'

function readRaw(): string | null {
  try {
    return localStorage.getItem(STORAGE_KEY)
  } catch {
    // storage unavailable (private mode, quota, etc.)
    return null
  }
}

function writeRaw(value: string): boolean {
  try {
    localStorage.setItem(STORAGE_KEY, value)
    return true
  } catch {
    return false
  }
}

export function loadTitles(): Record<string, string> {
  const raw = readRaw()
  if (!raw) {
    return {}
  }
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {}
    }
    const result: Record<string, string> = {}
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof value === 'string' && value) {
        result[key] = value
      }
    }
    return result
  } catch {
    // JSON parse error — corrupted data
    return {}
  }
}

function saveTitles(titles: Record<string, string>): boolean {
  try {
    const json = JSON.stringify(titles)
    return writeRaw(json)
  } catch {
    return false
  }
}

/**
 * Persist a single conversation title to localStorage.
 * Merges with existing titles — only overwrites the given conversationId.
 */
export function persistTitle(conversationId: string, title: string): void {
  if (!conversationId || !title) {
    return
  }
  const titles = loadTitles()
  titles[conversationId] = title
  saveTitles(titles)
}

/**
 * Overlay local titles onto a list of conversations from Gateway.
 * Local titles take precedence ONLY when the Gateway title is empty
 * or still the default "New Conversation" placeholder.
 * This ensures Gateway-authoritative titles are respected when available.
 */
export function overlayLocalTitles<T extends { id: string; title: string }>(
  conversations: T[],
): T[] {
  const localTitles = loadTitles()
  if (Object.keys(localTitles).length === 0) {
    return conversations
  }
  return conversations.map((conv) => {
    const localTitle = localTitles[conv.id]
    if (!localTitle) {
      return conv
    }
    // Only overlay when Gateway hasn't provided a real title.
    const isGatewayTitleEmpty = !conv.title || conv.title === 'New Conversation'
    if (isGatewayTitleEmpty) {
      return { ...conv, title: localTitle }
    }
    return conv
  })
}
