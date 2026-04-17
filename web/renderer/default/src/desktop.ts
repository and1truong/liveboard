// Desktop (Wails) integration for the renderer.
//
// The shell adds `desktop-app` to its own `<html>` and installs a drag
// handler on its own window. The renderer runs inside a same-origin iframe,
// so neither the class nor the mousedown events cross the boundary. This
// module mirrors both into the iframe context.

interface WailsBridge {
  webkit?: { messageHandlers?: { external?: { postMessage: (m: string) => void } } }
  WailsInvoke?: (m: string) => void
}

export function initDesktopMode(): void {
  if (window === window.parent) return
  let parentDoc: Document
  try {
    parentDoc = window.parent.document
  } catch {
    return
  }
  const html = document.documentElement

  const sync = (): void => {
    html.classList.toggle('desktop-app', parentDoc.documentElement.classList.contains('desktop-app'))
  }
  sync()
  new MutationObserver(sync).observe(parentDoc.documentElement, {
    attributes: true,
    attributeFilter: ['class'],
  })

  const parentWin = window.parent as unknown as WailsBridge
  const invoke = parentWin.webkit?.messageHandlers?.external
    ? (m: string): void => parentWin.webkit!.messageHandlers!.external!.postMessage(m)
    : (parentWin.WailsInvoke ?? ((): void => undefined))

  let shouldDrag = false
  window.addEventListener('mousedown', (e) => {
    if (!(e.target instanceof Element)) return
    const v = window.getComputedStyle(e.target).getPropertyValue('--wails-draggable')
    shouldDrag = v.trim() === 'drag' && e.buttons === 1 && e.detail === 1
  })
  window.addEventListener('mousemove', (e) => {
    if (!shouldDrag) return
    shouldDrag = false
    if (e.buttons > 0) invoke('drag')
  })
  window.addEventListener('mouseup', () => {
    shouldDrag = false
  })
}
