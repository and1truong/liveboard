let markedPromise: Promise<typeof import('marked')> | null = null

export async function renderMarkdown(src: string): Promise<string> {
  markedPromise ??= import('marked')
  const { marked } = await markedPromise
  const html = (await marked.parse(src, { async: false })) as string
  return sanitize(html)
}

const SCRIPT_RE = /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi
const ON_ATTR_RE = /\son[a-z]+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)/gi
const IFRAME_RE = /<\/?iframe\b[^>]*>/gi

export function sanitize(html: string): string {
  return html.replace(SCRIPT_RE, '').replace(ON_ATTR_RE, '').replace(IFRAME_RE, '')
}
