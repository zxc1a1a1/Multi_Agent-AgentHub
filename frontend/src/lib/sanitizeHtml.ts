// Allowlist sanitizer for syntax-highlighter output. highlight.js escapes code,
// but this keeps the dangerouslySetInnerHTML sink limited to hljs spans if the
// highlighter output is ever influenced by untrusted language metadata.
const ALLOWED_TAGS = new Set(['SPAN', 'BR'])
const HLJS_CLASS = /^hljs-[\w-]+$/

export function sanitizeHighlightHtml(html: string): string {
  if (!html) return ''
  const template = document.createElement('template')
  template.innerHTML = html

  const walker = document.createTreeWalker(template.content, NodeFilter.SHOW_ELEMENT)
  const disallowed: Element[] = []
  while (walker.nextNode()) {
    const element = walker.currentNode as Element
    if (!ALLOWED_TAGS.has(element.tagName)) {
      disallowed.push(element)
      continue
    }
    for (const attribute of Array.from(element.attributes)) {
      if (attribute.name !== 'class' || !HLJS_CLASS.test(attribute.value)) {
        element.removeAttribute(attribute.name)
      }
    }
  }
  for (const element of disallowed) {
    element.replaceWith(document.createTextNode(element.textContent || ''))
  }
  return template.innerHTML
}
