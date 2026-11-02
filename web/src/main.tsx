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
import { installPreloadRecovery } from './lib/systemStatus'
import { preloadScreen, router } from './router'

// The language and formats apply before the first render, so nothing
// flashes in another language; /me and /meta keep them current after.
connectI18n(queryClient)
const booted = await bootI18n()

// A member's screen starts loading its code while its routes check the
// session (/me), not once they have: on a slow phone the two took turns.
if (booted === 'member') preloadScreen(location.pathname)

// A tab from before an upgrade reloads when an old chunk is gone (FE-2).
installPreloadRecovery()

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
