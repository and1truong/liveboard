let markedPromise: Promise<typeof import('marked')> | null = null

export interface RenderOptions {
  // attachmentResolver returns the resolved URL for an `attachment:<hash>`
  // reference. Server mode: synchronous string. Local mode: async blob URL.
  // null/missing → leave the URL as-is (will not load).
  attachmentResolver?: (hash: string) => string | Promise<string>
}

export async function renderMarkdown(src: string, opts?: RenderOptions): Promise<string> {
  markedPromise ??= import('marked')
  const { marked } = await markedPromise
  let html = (await marked.parse(src, { async: false })) as string
  if (opts?.attachmentResolver) {
    html = await rewriteAttachmentURLs(html, opts.attachmentResolver)
  }
  return sanitize(html)
}

const ATTACHMENT_URL_RE = /\bsrc="attachment:([a-f0-9]{64}(?:\.[a-z0-9]{1,16})?)"|\bhref="attachment:([a-f0-9]{64}(?:\.[a-z0-9]{1,16})?)"/g

// rewriteAttachmentURLs walks the rendered HTML, finds `attachment:<hash>`
// URLs in <img src="..."> and <a href="..."> attributes, and replaces them
// with the resolved URL from the resolver. Pre-fetches all unique hashes
// in parallel so the final substitution is sync.
async function rewriteAttachmentURLs(
  html: string,
  resolver: (hash: string) => string | Promise<string>,
): Promise<string> {
  const hashes = new Set<string>()
  for (const m of html.matchAll(ATTACHMENT_URL_RE)) {
    const hash = m[1] ?? m[2]
    if (hash) hashes.add(hash)
  }
  if (hashes.size === 0) return html

  const resolved = new Map<string, string>()
  await Promise.all(
    [...hashes].map(async (h) => {
      try {
        resolved.set(h, await resolver(h))
      } catch {
        resolved.set(h, '')
      }
    }),
  )

  return html.replace(ATTACHMENT_URL_RE, (_match, srcHash: string | undefined, hrefHash: string | undefined) => {
    const hash = srcHash ?? hrefHash ?? ''
    const url = resolved.get(hash) ?? ''
    if (srcHash !== undefined) return `src="${url}"`
    return `href="${url}"`
  })
}

const SCRIPT_RE = /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi
const ON_ATTR_RE = /\son[a-z]+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)/gi
const IFRAME_RE = /<\/?iframe\b[^>]*>/gi

export function sanitize(html: string): string {
  return html.replace(SCRIPT_RE, '').replace(ON_ATTR_RE, '').replace(IFRAME_RE, '')
}
