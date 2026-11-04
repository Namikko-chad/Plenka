// Tiny allowlist sanitizer for indexer "info" settings (HTML from tracker definitions).
// Anything not allowed is unwrapped to its text; links keep only http(s) hrefs.

const ALLOWED = new Set(['B', 'STRONG', 'I', 'EM', 'U', 'BR', 'P', 'UL', 'OL', 'LI', 'A', 'CODE', 'SPAN', 'DIV'])
const DROP = new Set(['SCRIPT', 'STYLE', 'IFRAME', 'OBJECT', 'EMBED', 'TEMPLATE', 'NOSCRIPT', 'SVG', 'MATH', 'HEAD', 'TITLE'])

function clean(src: Node, dst: Node, doc: Document): void {
  for (const child of Array.from(src.childNodes)) {
    if (child.nodeType === 3) {
      dst.appendChild(doc.createTextNode(child.textContent ?? ''))
      continue
    }
    if (child.nodeType !== 1) continue
    const el = child as Element
    const tag = el.tagName.toUpperCase()
    if (DROP.has(tag)) continue
    if (!ALLOWED.has(tag)) {
      clean(el, dst, doc)
      continue
    }
    const out = doc.createElement(tag.toLowerCase())
    if (tag === 'A') {
      const href = el.getAttribute('href') ?? ''
      if (/^https?:\/\//i.test(href.trim())) {
        out.setAttribute('href', href.trim())
        out.setAttribute('target', '_blank')
        out.setAttribute('rel', 'noopener noreferrer')
      }
    }
    clean(el, out, doc)
    dst.appendChild(out)
  }
}

export function sanitizeHtml(html: string): string {
  const doc = new DOMParser().parseFromString(`<body>${html}</body>`, 'text/html')
  const out = doc.createElement('div')
  clean(doc.body, out, doc)
  return out.innerHTML
}
