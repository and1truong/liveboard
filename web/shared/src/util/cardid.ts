export const ALPHABET =
  'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'

type Generator = () => string

function defaultGenerator(): string {
  const bytes = new Uint32Array(10)
  crypto.getRandomValues(bytes)
  let out = ''
  for (let i = 0; i < 10; i++) out += ALPHABET[bytes[i]! % ALPHABET.length]
  return out
}

let generator: Generator = defaultGenerator

export function newCardId(): string {
  return generator()
}

export function _setGenerator(fn: Generator): void {
  generator = fn
}

export function _resetGenerator(): void {
  generator = defaultGenerator
}
