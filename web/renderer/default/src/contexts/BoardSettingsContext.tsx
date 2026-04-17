import { createContext, useContext } from 'react'

interface BoardSettingsContextValue {
  openSettings: () => void
}

export const BoardSettingsContext = createContext<BoardSettingsContextValue>({
  openSettings: () => {},
})

export function useBoardSettingsContext(): BoardSettingsContextValue {
  return useContext(BoardSettingsContext)
}
