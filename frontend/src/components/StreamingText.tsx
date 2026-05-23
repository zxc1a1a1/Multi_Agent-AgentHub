import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface Props {
  content: string
  isStreaming?: boolean
}

export default function StreamingText({ content, isStreaming }: Props) {
  if (!content && isStreaming) {
    return (
      <div className="flex items-center gap-1 text-gray-400 text-sm">
        <span>Thinking</span>
        <span className="animate-pulse">...</span>
      </div>
    )
  }

  return (
    <div className="message-content">
      {isStreaming ? (
        // During streaming: render as plain text to avoid Markdown parse flickering
        <span className="whitespace-pre-wrap text-sm">
          {content}
          <span className="inline-block w-1.5 h-4 bg-indigo-500 animate-pulse ml-0.5 align-middle" />
        </span>
      ) : (
        // After completion: render with full Markdown support
        <div className="prose prose-sm max-w-none prose-p:my-1 prose-pre:my-2">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {content}
          </ReactMarkdown>
        </div>
      )}
    </div>
  )
}
