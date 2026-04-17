import { useState, type FormEvent, type ReactNode } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { useAppSettings, useUpdateAppSettings } from '../queries/useAppSettings.js'
import { useExportUrl } from '../queries/useExportUrl.js'
import { useHasCapability } from '../queries/useCapabilities.js'
import { useTheme, type Mode, type ThemeName, THEME_NAMES } from '../contexts/ThemeContext.js'

const UNSUPPORTED_EXPORT_HINT = 'Not available in browser-only mode — connect to a workspace server to enable.'

const THEMES = [
  { value: 'system', label: 'System' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
]

const COLOR_THEMES = [
  { value: 'aqua', label: 'Aqua' },
  { value: 'emerald', label: 'Emerald' },
  { value: 'rose', label: 'Rose' },
]

const FONTS = [
  { value: 'system', label: 'System default' },
  { value: 'inter', label: 'Inter' },
  { value: 'ibm-plex-sans', label: 'IBM Plex Sans' },
  { value: 'source-sans-3', label: 'Source Sans 3' },
  { value: 'nunito-sans', label: 'Nunito Sans' },
  { value: 'dm-sans', label: 'DM Sans' },
  { value: 'rubik', label: 'Rubik' },
]

const CARD_POSITIONS = [
  { value: 'append', label: 'Bottom' },
  { value: 'prepend', label: 'Top' },
]

const CARD_DISPLAY_MODES = [
  { value: 'full', label: 'Show all' },
  { value: 'hide', label: 'Hide' },
  { value: 'trim', label: 'Trim' },
]

const NEWLINE_TRIGGERS = [
  { value: 'shift-enter', label: 'Shift+Enter' },
  { value: 'enter', label: 'Enter' },
]

const WEEK_STARTS = [
  { value: 'monday', label: 'Monday' },
  { value: 'sunday', label: 'Sunday' },
]

export function GlobalSettingsModal({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (next: boolean) => void
}): JSX.Element {
  const settings = useAppSettings()
  const mutation = useUpdateAppSettings()
  const htmlExportSupported = useHasCapability('export:html')
  const mdExportSupported = useHasCapability('export:markdown')
  const htmlExportUrl = useExportUrl('html')
  const mdExportUrl = useExportUrl('markdown')
  const { setMode, setTheme: setColorTheme } = useTheme()

  const triggerExport = (url: string | null): void => {
    if (!url) return
    const top = window.top ?? window
    top.location.href = url
  }

  const submit = (e: FormEvent<HTMLFormElement>): void => {
    e.preventDefault()
    const data = new FormData(e.currentTarget)
    const newTheme = String(data.get('theme') ?? 'system')
    const newColorTheme = String(data.get('color_theme') ?? 'aqua')
    const patch: Parameters<typeof mutation.mutate>[0] = {
      site_name: String(data.get('site_name') ?? '').trim() || 'LiveBoard',
      theme: newTheme,
      color_theme: newColorTheme,
      font_family: String(data.get('font_family') ?? 'system'),
      column_width: parseInt(String(data.get('column_width') ?? '280'), 10),
      show_checkbox: data.get('show_checkbox') === 'on',
      newline_trigger: String(data.get('newline_trigger') ?? 'shift-enter'),
      card_position: String(data.get('card_position') ?? 'append'),
      card_display_mode: String(data.get('card_display_mode') ?? 'full'),
      keyboard_shortcuts: data.get('keyboard_shortcuts') === 'on',
      week_start: String(data.get('week_start') ?? 'monday'),
    }
    mutation.mutate(patch, {
      onSuccess: () => {
        const validModes: Mode[] = ['light', 'dark', 'system']
        if (validModes.includes(newTheme as Mode)) setMode(newTheme as Mode)
        if ((THEME_NAMES as string[]).includes(newColorTheme)) setColorTheme(newColorTheme as ThemeName)
        onOpenChange(false)
      },
    })
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="lb-settings__overlay" />
        <Dialog.Content
          key={String(open)}
          aria-label="Settings"
          aria-describedby={undefined}
          className="lb-settings"
        >
          <header className="lb-settings__header">
            <div className="lb-settings__heading">
              <span className="lb-settings__icon" aria-hidden="true">⚙</span>
              <div className="lb-settings__heading-text">
                <Dialog.Title className="lb-settings__title">Settings</Dialog.Title>
                <p className="lb-settings__subtitle">Workspace preferences</p>
              </div>
            </div>
            <Dialog.Close asChild>
              <button type="button" aria-label="Close" className="lb-settings__close">×</button>
            </Dialog.Close>
          </header>

          <form onSubmit={submit} className="lb-settings__form">
            <div className="lb-settings__scroll">

              <Section label="General">
                <Row label="Site name" hint="Shown in the sidebar and browser tab.">
                  <input
                    name="site_name"
                    type="text"
                    defaultValue={settings.site_name}
                    maxLength={50}
                    className="lb-input"
                    aria-label="site name"
                  />
                </Row>
              </Section>

              <Section label="Appearance">
                <Row label="Theme">
                  <RadioGroup name="theme" options={THEMES} defaultValue={settings.theme} />
                </Row>
                <Row label="Color theme">
                  <RadioGroup name="color_theme" options={COLOR_THEMES} defaultValue={settings.color_theme} />
                </Row>
                <Row label="Font">
                  <SelectField name="font_family" defaultValue={settings.font_family} aria-label="font family">
                    {FONTS.map((f) => (
                      <option key={f.value} value={f.value}>{f.label}</option>
                    ))}
                  </SelectField>
                </Row>
                <Row label="Column width" hint={`${settings.column_width}px`}>
                  <input
                    name="column_width"
                    type="range"
                    min={180}
                    max={600}
                    step={10}
                    defaultValue={settings.column_width}
                    className="lb-range"
                    aria-label="column width"
                  />
                </Row>
              </Section>

              <Section label="Cards &amp; Columns">
                <ToggleRow
                  name="show_checkbox"
                  defaultChecked={settings.show_checkbox}
                  title="Show complete checkbox"
                  hint="Display a checkbox on each card."
                />
                <Row label="New card position">
                  <RadioGroup name="card_position" options={CARD_POSITIONS} defaultValue={settings.card_position} />
                </Row>
                <Row label="Card body display">
                  <RadioGroup name="card_display_mode" options={CARD_DISPLAY_MODES} defaultValue={settings.card_display_mode} />
                </Row>
                <Row label="New line trigger" hint="Key that submits a new card.">
                  <RadioGroup name="newline_trigger" options={NEWLINE_TRIGGERS} defaultValue={settings.newline_trigger} />
                </Row>
                <Row label="Week starts on">
                  <RadioGroup name="week_start" options={WEEK_STARTS} defaultValue={settings.week_start} />
                </Row>
                <ToggleRow
                  name="keyboard_shortcuts"
                  defaultChecked={settings.keyboard_shortcuts}
                  title="Keyboard shortcuts"
                  hint="Enable vim-style navigation keys."
                />
              </Section>

              <Section label="Data">
                <Row
                  label="Export to HTML"
                  hint={htmlExportSupported
                    ? 'Download all boards as a static HTML site (ZIP).'
                    : UNSUPPORTED_EXPORT_HINT}
                >
                  <button
                    type="button"
                    onClick={() => triggerExport(htmlExportUrl)}
                    disabled={!htmlExportSupported || !htmlExportUrl}
                    title={htmlExportSupported ? undefined : UNSUPPORTED_EXPORT_HINT}
                    aria-disabled={!htmlExportSupported || !htmlExportUrl}
                    className="lb-settings__btn lb-settings__btn--ghost"
                  >
                    Export
                  </button>
                </Row>
                <Row
                  label="Export to Markdown"
                  hint={mdExportSupported
                    ? 'Download raw .md source files (ZIP).'
                    : UNSUPPORTED_EXPORT_HINT}
                >
                  <button
                    type="button"
                    onClick={() => triggerExport(mdExportUrl)}
                    disabled={!mdExportSupported || !mdExportUrl}
                    title={mdExportSupported ? undefined : UNSUPPORTED_EXPORT_HINT}
                    aria-disabled={!mdExportSupported || !mdExportUrl}
                    className="lb-settings__btn lb-settings__btn--ghost"
                  >
                    Export
                  </button>
                </Row>
              </Section>

            </div>

            <footer className="lb-settings__footer">
              <button
                type="button"
                onClick={() => onOpenChange(false)}
                className="lb-settings__btn lb-settings__btn--ghost"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={mutation.isPending}
                className="lb-settings__btn lb-settings__btn--primary"
              >
                {mutation.isPending ? 'Saving…' : 'Save'}
              </button>
            </footer>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function Section({
  label,
  children,
}: {
  label: string
  children: ReactNode
}): JSX.Element {
  return (
    <section className="lb-settings__section">
      <div className="lb-settings__section-head">
        <span className="lb-settings__section-label" dangerouslySetInnerHTML={{ __html: label }} />
      </div>
      <div className="lb-settings__group">{children}</div>
    </section>
  )
}

function Row({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: ReactNode
}): JSX.Element {
  return (
    <div className="lb-settings__row">
      <div className="lb-settings__row-text">
        <span className="lb-settings__row-title">{label}</span>
        {hint && <span className="lb-settings__row-hint">{hint}</span>}
      </div>
      <div className="lb-settings__row-control">{children}</div>
    </div>
  )
}

function RadioGroup({
  name,
  options,
  defaultValue,
}: {
  name: string
  options: { value: string; label: string }[]
  defaultValue: string
}): JSX.Element {
  const [value, setValue] = useState(defaultValue)
  return (
    <div role="radiogroup" aria-label={name} className="lb-segmented">
      {options.map((opt) => (
        <button
          key={opt.value}
          type="button"
          role="radio"
          aria-checked={value === opt.value}
          aria-label={`${name} ${opt.value}`}
          onClick={() => setValue(opt.value)}
          className={`lb-segmented__btn${value === opt.value ? ' lb-segmented__btn--active' : ''}`}
          data-field={name}
          data-value={opt.value}
        >
          {opt.label}
        </button>
      ))}
      <input type="hidden" name={name} value={value} />
    </div>
  )
}

function SelectField({ children, ...rest }: React.SelectHTMLAttributes<HTMLSelectElement>): JSX.Element {
  return (
    <span className="lb-select">
      <select className="lb-select__el" {...rest}>
        {children}
      </select>
      <span className="lb-select__chevron" aria-hidden="true">⌄</span>
    </span>
  )
}

function ToggleRow({
  name,
  defaultChecked,
  title,
  hint,
}: {
  name: string
  defaultChecked: boolean
  title: string
  hint?: string
}): JSX.Element {
  return (
    <label className="lb-settings__row lb-settings__row--toggle">
      <div className="lb-settings__row-text">
        <span className="lb-settings__row-title">{title}</span>
        {hint && <span className="lb-settings__row-hint">{hint}</span>}
      </div>
      <span className="lb-toggle">
        <input
          name={name}
          type="checkbox"
          role="switch"
          defaultChecked={defaultChecked}
          className="lb-toggle__input"
        />
        <span className="lb-toggle__track" aria-hidden="true">
          <span className="lb-toggle__thumb" />
        </span>
      </span>
    </label>
  )
}
