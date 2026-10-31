/**
 * HTML the composer reads but didn't write (a paste, a text's content, the
 * canvas's live markup) is parsed in an inert document, never in the page's
 * own: there nothing in it loads or runs while it's read. Parsed into the
 * page's document, even a detached element fetches an <img src> and fires
 * its onerror.
 */
export function parseInert(html: string): HTMLElement {
  return new DOMParser().parseFromString(html, 'text/html').body
}

const ACTIVE = 'script, iframe, frame, frameset, object, embed, base, link, meta'
const URL_ATTR = /^(href|src|xlink:href|action|formaction|background|poster)$/i
const RUNS = /^\s*(javascript|vbscript):|^\s*data:text\/html/i

/**
 * Removes what could run once the HTML is placed in the app (a text edited
 * in Write mode): script-like elements, on* handlers, srcdoc and
 * javascript: URLs. Email HTML never needs them, and mail clients drop them.
 */
export function stripActive(root: Element): void {
  root.querySelectorAll(ACTIVE).forEach((el) => el.remove())
  for (const el of Array.from(root.querySelectorAll('*'))) {
    for (const a of Array.from(el.attributes)) {
      if (/^on/i.test(a.name) || a.name.toLowerCase() === 'srcdoc' || (URL_ATTR.test(a.name) && RUNS.test(a.value))) el.removeAttribute(a.name)
    }
  }
}
