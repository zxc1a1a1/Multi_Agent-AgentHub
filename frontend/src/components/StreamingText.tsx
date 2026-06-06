import { useState, useCallback } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Copy, Check } from 'lucide-react'

interface Props {
  content: string
  isStreaming?: boolean
}

function CodeBlockHeader({ language, code }: { language: string; code: string }) {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(code)
    } catch {
      const ta = document.createElement('textarea')
      ta.value = code
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [code])

  return (
    <div className="flex items-center justify-between px-4 py-1.5 bg-gray-800 text-gray-300 text-xs rounded-t-lg">
      <span className="font-mono">{language || 'code'}</span>
      <button
        onClick={handleCopy}
        className="flex items-center gap-1 px-2 py-0.5 rounded hover:bg-gray-700 transition-colors"
        title="Copy code"
      >
        {copied ? (
          <>
            <Check className="w-3 h-3 text-green-400" />
            <span className="text-green-400">Copied</span>
          </>
        ) : (
          <>
            <Copy className="w-3 h-3" />
            <span>Copy</span>
          </>
        )}
      </button>
    </div>
  )
}

export default function StreamingText({ content, isStreaming }: Props) {
  if (!content && isStreaming) {
    return (
      <div className="flex items-center gap-2 text-gray-400 text-sm py-1">
        <span className="flex gap-1">
          <span className="w-1.5 h-1.5 bg-indigo-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
          <span className="w-1.5 h-1.5 bg-indigo-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
          <span className="w-1.5 h-1.5 bg-indigo-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
        </span>
        <span className="text-xs">Thinking…</span>
      </div>
    )
  }

  return (
    <div className="message-content">
      {isStreaming ? (
        // During streaming: render as plain text to avoid Markdown parse flickering
        <span className="whitespace-pre-wrap text-sm break-words">
          {content}
          <span className="inline-block w-1.5 h-4 bg-indigo-500 animate-pulse ml-0.5 align-middle" />
        </span>
      ) : (
        // After completion: render with full Markdown support.
        // overflow-x-hidden + break-words prevent long URLs/code from breaking layout.
        <div className="prose prose-sm max-w-none prose-p:my-1 prose-pre:my-2 break-words overflow-x-hidden">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              pre: ({ children, ...props }) => {
                // Extract code content and language from children.
                const codeElement = children as React.ReactElement | undefined
                const codeText =
                  codeElement?.props?.children
                    ? (typeof codeElement.props.children === 'string'
                        ? codeElement.props.children
                        : String(codeElement.props.children))
                    : ''
                const className: string = codeElement?.props?.className || ''
                const langMatch = className.match(/language-(\w+)/)
                const language = langMatch ? langMatch[1] : ''

                return (
                  <div className="rounded-lg border border-gray-700 overflow-hidden my-2">
                    <CodeBlockHeader language={language} code={codeText} />
                    <pre className="p-4 overflow-x-auto bg-gray-900 text-sm leading-relaxed m-0 rounded-b-lg" {...props}>
                      {children}
                    </pre>
                  </div>
                )
              },
            }}
          >
            {content}
          </ReactMarkdown>
        </div>
      )}
    </div>
  )
}
