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
`
