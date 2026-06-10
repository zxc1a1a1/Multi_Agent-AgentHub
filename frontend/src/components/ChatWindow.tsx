import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useMessageStore } from '../stores/messageStore'
import { useConversationStore } from '../stores/conversationStore'
import { useAgentStore } from '../stores/agentStore'
import { useSendMessage } from '../agui/events'
import MessageBubble from './MessageBubble'
import MessageInput from './MessageInput'
import OrchestrationCard from './OrchestrationCard'
import ActivitySnapshotRenderer, { type PlanApprovalStatus } from './ActivitySnapshotRenderer'
import AgentPicker from './AgentPicker'
import AgentAvatar from './AgentAvatar'
import { useActivityStore } from '../stores/activityStore'
import { Bot, Users } from 'lucide-react'
import { normalizeAgentName, type AgentName } from '../lib/agents'
import type { ReplyTo, Quote } from '../types'

interface Props {
  conversationId: string
}

export default function ChatWindow({ conversationId }: Props) {
  const messages = useMessageStore((s) => s.messages[conversationId] || [])
  const orchestration = useMessageStore((s) => s.orchestrationByConversation[conversationId])
  const confirmation = useMessageStore((s) => s.confirmationByConversation[conversationId])
  const getConfirmation = useMessageStore((s) => s.getConfirmation)
  const confirmPlan = useMessageStore((s) => s.confirmPlan)
  const clearConfirmation = useMessageStore((s) => s.clearConfirmation)
  const loadMessages = useMessageStore((s) => s.loadMessages)
  const togglePinMessage = useMessageStore((s) => s.togglePinMessage)
  const regenerateMessage = useMessageStore((s) => s.regenerateMessage)
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
  const [selectedAgentNames, setSelectedAgentNames] = useState<AgentName[]>(() => {
    const name = normalizeAgentName(conversationAgentName || defaultAgentName())
    return [name]
  })
  const [confirmError, setConfirmError] = useState<string | null>(null)
  const [confirmStatus, setConfirmStatus] = useState<PlanApprovalStatus>('waiting')
  const [replyTo, setReplyTo] = useState<ReplyTo | undefined>(undefined)
  const [quote, setQuote] = useState<Quote | undefined>(undefined)

  const selectedAgentOption = useMemo(
    () => findAgentOption(selectedAgentNames[0] || 'auto'),
    [findAgentOption, selectedAgentNames],
  )

  // Resolve display info for all selected agents (for participant avatars)
  const selectedAgentInfos = useMemo(
    () =>
      selectedAgentNames.map((name) => {
        const opt = agentOptions.find((o) => o.name === name)
        return {
          name,
          displayName: opt?.displayName || name,
        }
      }),
    [selectedAgentNames, agentOptions],
  )

  const isGroupChat = selectedAgentNames.length > 1
  const pinnedMessages = useMemo(() => messages.filter((m) => m.pinned), [messages])

  // Load messages when conversation changes.
  useEffect(() => {
    loadMessages(conversationId)
  }, [conversationId, loadMessages])

  useEffect(() => {
    loadAgents()
  }, [loadAgents])

  useEffect(() => {
    const name = normalizeAgentName(conversationAgentName || defaultAgentName())
    setSelectedAgentNames([name])
  }, [conversationAgentName, conversationId, defaultAgentName])

  // Auto-scroll to bottom on new messages. Use instant scroll during streaming
  // to avoid smooth-scroll lag that can make content appear truncated.
  useEffect(() => {
    const el = bottomRef.current
    if (!el) return
    el.scrollIntoView({ behavior: streaming ? 'auto' : 'smooth', block: 'end' })
  }, [messages, streaming])

  // Reset confirmStatus when confirmation transitions back to pending
  // (e.g. new plan arrives after a revise).
  useEffect(() => {
    if (confirmation && confirmation.status === 'pending') {
      setConfirmStatus('waiting')
    }
  }, [confirmation])

  const handleSend = (content: string, mentions: string[]) => {
    setConfirmError(null)
    const effectiveAgentNames =
      mentions.length > 0
        ? mentions
        : selectedAgentNames.filter((n) => n !== 'auto')
    const primaryAgent = selectedAgentNames[0] || 'auto'
    sendMessage(conversationId, content, {
      agentName: primaryAgent,
      mentions,
      selectedAgentNames: effectiveAgentNames,
      replyTo,
      quote,
    })
    // Clear reply/quote after sending
    setReplyTo(undefined)
    setQuote(undefined)
  }

  const handleTogglePin = useCallback((message: import('../types').Message) => {
    togglePinMessage(conversationId, message.id)
  }, [conversationId, togglePinMessage])

  const handleRegenerate = useCallback((message: import('../types').Message) => {
    regenerateMessage(conversationId, message.id).catch(() => {})
  }, [conversationId, regenerateMessage])

  const handleReply = useCallback((rt: ReplyTo) => {
    setReplyTo(rt)
    setQuote(undefined)
  }, [])

  const handleCancelReply = useCallback(() => {
    setReplyTo(undefined)
  }, [])

  const handleCancelQuote = useCallback(() => {
    setQuote(undefined)
  }, [])

  const handleConfirm = useCallback(
    async (selectedParticipants?: string[]) => {
      setConfirmError(null)
      const pending = getConfirmation(conversationId)
      if (!pending) return
      setConfirmStatus('approving')
      try {
        await confirmPlan(conversationId, pending.runId, pending.actionId, true, undefined, undefined, undefined, undefined, selectedParticipants)
      } catch (err) {
        setConfirmStatus('waiting')
        setConfirmError(err instanceof Error ? err.message : 'Confirmation failed')
      }
    },
    [conversationId, getConfirmation, confirmPlan],
  )

  const handleCancel = useCallback(
    async (actionId: string, reason: string) => {
      setConfirmError(null)
      const pending = getConfirmation(conversationId)
      if (!pending) return
      setConfirmStatus('cancelling')
      try {
        await confirmPlan(conversationId, pending.runId, actionId, false, reason)
      } catch (err) {
        setConfirmStatus('waiting')
        setConfirmError(err instanceof Error ? err.message : 'Cancellation failed')
      }
    },
    [conversationId, getConfirmation, confirmPlan],
  )

  const handleRevise = useCallback(
    async (feedback: string, selectedParticipants?: string[]) => {
      setConfirmError(null)
      const pending = getConfirmation(conversationId)
      if (!pending) return
      setConfirmStatus('revising')
      try {
        await confirmPlan(
          conversationId,
          pending.runId,
          pending.actionId,
          false,
          '',
          'revise',
          feedback,
          pending.revision,
          selectedParticipants,
        )
      } catch (err) {
        setConfirmStatus('waiting')
        setConfirmError(err instanceof Error ? err.message : 'Revision failed')
      }
    },
    [conversationId, getConfirmation, confirmPlan],
  )

  const handleTimeout = useCallback(
    (actionId: string) => {
      const pending = getConfirmation(conversationId)
      if (!pending) return
      // Auto-reject on timeout.
      confirmPlan(conversationId, pending.runId, actionId, false, 'Confirmation timed out').catch(() => {})
    },
    [conversationId, getConfirmation, confirmPlan],
  )

  // Show plan approval UI when confirmation is pending.
  // ActivitySnapshotRenderer reads from activityStore; confirmationByConversation
  // tracks the lifecycle state. Both are populated by ACTIVITY_SNAPSHOT events.
  const pendingConfirmation = confirmation && confirmation.status === 'pending' ? confirmation : null

  return (
    <div className="flex flex-col h-full">
      {/* Plan Approval Card (rendered via ActivitySnapshotRenderer) */}
      {pendingConfirmation && (
        <div className="px-4 py-2 bg-amber-50/30 border-b border-amber-200">
          <ActivitySnapshotRenderer
            conversationId={conversationId}
            status={confirmStatus}
            onApprove={(selectedParticipants) => handleConfirm(selectedParticipants)}
            onCancel={() => handleCancel(pendingConfirmation.actionId, 'User cancelled')}
            onRevise={(feedback, selectedParticipants) => handleRevise(feedback, selectedParticipants)}
          />
          {confirmError && (
            <div className="mt-2 p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700">
              {confirmError}
              <button
                className="ml-2 underline text-red-600 hover:text-red-800"
                onClick={() => setConfirmError(null)}
              >
                Dismiss
              </button>
            </div>
          )}
        </div>
      )}

      {/* Confirmed/rejected status banner */}
      {confirmation && confirmation.status === 'confirmed' && (
        <div className="px-4 py-1.5 bg-green-50 border-b border-green-200 text-sm text-green-700 text-center">
          Plan confirmed — executing...
          <button
            className="ml-2 underline text-green-600 hover:text-green-800"
            onClick={() => clearConfirmation(conversationId)}
          >
            Dismiss
          </button>
        </div>
      )}
      {confirmation && confirmation.status === 'rejected' && (
        <div className="px-4 py-1.5 bg-gray-50 border-b border-gray-200 text-sm text-gray-600 text-center">
          Plan rejected{confirmation.rejectReason ? `: ${confirmation.rejectReason}` : ''}
          <button
            className="ml-2 underline text-gray-500 hover:text-gray-700"
            onClick={() => clearConfirmation(conversationId)}
          >
            Dismiss
          </button>
        </div>
      )}
      {confirmation && confirmation.status === 'cancelled' && (
        <div className="px-4 py-1.5 bg-gray-50 border-b border-gray-200 text-sm text-gray-600 text-center">
          Plan cancelled{confirmation.rejectReason ? `: ${confirmation.rejectReason}` : ''}
          <button
            className="ml-2 underline text-gray-500 hover:text-gray-700"
            onClick={() => clearConfirmation(conversationId)}
          >
            Dismiss
          </button>
        </div>
      )}

      {pinnedMessages.length > 0 && (
        <div className="px-4 py-2 border-b border-yellow-200 bg-yellow-50/60">
          <div className="max-w-3xl mx-auto text-xs text-yellow-800">
            <span className="font-semibold">Pinned context:</span>{' '}
            {pinnedMessages.slice(0, 3).map((msg) => (
              <span key={msg.id} className="inline-block ml-2 px-2 py-0.5 rounded bg-white border border-yellow-200 max-w-[14rem] truncate align-bottom">
                {msg.content.slice(0, 80)}
              </span>
            ))}
            {pinnedMessages.length > 3 && <span className="ml-2">+{pinnedMessages.length - 3} more</span>}
          </div>
        </div>
      )}

      {/* Messages area */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto space-y-6">
          {messages.length === 0 && !streaming && !orchestration && (
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
            <MessageBubble
              key={msg.id}
              message={msg}
              onReply={handleReply}
              onTogglePin={handleTogglePin}
              onRegenerate={handleRegenerate}
            />
          ))}
          {/* Only show skeleton when streaming has started but no agent message bubble exists yet. */}
          {streaming && !messages.some((msg) => msg.status === 'streaming') && (
            <div className="flex gap-3">
              {/* Avatar placeholder — matches AgentAvatar size */}
              <div className="w-8 h-8 rounded-full bg-indigo-100 flex-shrink-0 flex items-center justify-center">
                <span className="w-3 h-3 bg-indigo-300 rounded-full animate-pulse" />
              </div>
              {/* Skeleton bubble */}
              <div className="max-w-[80%] min-w-0 flex flex-col items-start">
                <p className="text-xs text-gray-400 mb-1 px-1">{selectedAgentOption?.displayName || 'Agent'}</p>
                <div className="inline-block px-4 py-3 rounded-2xl rounded-tl-sm bg-gray-100">
                  <div className="flex items-center gap-2">
                    <span className="flex gap-1">
                      <span className="w-1.5 h-1.5 bg-indigo-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                      <span className="w-1.5 h-1.5 bg-indigo-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                      <span className="w-1.5 h-1.5 bg-indigo-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                    </span>
                    <span className="text-xs text-gray-500">Thinking…</span>
                  </div>
                </div>
              </div>
            </div>
          )}
          <div ref={bottomRef} />
        </div>
      </div>

      <div className="px-4 py-2 border-t border-gray-200 bg-gray-50">
        <div className="max-w-3xl mx-auto flex items-center gap-3">
          <label className="text-xs text-gray-600 whitespace-nowrap">
            Agent
          </label>
          <AgentPicker
            selectedAgentNames={selectedAgentNames}
            onChange={setSelectedAgentNames}
            disabled={streaming}
          />
          {isGroupChat && (
            <div className="flex items-center gap-1 ml-1">
              <Users className="w-3.5 h-3.5 text-indigo-400" />
              <div className="flex -space-x-1.5">
                {selectedAgentInfos.map((info) => (
                  <div
                    key={info.name}
                    title={info.displayName}
                    className="w-6 h-6 rounded-full bg-indigo-100 border-2 border-white flex items-center justify-center"
                  >
                    <span className="text-[10px] font-medium text-indigo-600">
                      {info.displayName.charAt(0).toUpperCase()}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
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
        knownAgentNames={agentOptions.map((o) => o.name)}
        replyTo={replyTo}
        quote={quote}
        onCancelReply={handleCancelReply}
        onCancelQuote={handleCancelQuote}
      />
    </div>
  )
}
