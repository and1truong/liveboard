import type { Card, Attachment } from '@shared/types.js'
import { getBlob } from '@shared/adapters/local-attachments.js'
import type { Client } from '@shared/client.js'

// buildAttachmentResolver returns a resolver function suitable for passing
// to renderMarkdown(src, { attachmentResolver }).
//
// Server mode (Client.attachmentsBaseURL() returns a prefix): builds
// HTTP URLs synchronously.
//
// Local mode (returns null): looks up the blob in IndexedDB and produces
// an object URL via URL.createObjectURL. The card's `attachments` array
// supplies the display-name/MIME so we can construct a valid blob URL
// (the IDB blob's MIME type carries it).
//
// `card` is the card being rendered; its `attachments` list lets us look
// up display names (used in server-mode HTTP URL construction). Hashes
// referenced in body that aren't on the card render with empty-string URL
// (broken-image icon).
export function buildAttachmentResolver(
  client: Client,
  card: Pick<Card, 'attachments'>,
): (hash: string) => string | Promise<string> {
  const base = client.attachmentsBaseURL()
  const byHash = new Map<string, Attachment>()
  for (const a of card.attachments ?? []) {
    byHash.set(a.h, a)
  }

  if (base !== null) {
    return (hash: string): string => {
      const att = byHash.get(hash)
      const name = att?.n ?? hash
      return `${base}/${hash}/${encodeURIComponent(name)}`
    }
  }

  // Local mode: async IDB lookup, cache resulting blob URLs per hash.
  const blobURLCache = new Map<string, string>()
  return async (hash: string): Promise<string> => {
    const cached = blobURLCache.get(hash)
    if (cached !== undefined) return cached
    const blob = await getBlob(hash)
    if (!blob) return ''
    const url = URL.createObjectURL(blob)
    blobURLCache.set(hash, url)
    return url
  }
}
