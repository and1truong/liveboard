import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { App } from './App.js'
import { ClientProvider } from './queries.js'
import { createClient, queryClient } from './client.js'
import './styles/tailwind.css'

async function boot(): Promise<void> {
  const root = document.getElementById('root')
  if (!root) throw new Error('#root missing')

  const client = createClient()
  try {
    await client.ready()
  } catch (e) {
    root.textContent = `Couldn't connect to shell: ${(e as Error).message}`
    return
  }

  createRoot(root).render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <ClientProvider client={client}>
          <App client={client} />
        </ClientProvider>
      </QueryClientProvider>
    </StrictMode>,
  )
}

void boot()
