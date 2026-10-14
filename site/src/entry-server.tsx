import { StrictMode } from 'react'
import { renderToString } from 'react-dom/server'
import { App } from './App'

/** Renders the page to HTML at build time (scripts/build.ts). */
export function render(): string {
  return renderToString(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}
