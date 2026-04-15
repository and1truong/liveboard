import { beforeEach, describe, expect, it } from 'bun:test'
import { render } from '@testing-library/react'
import { ThemeProvider } from '../contexts/ThemeContext.js'
import { ThemePicker } from './ThemePicker.js'

describe('ThemePicker', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.className = ''
  })

  it('renders the trigger', () => {
    const { getByLabelText } = render(
      <ThemeProvider>
        <ThemePicker />
      </ThemeProvider>,
    )
    expect(getByLabelText('Theme picker')).toBeDefined()
  })
})
