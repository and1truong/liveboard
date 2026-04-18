import { useState } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { THEME_NAMES, useTheme, type Mode, type ThemeName } from '../contexts/ThemeContext.js'

const SWATCH_COLOR: Record<ThemeName, string> = {
  aqua:   '#007AFF',
  green:  '#34C759',
  red:    '#FF3B30',
  orange: '#FF9500',
  purple: '#AF52DE',
  pink:   '#FF375F',
  black:  '#1C1C1E',
}

const MODE_META: Record<Mode, { icon: string; label: string }> = {
  light:  { icon: '☀', label: 'Light' },
  dark:   { icon: '☾', label: 'Dark' },
  system: { icon: '◐', label: 'Auto' },
}

export function ThemePicker(): JSX.Element {
  const { mode, theme, setMode, setTheme } = useTheme()
  const [open, setOpen] = useState(false)
  return (
    <DropdownMenu.Root open={open} onOpenChange={setOpen}>
      <DropdownMenu.Trigger aria-label="Theme picker" className="lb-iconbtn">
        <span aria-hidden style={{ fontSize: 14, lineHeight: 1 }}>🎨</span>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content sideOffset={6} className="lb-popover" style={{ minWidth: 200 }}>
          <DropdownMenu.Label className="lb-popover__label">Appearance</DropdownMenu.Label>
          <div className="lb-mode-seg">
            {(['light', 'dark', 'system'] as Mode[]).map((m) => (
              <button
                key={m}
                type="button"
                aria-pressed={mode === m}
                onClick={() => setMode(m)}
                className="lb-mode-seg__btn"
              >
                <span aria-hidden>{MODE_META[m].icon}</span>
                <span>{MODE_META[m].label}</span>
              </button>
            ))}
          </div>
          <hr className="lb-popover__sep" />
          <DropdownMenu.Label className="lb-popover__label">Accent</DropdownMenu.Label>
          <div className="lb-swatches">
            {THEME_NAMES.map((t) => (
              <button
                key={t}
                type="button"
                aria-label={`theme ${t}`}
                aria-pressed={theme === t}
                onClick={() => setTheme(t)}
                style={{ backgroundColor: SWATCH_COLOR[t], color: SWATCH_COLOR[t] }}
                className="lb-swatches__dot"
              />
            ))}
          </div>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
