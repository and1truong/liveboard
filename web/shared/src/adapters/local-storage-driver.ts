// Minimal key/value abstraction so LocalAdapter tests can inject an in-memory store.
export interface StorageDriver {
  get(key: string): string | null
  set(key: string, value: string): void
  remove(key: string): void
  keys(prefix: string): string[]
}

export class MemoryStorage implements StorageDriver {
  private readonly map = new Map<string, string>()

  get(key: string): string | null {
    return this.map.has(key) ? this.map.get(key)! : null
  }

  set(key: string, value: string): void {
    this.map.set(key, value)
  }

  remove(key: string): void {
    this.map.delete(key)
  }

  keys(prefix: string): string[] {
    const out: string[] = []
    for (const k of this.map.keys()) {
      if (k.startsWith(prefix)) out.push(k)
    }
    return out
  }
}

interface WebStorage {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
  key(index: number): string | null
  readonly length: number
}

export class BrowserStorage implements StorageDriver {
  constructor(private readonly storage: WebStorage = globalThis.localStorage as WebStorage) {}

  get(key: string): string | null {
    return this.storage.getItem(key)
  }

  set(key: string, value: string): void {
    this.storage.setItem(key, value)
  }

  remove(key: string): void {
    this.storage.removeItem(key)
  }

  keys(prefix: string): string[] {
    const out: string[] = []
    for (let i = 0; i < this.storage.length; i++) {
      const k = this.storage.key(i)
      if (k !== null && k.startsWith(prefix)) out.push(k)
    }
    return out
  }
}
