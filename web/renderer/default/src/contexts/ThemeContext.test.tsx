import { afterEach, beforeEach, describe, expect, it } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import { ThemeProvider, useTheme } from './ThemeContext.js'

if (typeof window.matchMedia === 'undefined') {
  window.matchMedia = () => ({ matches: false, addEventListener() {}, removeEventListener() {} } as any)
}

function cleanDocClasses(): void {
  const el = document.documentElement
  el.className = ''
}

describe('ThemeContext', () => {
  beforeEach(() => {
    localStorage.clear()
    cleanDocClasses()
  })
  afterEach(() => {
    localStorage.clear()
    cleanDocClasses()
  })

  it('defaults to system mode and indigo theme', () => {
    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider })
    expect(result.current.mode).toBe('system')
    expect(result.current.theme).toBe('indigo')
  })

  it('setMode(dark) adds dark class and persists', () => {
    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider })
    act(() => result.current.setMode('dark'))
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('liveboard:mode')).toBe('dark')
  })

  it('setMode(light) removes dark class', () => {
    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider })
    act(() => result.current.setMode('dark'))
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    act(() => result.current.setMode('light'))
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem('liveboard:mode')).toBe('light')
  })

  it('setTheme swaps theme-* classes and persists', () => {
    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider })
    act(() => result.current.setTheme('emerald'))
    expect(document.documentElement.classList.contains('theme-emerald')).toBe(true)
    expect(document.documentElement.classList.contains('theme-indigo')).toBe(false)
    expect(localStorage.getItem('liveboard:theme')).toBe('emerald')
    act(() => result.current.setTheme('rose'))
    expect(document.documentElement.classList.contains('theme-rose')).toBe(true)
    expect(document.documentElement.classList.contains('theme-emerald')).toBe(false)
  })

  it('reads seeded localStorage values on mount', () => {
    localStorage.setItem('liveboard:mode', 'dark')
    localStorage.setItem('liveboard:theme', 'sunset')
    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider })
    expect(result.current.mode).toBe('dark')
    expect(result.current.theme).toBe('sunset')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(document.documentElement.classList.contains('theme-sunset')).toBe(true)
  })
})
