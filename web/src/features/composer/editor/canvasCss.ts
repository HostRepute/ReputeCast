/**
 * CSS injected into the canvas iframe. It only styles editor affordances
 * (selection, columns of the selected section, text being edited, empty
 * image placeholders); the email itself is the compiled MJML.
 */
export const canvasCss = `
html, body { margin: 0; }
body { -webkit-font-smoothing: antialiased; }
/* The frame grows with the email (auto height), so nothing may size itself on the viewport. */
[data-gjs-type="wrapper"], [data-gjs-type="mjml"], [data-gjs-type="mj-body"] { min-height: 0 !important; }
.gjs-selected { outline: 2px solid #1D4ED8 !important; outline-offset: -2px; }
.gjs-selected-parent { outline: none !important; }
.gjs-selected [data-gjs-type="mj-column"] { outline: 1px dashed #93C5FD; outline-offset: 2px; }
.gjs-selected[data-gjs-type="mj-column"] { outline-offset: 2px; }
[data-gjs-type="mj-text"] [contenteditable="true"] { outline: none; cursor: text; }
.gjs-plh-image { background: #E9ECF0; border: 1px dashed #C9CED6; outline: none; height: 120px; width: 100%; }
.gjs-freezed { opacity: .5; pointer-events: none; }
.gjs-no-pointer { pointer-events: none; }
.gjs-grabbing, .gjs-grabbing * { cursor: grabbing !important; }
[data-rc-warning] { outline: 2px solid currentColor; }
/* Personalization tokens as chips (CO-4): "First name · fallback “there”"; they export as their Liquid. */
.rc-lq { display: inline-flex; align-items: center; gap: 4px; height: 22px; padding: 0 8px; margin: 0 1px; border-radius: 999px; border: 1px solid transparent; background: #E3F1EE; color: #0F766E; font: 600 13px/1 'IBM Plex Sans', system-ui, sans-serif; vertical-align: 1px; cursor: pointer; white-space: nowrap; text-decoration: none; }
.rc-lq .fb { font-weight: 500; color: #115E59; }
.rc-lq:hover, .rc-lq:focus-visible { border-color: #0F766E; outline: none; }
.rc-lq.unknown { background: #FEF3C7; color: #92400E; }
.rc-lq.unknown .fb { color: #92400E; }
/* MJML's own rule for fluid-on-mobile images, which the compiled email carries in its head. */
@media only screen and (max-width: 479px) {
  table.mj-full-width-mobile { width: 100% !important; }
  td.mj-full-width-mobile { width: auto !important; }
}
/* Hidden on this device: dimmed while editing (the preview shows the email as sent). */
@media (max-width: 480px) { html body .rc-hide-mobile { display: revert !important; opacity: .35; } }
@media (min-width: 481px) { html body .rc-hide-desktop { display: revert !important; opacity: .35; } }
/* Dark-mode check: inverted like a client that ignores the email's colours; images keep theirs. */
html.rc-dark { filter: invert(1) hue-rotate(180deg); background: #fff; }
html.rc-dark img { filter: invert(1) hue-rotate(180deg); }
`
