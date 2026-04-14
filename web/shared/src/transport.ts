import type { Message } from './protocol.js'

export type MessageHandler = (msg: Message) => void

export interface Transport {
  send(msg: Message): void
  onMessage(handler: MessageHandler): void
  close(): void
}

// Two transports wired together in-memory. Useful for tests.
export function createMemoryPair(): [Transport, Transport] {
  const handlersA: MessageHandler[] = []
  const handlersB: MessageHandler[] = []
  let closed = false

  const a: Transport = {
    send(msg) {
      if (closed) return
      queueMicrotask(() => {
        for (const h of handlersB) h(msg)
      })
    },
    onMessage(h) {
      handlersA.push(h)
    },
    close() {
      closed = true
    },
  }

  const b: Transport = {
    send(msg) {
      if (closed) return
      queueMicrotask(() => {
        for (const h of handlersA) h(msg)
      })
    },
    onMessage(h) {
      handlersB.push(h)
    },
    close() {
      closed = true
    },
  }

  return [a, b]
}
