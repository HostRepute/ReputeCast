// First: zod's settings, before the chunk zod is in runs (src/boot/zod.ts).
import './boot/zod'
import './styles/app.css'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { TooltipProvider } from './components/ui/Menu'
import { Toaster } from './components/ui/Toast'
import { connectTheme } from './features/appearance/useTheme'
import { I18nRoot } from './i18n/root'
import { applyMember, bootI18n, connectI18n, isVisitorPath } from './i18n/sync'
import { queryClient } from './lib/query'
import { fetchMe, meQueryKey } from './lib/session'
import { installPreloadRecovery } from './lib/systemStatus'
import { startTheme } from './lib/theme'
import { preloadScreen, router } from './router'

// The theme (already set by src/early.ts) follows the device and other
// tabs from here, and the member's account once /me comes (M54).
startTheme()
connectTheme(queryClient)

// The language and formats apply before the first render, so nothing
// flashes in another language; /me and /meta keep them current after.
connectI18n(queryClient)
const booted = await bootI18n()

// A member's screen starts loading its code while its routes check the
// session (/me), not once they have: on a slow phone the two took turns.
if (booted === 'member') preloadScreen(location.pathname)
// A member's screen in a browser that hasn't kept their language (a new
// device, cleared site data) waits for /me and their language first: drawn
// in the visitor's formats, it was drawn again from the top once /me came
// (I18nRoot), and everything on it moved. The ReputeCast mark stays up
// meanwhile, and the routes ask for /me anyway (the same query).
else if (!isVisitorPath(location.pathname)) {
  const me = await queryClient.fetchQuery({ queryKey: meQueryKey, queryFn: fetchMe }).catch(() => null)
  if (me) await applyMember(me)
}

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
