import { useMemo, useState, useCallback } from 'react'
import type { Message, SkillCardData, ReplyTo } from '../types'
import AgentAvatar from './AgentAvatar'
import StreamingText from './StreamingText'
import CodePreview from './CodePreview'
import WebPreview from './WebPreview'
import TerminalOutput from './skills/TerminalOutput'
import DiffPreview from './skills/DiffPreview'
import DeployStatusCard from './skills/DeployStatusCard'
import ChartRender from './skills/ChartRender'
import FileDownloadCard from './skills/FileDownloadCard'
import ImagePreview from './skills/ImagePreview'
import UnknownSkillFallback from './skills/UnknownSkillFallback'
import ConfirmActionCard from './skills/ConfirmActionCard'
import FormInputCard from './skills/FormInputCard'
import ArtifactCard from './skills/ArtifactCard'
import OrchestrationSummary from './OrchestrationSummary'
import { User, Reply, Copy, Check, RotateCcw, Pin, PinOff } from 'lucide-react'
import { getAgentDisplayName } from '../lib/agents'
import { useAgentStore } from '../stores/agentStore'

interface Props {
  message: Message
  onReply?: (replyTo: ReplyTo) => void
  onRegenerate?: (message: Message) => void
  onTogglePin?: (message: Message) => void
}

/**
 * Maps SkillCardData union type to the correct display component.
 * Wraps each card in a lightweight error boundary.
 */
function SkillCardRenderer({ card }: { card: SkillCardData }) {
  try {
    switch (card.type) {
      case 'terminal_output':
        return <TerminalOutput command={card.command} stdout={card.stdout} stderr={card.stderr} exitCode={card.exitCode} />
      case 'diff_preview':
        return <DiffPreview diffText={card.diffText} filename={card.filename} />
      case 'deploy_status':
        return <DeployStatusCard environment={card.environment} version={card.version} status={card.status} timestamp={card.timestamp} />
      case 'chart_render':
        return <ChartRender chartType={card.chartType} data={card.data} />
      case 'file_download':
        return <FileDownloadCard filename={card.filename} size={card.size} mimeType={card.mimeType} createdAt={card.createdAt} />
      case 'image_preview':
        return <ImagePreview url={card.url} alt={card.alt} />
      case 'confirm_action':
        return <ConfirmActionCard runId={card.runId} taskId={card.taskId} toolCallId={card.toolCallId} title={card.title} message={card.message} riskLevel={card.riskLevel} />
      case 'form_input':
        return <FormInputCard runId={card.runId} taskId={card.taskId} toolCallId={card.toolCallId} title={card.title} schema={card.schema} />
      case 'artifact_card':
        return <ArtifactCard id={card.id} name={card.name} kind={card.kind} mimeType={card.mimeType} size={card.size} downloadPath={card.downloadPath} createdAt={card.createdAt} sourceAgent={card.sourceAgent} metadata={card.metadata} />
      case 'unknown_skill':
        return <UnknownSkillFallback toolName={card.toolName} args={card.args} />
      default:
        return <UnknownSkillFallback toolName={(card as SkillCardData).type} />
    }
  } catch {
    return <UnknownSkillFallback toolName="render_error" args={{ error: 'Component render failed', type: card.type }} />
  }
}

function buildReplyTo(message: Message): ReplyTo {
  const author =
    message.senderType === 'user'
      ? 'You'
      : message.senderName || getAgentDisplayName(message.agentName) || 'Agent'
  const contentPreview = message.content.slice(0, 100)
  return {
    id: message.id,
    author,
    senderType: message.senderType,
    contentPreview,
  }
}

export default function MessageBubble({ message, onReply, onRegenerate, onTogglePin }: Props) {
  const isUser = message.senderType === 'user'
  const isStreaming = message.status === 'streaming'
  const [hover, setHover] = useState(false)
  const [copied, setCopied] = useState(false)

  const senderLabel = isUser
    ? 'You'
    : message.senderName || getAgentDisplayName(message.agentName)

  // Resolve agent capabilities from agentStore for capability tags
  const agentManagement = useAgentStore((s) => s.agents)
  const capabilityTags = useMemo(() => {
    if (isUser || !message.agentName) return []
    const agent = agentManagement.find(
      (a) => a.name === message.agentName,
    )
    return agent?.capabilities || []
  }, [isUser, message.agentName, agentManagement])

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(message.content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Clipboard API not available — silently ignore
    }
  }, [message.content])

  const handleReply = useCallback(() => {
    if (onReply && !isStreaming && message.status !== 'failed') {
      onReply(buildReplyTo(message))
    }
  }, [onReply, isStreaming, message])

  const handleRegenerate = useCallback(() => {
    if (onRegenerate && !isUser && !isStreaming && message.status === 'sent') {
      onRegenerate(message)
    }
  }, [onRegenerate, isUser, isStreaming, message])

  const handleTogglePin = useCallback(() => {
    if (onTogglePin && !isStreaming && message.status !== 'failed') {
      onTogglePin(message)
    }
  }, [onTogglePin, isStreaming, message])

  return (
    <div
      className={`flex gap-3 ${isUser ? 'flex-row-reverse' : ''}`}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      {/* Avatar */}
      {isUser ? (
        <div className="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center flex-shrink-0">
          <User className="w-5 h-5 text-gray-600" />
        </div>
      ) : (
        <AgentAvatar name={message.agentName} isStreaming={isStreaming} />
      )}

      {/* Content */}
      <div className={`max-w-[80%] min-w-0 ${isUser ? 'items-end' : 'items-start'} flex flex-col`}>
        {!isUser && (
          <div className="flex items-center gap-1.5 mb-1 px-1">
            <p className="text-xs text-gray-500">{senderLabel}</p>
            {capabilityTags.length > 0 && (
              <div className="flex gap-0.5">
                {capabilityTags.map((cap) => (
                  <span
                    key={cap}
                    className="text-[10px] px-1.5 py-0.5 rounded-full bg-indigo-50 text-indigo-600 border border-indigo-100"
                  >
                    {cap}
                  </span>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Reply-to preview */}
        {message.replyTo && (
          <div className={`mb-1 px-3 py-1.5 rounded-lg border border-gray-200 bg-white text-xs ${isUser ? 'text-right' : 'text-left'}`}>
            <span className="text-gray-400">Replying to </span>
            <span className="font-medium text-gray-600">{message.replyTo.author}</span>
            <span className="text-gray-400">: </span>
            <span className="text-gray-500 italic">"{message.replyTo.contentPreview}"</span>
          </div>
        )}

        {message.pinned && (
          <div className="mb-1 px-2 py-0.5 rounded-full bg-yellow-50 border border-yellow-200 text-[10px] text-yellow-700 inline-flex items-center gap-1">
            <Pin className="w-3 h-3" /> pinned
          </div>
        )}

        <div
          className={`inline-block px-4 py-2.5 rounded-2xl ${
            isUser
              ? 'bg-indigo-600 text-white rounded-tr-sm'
              : 'bg-gray-100 text-gray-900 rounded-tl-sm'
          }`}
        >
          {isUser ? (
            <p className="text-sm whitespace-pre-wrap break-words overflow-hidden m-0">{message.content}</p>
          ) : (
            <StreamingText content={message.content} isStreaming={isStreaming} />
          )}
        </div>

        {/* Action bar (hover) */}
        {hover && !isStreaming && message.status !== 'failed' && (
          <div className={`flex items-center gap-1 mt-1 ${isUser ? 'flex-row-reverse' : ''}`}>
            <button
              onClick={handleReply}
              className="p-1 rounded text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 transition-colors"
              title="Reply"
            >
              <Reply className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={handleTogglePin}
              className="p-1 rounded text-gray-400 hover:text-yellow-600 hover:bg-yellow-50 transition-colors"
              title={message.pinned ? 'Unpin message' : 'Pin message'}
            >
              {message.pinned ? <PinOff className="w-3.5 h-3.5" /> : <Pin className="w-3.5 h-3.5" />}
            </button>
            {!isUser && (
              <button
                onClick={handleRegenerate}
                className="p-1 rounded text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 transition-colors"
                title="Regenerate"
              >
                <RotateCcw className="w-3.5 h-3.5" />
              </button>
            )}
            <button
              onClick={handleCopy}
              className="p-1 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
              title="Copy message"
            >
              {copied ? (
                <Check className="w-3.5 h-3.5 text-green-500" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
            </button>
          </div>
        )}

        {/* Skill cards */}
        {message.skillCards && message.skillCards.length > 0 && (
          <div className="w-full mt-1">
            {message.skillCards.map((card, i) => (
              <SkillCardRenderer key={`${message.id}-skill-${i}`} card={card} />
            ))}
          </div>
        )}

        {/* Orchestration summary */}
        {message.orchestrationSummary && (
          <div className="w-full mt-1">
            <OrchestrationSummary {...message.orchestrationSummary} />
          </div>
        )}

        {/* Code blocks */}
        {message.codeBlocks && message.codeBlocks.length > 0 && (
          <div className="w-full mt-1">
            {message.codeBlocks.map((block, i) => (
              <CodePreview key={`${message.id}-code-${i}`} block={block} />
            ))}
          </div>
        )}

        {message.webPreviews && message.webPreviews.length > 0 && (
          <div className="w-full mt-1">
            {message.webPreviews.map((block, i) => (
              <WebPreview key={`${message.id}-web-${i}`} block={block} />
            ))}
          </div>
        )}

        {/* Error indicator */}
        {message.status === 'failed' && (
          <div className="mt-1 p-2 bg-red-50 border border-red-200 rounded text-xs">
            <p className="text-red-600 font-medium">
              {message.errorCode ? `Error: ${message.errorCode}` : 'Error'}
            </p>
            {message.errorMessage && message.errorMessage !== 'Error' && (
              <p className="text-red-500 mt-0.5">{message.errorMessage}</p>
            )}
            {(!message.errorMessage || message.errorMessage === 'Error') && (
              <p className="text-red-500 mt-0.5">Failed to generate response</p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
