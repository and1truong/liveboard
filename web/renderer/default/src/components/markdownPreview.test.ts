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

describe('renderMarkdown attachment URL rewriting', () => {
  it('rewrites attachment: URLs in img src via resolver', async () => {
    const html = await renderMarkdown('![](attachment:aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111.png)', {
      attachmentResolver: async (hash) => `/api/v1/attachments/${hash}/x.png`,
    })
    expect(html).toContain('src="/api/v1/attachments/aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111.png/x.png"')
  })

  it('rewrites attachment: URLs in anchor href via resolver', async () => {
    const html = await renderMarkdown('[doc](attachment:bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222.pdf)', {
      attachmentResolver: (hash) => `/x/${hash}`,
    })
    expect(html).toContain('href="/x/bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222.pdf"')
  })

  it('without resolver, attachment: URLs are sanitized but not resolved', async () => {
    const html = await renderMarkdown('![](attachment:aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111.png)')
    expect(html).toContain('attachment:')
  })
})
