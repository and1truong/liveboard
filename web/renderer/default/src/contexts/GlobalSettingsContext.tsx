import { createContext, useContext } from 'react'

export type GlobalSettingsSection = 'data'

interface GlobalSettingsContextValue {
  openSettings: (section?: GlobalSettingsSection) => void
}

export const GlobalSettingsContext = createContext<GlobalSettingsContextValue>({
  openSettings: () => {},
})

export function useGlobalSettingsContext(): GlobalSettingsContextValue {
  return useContext(GlobalSettingsContext)
}
