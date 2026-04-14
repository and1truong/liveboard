import type { Message } from '../protocol.js'
import type { MessageHandler, Transport } from '../transport.js'

// Transport used by the shell to talk to a specific iframe.
export function shellTransport(iframe: HTMLIFrameElement, expectedOrigin: string): Transport {
  const handlers: MessageHandler[] = []
  const listener = (ev: MessageEvent): void => {
    if (ev.source !== iframe.contentWindow) return
    if (ev.origin !== expectedOrigin) return
    for (const h of handlers) h(ev.data as Message)
  }
  window.addEventListener('message', listener)
  return {
    send(msg) {
      iframe.contentWindow?.postMessage(msg, expectedOrigin)
    },
    onMessage(h) {
      handlers.push(h)
    },
    close() {
      window.removeEventListener('message', listener)
    },
  }
}

// Transport used inside the iframe to talk back to the parent shell.
export function iframeTransport(allowedParentOrigin: string): Transport {
  const handlers: MessageHandler[] = []
  const listener = (ev: MessageEvent): void => {
    if (ev.source !== window.parent) return
    if (ev.origin !== allowedParentOrigin) return
    for (const h of handlers) h(ev.data as Message)
  }
  window.addEventListener('message', listener)
  return {
    send(msg) {
      window.parent.postMessage(msg, allowedParentOrigin)
    },
    onMessage(h) {
      handlers.push(h)
    },
    close() {
      window.removeEventListener('message', listener)
    },
  }
}

// MessagePort-based transport — also usable for hidden-channel scenarios.
export function messagePortTransport(port: MessagePort): Transport {
  const handlers: MessageHandler[] = []
  port.onmessage = (ev: MessageEvent) => {
    for (const h of handlers) h(ev.data as Message)
  }
  port.start()
  return {
    send(msg) {
      port.postMessage(msg)
    },
    onMessage(h) {
      handlers.push(h)
    },
    close() {
      port.close()
    },
  }
}
