import { useEffect, useState } from 'react'
import DOMPurify from 'dompurify'

interface Props {
  code: string
}

let sequence = 0

/**
 * Mermaid is loaded on demand. securityLevel=strict disables click handlers and
 * HTML labels; the resulting SVG is additionally sanitized with the SVG profile
 * before it reaches dangerouslySetInnerHTML.
 */
export default function MermaidDiagram({ code }: Props) {
  const [svg, setSvg] = useState('')
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let cancelled = false
    setSvg('')
    setFailed(false)
    const id = `agenthub-mermaid-${++sequence}`
    import('mermaid')
      .then(async (module) => {
        const mermaid = module.default
        mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: 'dark' })
        try {
          const result = await mermaid.render(id, code)
          if (cancelled) return
          const clean = DOMPurify.sanitize(result.svg, {
            USE_PROFILES: { svg: true, svgFilters: true },
          })
          setSvg(clean)
        } catch {
          if (!cancelled) setFailed(true)
        }
      })
      .catch(() => {
        if (!cancelled) setFailed(true)
      })
    return () => {
      cancelled = true
    }
  }, [code])

  if (failed || !svg) {
    return (
      <pre className="p-3 overflow-x-auto bg-gray-900 text-xs text-gray-200 rounded-lg my-2">
        <code>{code}</code>
      </pre>
    )
  }
  return (
    <div
      className="mermaid-diagram my-2 overflow-x-auto rounded-lg bg-white p-2"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  )
}
