import type { Attachment } from '../types.js'

const DB_NAME = 'liveboard-attachments'
const STORE = 'blobs'

let dbPromise: Promise<IDBDatabase> | null = null

function openDB(): Promise<IDBDatabase> {
  if (dbPromise) return dbPromise
  dbPromise = new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1)
    req.onupgradeneeded = (): void => {
      req.result.createObjectStore(STORE)
    }
    req.onsuccess = (): void => resolve(req.result)
    req.onerror = (): void => reject(req.error)
  })
  return dbPromise
}

async function txn<T>(
  mode: IDBTransactionMode,
  fn: (s: IDBObjectStore) => IDBRequest<T>,
): Promise<T> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const t = db.transaction(STORE, mode)
    const s = t.objectStore(STORE)
    const req = fn(s)
    req.onsuccess = (): void => resolve(req.result)
    req.onerror = (): void => reject(req.error)
  })
}

async function sha256Hex(blob: Blob): Promise<string> {
  const buf = await blob.arrayBuffer()
  const digest = await crypto.subtle.digest('SHA-256', buf)
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

function ext(filename: string): string {
  const i = filename.lastIndexOf('.')
  if (i < 0) return ''
  return filename.slice(i).toLowerCase()
}

// putBlob hashes the blob with SHA-256, derives a hash key (hex + extension
// from `name`), and stores the blob in IndexedDB keyed by that hash.
// Returns an Attachment descriptor matching the wire format used by the
// add_attachments mutation.
export async function putBlob(blob: Blob, name: string): Promise<Attachment> {
  const hex = await sha256Hex(blob)
  const hash = hex + ext(name)
  await txn('readwrite', (s) => s.put(blob, hash))
  // Use only the base MIME type (strip charset and other parameters) so the
  // value is stable across runtimes that auto-append ;charset=utf-8.
  const rawType = blob.type !== '' ? blob.type : 'application/octet-stream'
  const mime = rawType.split(';')[0]!.trim()
  return {
    h: hash,
    n: name,
    s: blob.size,
    m: mime,
  }
}

// getBlob looks up a previously-stored blob by hash. Returns null when not
// present; callers should treat null as "broken/missing reference" and
// render a placeholder.
export async function getBlob(hash: string): Promise<Blob | null> {
  const v = await txn<Blob | undefined>('readonly', (s) => s.get(hash) as IDBRequest<Blob | undefined>)
  return v ?? null
}

// deleteBlob removes the blob keyed by hash. Idempotent: a missing key is a
// no-op (IndexedDB returns success).
export async function deleteBlob(hash: string): Promise<void> {
  await txn('readwrite', (s) => s.delete(hash))
}
