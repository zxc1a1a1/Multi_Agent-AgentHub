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
import OrchestrationSummary from './OrchestrationSummary'
import { User, Reply, Copy, Check } from 'lucide-react'
import { getAgentDisplayName } from '../lib/agents'
import { useAgentStore } from '../stores/agentStore'

interface Props {
  message: Message
  onReply?: (replyTo: ReplyTo) => void
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

export default function MessageBubble({ message, onReply }: Props) {
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
