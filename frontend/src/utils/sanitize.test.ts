import { describe, expect, it } from 'vitest'
import { sanitizeHtml } from './sanitize'

describe('sanitizeHtml', () => {
  it('keeps the allowlisted formatting', () => {
    expect(sanitizeHtml('<b>Bold</b> <i>it</i><br><ul><li>x</li></ul>')).toBe('<b>Bold</b> <i>it</i><br><ul><li>x</li></ul>')
  })
  it('drops scripts, styles and event handlers', () => {
    const out = sanitizeHtml('<p onclick="alert(1)">hi<script>alert(1)</script><style>p{}</style></p><img src=x onerror=alert(1)>')
    expect(out).toBe('<p>hi</p>')
  })
  it('keeps only http(s) links and opens them safely', () => {
    expect(sanitizeHtml('<a href="https://x.org/p">site</a>')).toBe(
      '<a href="https://x.org/p" target="_blank" rel="noopener noreferrer">site</a>',
    )
    expect(sanitizeHtml('<a href="javascript:alert(1)">bad</a>')).toBe('<a>bad</a>')
  })
  it('unwraps unknown tags to their text', () => {
    expect(sanitizeHtml('<font color=red>Red <u>u</u></font>')).toBe('Red <u>u</u>')
  })
  it('escapes plain text', () => {
    expect(sanitizeHtml('a &lt;b&gt; c')).toBe('a &lt;b&gt; c')
  })
})
