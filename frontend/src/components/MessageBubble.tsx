import type { Message } from '../types'
import AgentAvatar from './AgentAvatar'
import StreamingText from './StreamingText'
import CodePreview from './CodePreview'
import WebPreview from './WebPreview'
import { User } from 'lucide-react'
import { getAgentDisplayName } from '../lib/agents'

interface Props {
  message: Message
}

export default function MessageBubble({ message }: Props) {
  const isUser = message.senderType === 'user'
  const isStreaming = message.status === 'streaming'
  const senderLabel = isUser
    ? 'You'
    : message.senderName || getAgentDisplayName(message.agentName)

  return (
    <div className={`flex gap-3 ${isUser ? 'flex-row-reverse' : ''}`}>
      {/* Avatar */}
      {isUser ? (
        <div className="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center flex-shrink-0">
          <User className="w-5 h-5 text-gray-600" />
        </div>
      ) : (
        <AgentAvatar isStreaming={isStreaming} />
      )}

      {/* Content */}
      <div className={`max-w-[80%] min-w-0 ${isUser ? 'items-end' : 'items-start'} flex flex-col`}>
        {!isUser && <p className="text-xs text-gray-500 mb-1 px-1">{senderLabel}</p>}
        <div
          className={`inline-block px-4 py-2.5 rounded-2xl ${
            isUser
              ? 'bg-indigo-600 text-white rounded-tr-sm'
              : 'bg-gray-100 text-gray-900 rounded-tl-sm'
          }`}
        >
          {isUser ? (
            <p className="text-sm whitespace-pre-wrap m-0">{message.content}</p>
          ) : (
            <StreamingText content={message.content} isStreaming={isStreaming} />
          )}
        </div>

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
          <p className="text-xs text-red-500 mt-1">Failed to generate response</p>
        )}
      </div>
    </div>
  )
}
