import { describe, expect, it } from 'bun:test'
import { renderMarkdown, sanitize } from './markdownPreview.js'

describe('renderMarkdown', () => {
  it('renders headings and bold', async () => {
    const html = await renderMarkdown('# Hello\n\n**bold**')
    expect(html).toContain('<h1>Hello</h1>')
    expect(html).toContain('<strong>bold</strong>')
  })

  it('strips script tags', async () => {
    const html = await renderMarkdown('<script>alert(1)</script>safe')
    expect(html).not.toContain('<script>')
    expect(html).toContain('safe')
  })

  it('strips on* attributes', async () => {
    const html = await renderMarkdown('<a href="x" onclick="bad()">link</a>')
    expect(html).not.toContain('onclick')
    expect(html).toContain('href="x"')
  })

  it('strips iframe tags', async () => {
    const html = await renderMarkdown('<iframe src="x"></iframe>after')
    expect(html).not.toContain('iframe')
    expect(html).toContain('after')
  })
})

describe('sanitize', () => {
  it('handles double-quoted attrs', () => {
    expect(sanitize('<a onmouseover="x">a</a>')).not.toContain('onmouseover')
  })
  it('handles single-quoted attrs', () => {
    expect(sanitize("<a onmouseover='x'>a</a>")).not.toContain('onmouseover')
  })
})
