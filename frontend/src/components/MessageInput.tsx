import { useState, useRef, useMemo, useCallback, useEffect } from 'react'
import { Send, Square, AtSign, X, Reply, Quote } from 'lucide-react'
import MentionDropdown from './MentionDropdown'
import type { ReplyTo, Quote as QuoteType } from '../types'

interface Props {
  onSend: (content: string, mentions: string[]) => void
  onStop?: () => void
  disabled?: boolean
  streaming?: boolean
  knownAgentNames?: string[]
  replyTo?: ReplyTo
  quote?: QuoteType
  onCancelReply?: () => void
  onCancelQuote?: () => void
}

const MENTION_RE = /@([a-zA-Z][a-zA-Z0-9_-]*)/g

function extractMentions(text: string, knownAgentNames?: string[]): string[] {
  const matches = text.match(MENTION_RE)
  if (!matches) return []
  const names = matches.map((m) => m.slice(1))
  const unique = [...new Set(names)]
  // If knownAgentNames is provided, filter to known agents only.
  if (knownAgentNames && knownAgentNames.length > 0) {
    return unique.filter((n) => knownAgentNames.includes(n))
  }
  return unique
}

export default function MessageInput({
  onSend,
  onStop,
  disabled,
  streaming,
  knownAgentNames,
  replyTo,
  quote,
  onCancelReply,
  onCancelQuote,
}: Props) {
  const [text, setText] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // @mention autocomplete state
  const [mentionActive, setMentionActive] = useState(false)
  const [mentionQuery, setMentionQuery] = useState('')
  const [mentionStartPos, setMentionStartPos] = useState(-1)

  const mentions = useMemo(() => extractMentions(text, knownAgentNames), [text, knownAgentNames])

  // Detect @ trigger from textarea cursor position
  const detectMentionTrigger = useCallback((value: string, cursorPos: number) => {
    // Find the last @ before cursor that is not part of a completed mention
    let atIdx = -1
    for (let i = cursorPos - 1; i >= 0; i--) {
      const ch = value[i]
      if (ch === '@') {
        // Check if @ is at start or preceded by whitespace
        if (i === 0 || /\s/.test(value[i - 1])) {
          atIdx = i
          break
        }
      }
      // Stop at whitespace (don't search across words)
      if (/\s/.test(ch) && atIdx === -1) {
        break
      }
    }

    if (atIdx >= 0) {
      const query = value.slice(atIdx + 1, cursorPos)
      // Only show dropdown if query doesn't contain whitespace (not a completed mention)
      if (!/\s/.test(query)) {
        setMentionActive(true)
        setMentionQuery(query)
        setMentionStartPos(atIdx)
        return
      }
    }
    setMentionActive(false)
    setMentionQuery('')
    setMentionStartPos(-1)
  }, [])

  const handleMentionSelect = useCallback(
    (name: string) => {
      if (mentionStartPos >= 0) {
        const before = text.slice(0, mentionStartPos)
        const after = text.slice(mentionStartPos + 1 + mentionQuery.length)
        const newText = `${before}@${name} ${after}`
        setText(newText)
        setMentionActive(false)
        setMentionQuery('')
        setMentionStartPos(-1)
        // Focus back on textarea and set cursor after the inserted mention
        setTimeout(() => {
          if (textareaRef.current) {
            const cursorPos = mentionStartPos + name.length + 2 // @name + space
            textareaRef.current.focus()
            textareaRef.current.setSelectionRange(cursorPos, cursorPos)
          }
        }, 0)
      }
    },
    [text, mentionStartPos, mentionQuery],
  )

  // Close mention dropdown on blur (deferred to allow click on dropdown items)
  const handleBlur = useCallback(() => {
    setTimeout(() => {
      setMentionActive(false)
    }, 150)
  }, [])

  const handleSubmit = () => {
    const trimmed = text.trim()
    if (!trimmed || disabled) return
    onSend(trimmed, mentions)
    setText('')
    setMentionActive(false)
    setMentionQuery('')
    setMentionStartPos(-1)
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // If mention dropdown is open, let arrow keys and Enter navigate/select
    if (mentionActive) {
      if (e.key === 'Escape') {
        e.preventDefault()
        setMentionActive(false)
        return
      }
      if (e.key === 'Enter' && !e.shiftKey) {
        // Don't submit when dropdown is open — user is selecting a mention
        e.preventDefault()
        return
      }
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <div className="border-t border-gray-200 p-4 bg-white">
      {/* Reply/Quote banners */}
      {(replyTo || quote) && (
        <div className="max-w-3xl mx-auto mb-2 space-y-1.5">
          {replyTo && (
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-indigo-50 border border-indigo-200 text-xs">
              <Reply className="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" />
              <span className="text-gray-500">Replying to</span>
              <span className="font-medium text-indigo-700">{replyTo.author}</span>
              <span className="text-gray-400 truncate flex-1">
                — "{replyTo.contentPreview}"
              </span>
              <button
                onClick={onCancelReply}
                className="p-0.5 rounded text-gray-400 hover:text-gray-600 hover:bg-indigo-100 flex-shrink-0"
                title="Cancel reply"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          )}
          {quote && (
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-amber-50 border border-amber-200 text-xs">
              <Quote className="w-3.5 h-3.5 text-amber-500 flex-shrink-0" />
              <span className="text-gray-500">Quoting</span>
              <span className="font-medium text-amber-700">{quote.author}</span>
              <span className="text-gray-400 truncate flex-1">
                — "{quote.text.slice(0, 80)}"
              </span>
              <button
                onClick={onCancelQuote}
                className="p-0.5 rounded text-gray-400 hover:text-gray-600 hover:bg-amber-100 flex-shrink-0"
                title="Cancel quote"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          )}
        </div>
      )}
      <div ref={containerRef} className="flex items-end gap-2 max-w-3xl mx-auto">
        <div className="relative flex-1">
          <textarea
            ref={textareaRef}
            value={text}
            onChange={(e) => {
              setText(e.target.value)
              const cursorPos = e.target.selectionStart || 0
              detectMentionTrigger(e.target.value, cursorPos)
              // Auto-resize textarea
              e.target.style.height = 'auto'
              e.target.style.height = `${Math.min(e.target.scrollHeight, 200)}px`
            }}
            onKeyUp={(e) => {
              // Handle cursor movement (arrow keys)
              const cursorPos = (e.target as HTMLTextAreaElement).selectionStart || 0
              detectMentionTrigger((e.target as HTMLTextAreaElement).value, cursorPos)
            }}
            onClick={(e) => {
              const cursorPos = (e.target as HTMLTextAreaElement).selectionStart || 0
              detectMentionTrigger((e.target as HTMLTextAreaElement).value, cursorPos)
            }}
            onKeyDown={handleKeyDown}
            onBlur={handleBlur}
            placeholder="Type a message... Use @agent-name to mention an agent. Enter to send."
            rows={1}
            className="w-full resize-none rounded-xl border border-gray-300 px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-shadow"
            disabled={disabled || streaming}
          />
          {/* Mention autocomplete dropdown */}
          {mentionActive && (
            <MentionDropdown
              query={mentionQuery}
              onSelect={handleMentionSelect}
              knownAgentNames={knownAgentNames}
            />
          )}
        </div>
        {streaming ? (
          <button
            onClick={onStop}
            className="p-3 rounded-xl bg-red-500 text-white hover:bg-red-600 transition-colors flex-shrink-0"
            title="Stop generation"
          >
            <Square className="w-5 h-5" />
          </button>
        ) : (
          <button
            onClick={handleSubmit}
            disabled={!text.trim() || disabled}
            className="p-3 rounded-xl bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex-shrink-0"
            title="Send message"
          >
            <Send className="w-5 h-5" />
          </button>
        )}
      </div>
      {mentions.length > 0 && (
        <div data-testid="mentioning-indicator" className="max-w-3xl mx-auto mt-1.5 flex items-center gap-1.5 text-xs text-indigo-600">
          <AtSign className="w-3 h-3" />
          <span>Mentioning:</span>
          {mentions.map((m) => (
            <span key={m} className="px-1.5 py-0.5 rounded bg-indigo-50 border border-indigo-200 font-medium">
              @{m}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
