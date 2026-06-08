import { useState, useRef, useMemo } from 'react'
import { Send, Square, AtSign } from 'lucide-react'

interface Props {
  onSend: (content: string, mentions: string[]) => void
  onStop?: () => void
  disabled?: boolean
  streaming?: boolean
  knownAgentNames?: string[]
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

export default function MessageInput({ onSend, onStop, disabled, streaming, knownAgentNames }: Props) {
  const [text, setText] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const mentions = useMemo(() => extractMentions(text, knownAgentNames), [text, knownAgentNames])

  const handleSubmit = () => {
    const trimmed = text.trim()
    if (!trimmed || disabled) return
    onSend(trimmed, mentions)
    setText('')
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <div className="border-t border-gray-200 p-4 bg-white">
      <div className="flex items-end gap-2 max-w-3xl mx-auto">
        <textarea
          ref={textareaRef}
          value={text}
          onChange={(e) => {
            setText(e.target.value)
            // Auto-resize textarea
            e.target.style.height = 'auto'
            e.target.style.height = `${Math.min(e.target.scrollHeight, 200)}px`
          }}
          onKeyDown={handleKeyDown}
          placeholder="Type a message... Use @agent-name to mention an agent. Enter to send."
          rows={1}
          className="flex-1 resize-none rounded-xl border border-gray-300 px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-shadow"
          disabled={disabled || streaming}
        />
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
        <div className="max-w-3xl mx-auto mt-1.5 flex items-center gap-1.5 text-xs text-indigo-600">
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
