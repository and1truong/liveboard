import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { THEME_NAMES, useTheme, type Mode, type ThemeName } from '../contexts/ThemeContext.js'

const SWATCH_COLOR: Record<ThemeName, string> = {
  indigo: '#6366f1',
  github: '#2da44e',
  gitlab: '#fc6d26',
  emerald: '#10b981',
  rose: '#f43f5e',
  sunset: '#f97316',
  aqua: '#007AFF',
}

export function ThemePicker(): JSX.Element {
  const { mode, theme, setMode, setTheme } = useTheme()
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger aria-label="Theme picker" className="lb-iconbtn">
        <span aria-hidden style={{ fontSize: 14, lineHeight: 1 }}>🎨</span>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content sideOffset={6} className="lb-popover" style={{ minWidth: 200 }}>
          <DropdownMenu.Label className="lb-popover__label">Mode</DropdownMenu.Label>
          <DropdownMenu.RadioGroup value={mode} onValueChange={(v) => setMode(v as Mode)}>
            {(['light', 'dark', 'system'] as Mode[]).map((m) => (
              <DropdownMenu.RadioItem
                key={m}
                value={m}
                className="lb-popover__item"
              >
                <span className="lb-popover__icon" aria-hidden>
                  {m === 'light' ? '☀' : m === 'dark' ? '☾' : '◐'}
                </span>
                <span>{m.charAt(0).toUpperCase() + m.slice(1)}</span>
              </DropdownMenu.RadioItem>
            ))}
          </DropdownMenu.RadioGroup>
          <hr className="lb-popover__sep" />
          <DropdownMenu.Label className="lb-popover__label">Theme</DropdownMenu.Label>
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
