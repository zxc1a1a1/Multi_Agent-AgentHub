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

interface StyledBlock {
  type: 'text' | 'code'
  content: string
  language?: string
}

/**
 * Parse streaming content into alternating text and code blocks.
 * Detects open/closed fenced code blocks so code renders progressively.
 */
function parseStreamingBlocks(content: string): StyledBlock[] {
  const blocks: StyledBlock[] = []
  const lines = content.split('\n')
  let currentText = ''
  let currentCode = ''
  let codeLang = ''
  let inCodeBlock = false

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const fenceMatch = line.match(/^```(\w*)$/)

    if (!inCodeBlock && fenceMatch) {
      // Enter code block: flush accumulated text first.
      if (currentText) {
        blocks.push({ type: 'text', content: currentText })
        currentText = ''
      }
      inCodeBlock = true
      codeLang = fenceMatch[1] || ''
      continue
    }

    if (inCodeBlock) {
      const closingFence = line.match(/^```\s*$/)
      if (closingFence) {
        // Close code block, flush it.
        if (currentCode || blocks.length === 0 || blocks[blocks.length - 1].type !== 'code') {
          blocks.push({ type: 'code', content: currentCode, language: codeLang })
        }
        currentCode = ''
        codeLang = ''
        inCodeBlock = false
        continue
      }
      currentCode += line + '\n'
      continue
    }

    currentText += line + '\n'
  }

  // Flush remaining content.
  if (inCodeBlock) {
    // Open code fence at end of stream — render it progressively.
    blocks.push({ type: 'code', content: currentCode, language: codeLang })
  } else if (currentText) {
    blocks.push({ type: 'text', content: currentText })
  }

  return blocks
}

/**
 * Streaming text component.
 *
 * During streaming: renders plain text outside code fences, and styled
 * code blocks for fenced sections that are open (not yet closed). This
 * lets code, markdown code blocks, and HTML output stream progressively.
 *
 * After streaming: full Markdown render with syntax-highlighted code blocks.
 */
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

  if (!isStreaming) {
    return (
      <div className="message-content">
        <div className="prose prose-sm max-w-none prose-p:my-1 prose-pre:my-2 break-words overflow-x-hidden">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              pre: ({ children, ...props }) => {
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
      </div>
    )
  }

  // Streaming mode: progressive rendering with code blocks.
  const blocks = parseStreamingBlocks(content)

  return (
    <div className="message-content">
      <div className="text-sm break-words">
        {blocks.map((block, index) => {
          if (block.type === 'code') {
            const code = block.content || ''
            const lang = block.language || ''
            return (
              <div key={index} className="rounded-lg border border-gray-700 overflow-hidden my-2">
                <CodeBlockHeader language={lang || 'code'} code={code} />
                <pre className="p-4 overflow-x-auto bg-gray-900 text-sm leading-relaxed m-0 rounded-b-lg">
                  <code className={`whitespace-pre-wrap ${lang ? `language-${lang}` : ''}`}>
                    {code}
                  </code>
                </pre>
              </div>
            )
          }
          return (
            <span key={index} className="whitespace-pre-wrap">
              {block.content}
            </span>
          )
        })}
        <span className="inline-block w-1.5 h-4 bg-indigo-500 animate-pulse ml-0.5 align-middle" />
      </div>
    </div>
  )
}
