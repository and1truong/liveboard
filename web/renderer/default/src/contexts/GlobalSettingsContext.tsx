import { createContext, useContext } from 'react'

interface GlobalSettingsContextValue {
  openSettings: () => void
}

export const GlobalSettingsContext = createContext<GlobalSettingsContextValue>({
  openSettings: () => {},
})

export function useGlobalSettingsContext(): GlobalSettingsContextValue {
  return useContext(GlobalSettingsContext)
}
