/**
 * CSS injected into the page builder's canvas, after the site's own
 * stylesheets (site.css, pages.css, theme.css). It only styles editor
 * affordances: the page itself is drawn by the public pages' classes, so
 * it looks the way readers see it. The selection, the hover outline and
 * the block toolbar are drawn by the builder over the canvas.
 */
export const canvasCss = `
html, body { margin: 0; background: #fff; }
body { -webkit-font-smoothing: antialiased; }
/* The frame grows with the page (auto height): nothing may size itself on the viewport. */
.st { min-height: 0 !important; }
[data-gjs-type="wrapper"] { min-height: 160px; }
/* Live blocks are custom elements the server draws; they're blocks. */
rc-form, rc-countdown, rc-event, rc-product, rc-products, rc-store-product, rc-tiers, rc-posts, rc-video { display: block; }
.lp-cols > rc-form, .lp-cols > rc-product { min-width: 0; }
.rc-live { pointer-events: none; }
.rc-live .pg-empty { margin: 0; }
/* The site's header and footer: the theme's, drawn as readers see them, not edited here. */
.rc-frame { pointer-events: none; user-select: none; }
.gjs-selected, .gjs-selected-parent { outline: none !important; }
[contenteditable="true"] { outline: none; cursor: text; }
.gjs-freezed { opacity: .5; pointer-events: none; }
.gjs-no-pointer { pointer-events: none; }
.gjs-grabbing, .gjs-grabbing * { cursor: grabbing !important; }
.gjs-plh-image { background: #E9ECF0; border: 1px dashed #C9CED6; outline: none; min-height: 120px; }
/* Summaries open and close as on the site; the click also selects them. */
.faq summary { cursor: pointer; }
`
