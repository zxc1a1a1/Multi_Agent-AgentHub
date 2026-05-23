import { useEffect, useRef } from 'react'
import { useMessageStore } from '../stores/messageStore'
import { useSendMessage } from '../agui/events'
import MessageBubble from './MessageBubble'
import MessageInput from './MessageInput'
import { Bot } from 'lucide-react'

interface Props {
  conversationId: string
}

export default function ChatWindow({ conversationId }: Props) {
  const messages = useMessageStore((s) => s.messages[conversationId] || [])
  const loadMessages = useMessageStore((s) => s.loadMessages)
  const { sendMessage, streaming, stopStreaming } = useSendMessage()
  const bottomRef = useRef<HTMLDivElement>(null)

  // Load messages when conversation changes
  useEffect(() => {
    loadMessages(conversationId)
  }, [conversationId, loadMessages])

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSend = (content: string) => {
    sendMessage(conversationId, content)
  }

  return (
    <div className="flex flex-col h-full">
      {/* Messages area */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto space-y-6">
          {messages.length === 0 && (
            <div className="text-center text-gray-400 mt-20">
              <Bot className="w-12 h-12 mx-auto mb-4 text-indigo-300" />
              <p className="text-lg font-medium text-gray-600">Start a conversation</p>
              <p className="text-sm mt-1">Ask me to write, explain, or review code</p>
            </div>
          )}
          {messages.map((msg) => (
            <MessageBubble key={msg.id} message={msg} />
          ))}
          <div ref={bottomRef} />
        </div>
      </div>

      {/* Input */}
      <MessageInput
        onSend={handleSend}
        onStop={stopStreaming}
        streaming={streaming}
      />
    </div>
  )
}
