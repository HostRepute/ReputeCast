import './styles/app.css'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { TooltipProvider } from './components/ui/Menu'
import { Toaster } from './components/ui/Toast'
import { I18nRoot } from './i18n/root'
import { bootI18n, connectI18n } from './i18n/sync'
import { queryClient } from './lib/query'
import { router } from './router'

// The language and formats apply before the first render, so nothing
// flashes in another language; /me and /meta keep them current after.
connectI18n(queryClient)
await bootI18n()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={300}>
        <I18nRoot>
          <RouterProvider router={router} />
        </I18nRoot>
        <Toaster />
      </TooltipProvider>
    </QueryClientProvider>
  </StrictMode>,
)
