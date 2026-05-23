import { useState, useRef } from 'react'
import { Send, Square } from 'lucide-react'

interface Props {
  onSend: (content: string) => void
  onStop?: () => void
  disabled?: boolean
  streaming?: boolean
}

export default function MessageInput({ onSend, onStop, disabled, streaming }: Props) {
  const [text, setText] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const handleSubmit = () => {
    const trimmed = text.trim()
    if (!trimmed || disabled) return
    onSend(trimmed)
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
          placeholder="Type a message... (Enter to send, Shift+Enter for new line)"
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
    </div>
  )
}
