import { useEffect, useMemo, useRef, useState } from 'react'
import { useMessageStore } from '../stores/messageStore'
import { useConversationStore } from '../stores/conversationStore'
import { useAgentStore } from '../stores/agentStore'
import { useSendMessage } from '../agui/events'
import MessageBubble from './MessageBubble'
import MessageInput from './MessageInput'
import OrchestrationCard from './OrchestrationCard'
import { Bot } from 'lucide-react'
import { normalizeAgentName, type AgentName } from '../lib/agents'

interface Props {
  conversationId: string
}

export default function ChatWindow({ conversationId }: Props) {
  const messages = useMessageStore((s) => s.messages[conversationId] || [])
  const orchestration = useMessageStore((s) => s.orchestrationByConversation[conversationId])
  const loadMessages = useMessageStore((s) => s.loadMessages)
  const conversationAgentName = useConversationStore((s) => {
    const conversation = s.conversations.find((item) => item.id === conversationId)
    return conversation?.agentName
  })
  const agentOptions = useAgentStore((s) => s.options)
  const loadAgents = useAgentStore((s) => s.load)
  const findAgentOption = useAgentStore((s) => s.findOption)
  const defaultAgentName = useAgentStore((s) => s.defaultAgentName)
  const { sendMessage, isStreaming, stopStreaming } = useSendMessage()
  const streaming = isStreaming(conversationId)
  const bottomRef = useRef<HTMLDivElement>(null)
  const [selectedAgentName, setSelectedAgentName] = useState<AgentName>(() =>
    normalizeAgentName(conversationAgentName || defaultAgentName()),
  )

  const selectedAgentOption = useMemo(
    () => findAgentOption(selectedAgentName),
    [findAgentOption, selectedAgentName],
  )

  // Load messages when conversation changes.
  useEffect(() => {
    loadMessages(conversationId)
  }, [conversationId, loadMessages])

  useEffect(() => {
    loadAgents()
  }, [loadAgents])

  useEffect(() => {
    setSelectedAgentName(normalizeAgentName(conversationAgentName || defaultAgentName()))
  }, [conversationAgentName, conversationId, defaultAgentName])

  // Auto-scroll to bottom on new messages.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSend = (content: string) => {
    sendMessage(conversationId, content, { agentName: selectedAgentName })
  }

  return (
    <div className="flex flex-col h-full">
      {/* Messages area */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto space-y-6">
          {messages.length === 0 && !orchestration && (
            <div className="text-center text-gray-400 mt-20">
              <Bot className="w-12 h-12 mx-auto mb-4 text-indigo-300" />
              <p className="text-lg font-medium text-gray-600">Start a conversation</p>
              <p className="text-sm mt-1">Ask me to write, explain, or review code</p>
            </div>
          )}
          {orchestration && (
            <OrchestrationCard info={orchestration} />
          )}
          {messages.map((msg) => (
            <MessageBubble key={msg.id} message={msg} />
          ))}
          <div ref={bottomRef} />
        </div>
      </div>

      <div className="px-4 py-2 border-t border-gray-200 bg-gray-50">
        <div className="max-w-3xl mx-auto flex items-center gap-3">
          <label htmlFor="agent-select" className="text-xs text-gray-600 whitespace-nowrap">
            Agent
          </label>
          <select
            id="agent-select"
            value={selectedAgentName}
            onChange={(event) => setSelectedAgentName(normalizeAgentName(event.target.value))}
            className="text-sm rounded-md border border-gray-300 bg-white px-2 py-1 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            disabled={streaming}
          >
            {agentOptions.map((option) => (
              <option key={option.name} value={option.name}>
                {option.displayName}
              </option>
            ))}
          </select>
          <span className="text-xs text-gray-500 truncate">
            {selectedAgentOption?.description || 'Supports text responses'}
          </span>
        </div>
      </div>

      {/* Input */}
      <MessageInput
        onSend={handleSend}
        onStop={() => stopStreaming(conversationId)}
        streaming={streaming}
      />
    </div>
  )
}
