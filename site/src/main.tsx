import './styles/site.css'
import { StrictMode } from 'react'
import { createRoot, hydrateRoot } from 'react-dom/client'
import { App } from './App'

const root = document.getElementById('root')!
const app = (
  <StrictMode>
    <App />
  </StrictMode>
)

// The production page is prerendered (scripts/build.ts) and hydrated;
// the dev server serves the empty shell and renders on the client.
if (root.firstElementChild) hydrateRoot(root, app)
else createRoot(root).render(app)
