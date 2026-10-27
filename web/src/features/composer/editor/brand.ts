import type { Schemas } from '@/api/client'

/**
 * The brand kit on new content (CO-11): blocks are written in ReputeCast's
 * default style (teal buttons and links, IBM Plex Sans, #1F2937 text), and a
 * saved kit replaces those defaults with its own colours, fonts and button
 * style. Mirrors brandkit.Apply, which does the same to starter templates on
 * the server. Content people wrote is never touched: only defaults change.
 */

type Kit = Pick<Schemas['BrandKit'], 'saved' | 'colors' | 'heading' | 'body' | 'button'>

const plex = `'IBM Plex Sans', Helvetica, Arial, sans-serif`
const georgia = `Georgia, 'Times New Roman', serif`
const radius: Record<string, string> = { square: '0px', rounded: '6px', pill: '999px' }

/** The font-family value emails use: the web font first, then the fallback. */
export function fontStack(f: { name: string; fallback: string }): string {
  const name = f.name.trim()
  return name ? `'${name}', ${f.fallback}` : f.fallback
}

function setAttr(tag: string, name: string, value: string): string {
  const re = new RegExp(`\\s${name}="[^"]*"`)
  return re.test(tag) ? tag.replace(re, ` ${name}="${value}"`) : tag.replace(/^<([a-z-]+)/, `<$1 ${name}="${value}"`)
}

export function applyBrand(src: string, kit: Kit | null | undefined): string {
  if (!kit?.saved) return src
  const c = kit.colors
  let out = src.replace(/<mj-button\b[^>]*>/g, (tag) => {
    let t = tag.replace(/background-color="#0F766E"/gi, `background-color="${c.button}"`)
    t = t.replace(/border-radius="[^"]*"/, `border-radius="${radius[kit.button.shape] ?? '6px'}"`)
    t = t.replace(/font-size="1[34]px"/, `font-size="${kit.button.size}px"`)
    // 44 px tall, easy to tap (BrandKit · Button style).
    return t.replace(/inner-padding="[^"]*"/, 'inner-padding="14px 22px"')
  })
  out = out.replace(/color:\s*#0F766E/gi, `color:${c.link}`)
  out = out.replace(/#1F2937|#0B1220/gi, c.text)
  out = out.replace(/background-color="#F3F4F6"/gi, `background-color="${c.background}"`)
  const body = fontStack(kit.body)
  out = out.split(plex).join(body).split(georgia).join(body)
  // Text blocks holding a heading take the heading font.
  out = out.replace(/<mj-text\b[^>]*>(?=\s*<h[12][\s>])/g, (tag) => setAttr(tag, 'font-family', fontStack(kit.heading)))
  return out
}
