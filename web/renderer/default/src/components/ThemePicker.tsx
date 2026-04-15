import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { THEME_NAMES, useTheme, type Mode, type ThemeName } from '../contexts/ThemeContext.js'

const SWATCH_COLOR: Record<ThemeName, string> = {
  indigo: '#6366f1',
  github: '#2da44e',
  gitlab: '#fc6d26',
  emerald: '#10b981',
  rose: '#f43f5e',
  sunset: '#f97316',
}

export function ThemePicker(): JSX.Element {
  const { mode, theme, setMode, setTheme } = useTheme()
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger
        aria-label="Theme picker"
        className="rounded p-1 text-slate-500 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
      >
        🎨
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          sideOffset={4}
          className="z-50 min-w-48 rounded-md bg-white p-2 shadow-lg ring-1 ring-slate-200 dark:bg-slate-800 dark:ring-slate-700"
        >
          <DropdownMenu.Label className="px-2 py-1 text-xs uppercase tracking-wide text-slate-400">
            Mode
          </DropdownMenu.Label>
          <DropdownMenu.RadioGroup value={mode} onValueChange={(v) => setMode(v as Mode)}>
            {(['light', 'dark', 'system'] as Mode[]).map((m) => (
              <DropdownMenu.RadioItem
                key={m}
                value={m}
                className="cursor-pointer rounded px-2 py-1 text-sm text-slate-800 outline-none aria-checked:bg-slate-100 dark:text-slate-100 dark:aria-checked:bg-slate-700"
              >
                {m.charAt(0).toUpperCase() + m.slice(1)}
              </DropdownMenu.RadioItem>
            ))}
          </DropdownMenu.RadioGroup>
          <DropdownMenu.Separator className="my-1 h-px bg-slate-200 dark:bg-slate-700" />
          <DropdownMenu.Label className="px-2 py-1 text-xs uppercase tracking-wide text-slate-400">
            Theme
          </DropdownMenu.Label>
          <div className="flex gap-1 p-1">
            {THEME_NAMES.map((t) => (
              <button
                key={t}
                type="button"
                aria-label={`theme ${t}`}
                aria-pressed={theme === t}
                onClick={() => setTheme(t)}
                style={{ backgroundColor: SWATCH_COLOR[t] }}
                className={`h-6 w-6 rounded-full ${theme === t ? 'ring-2 ring-offset-2 ring-slate-400 dark:ring-offset-slate-800' : ''}`}
              />
            ))}
          </div>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
