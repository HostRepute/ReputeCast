import type { ReactNode } from 'react'
import { useT, type MessageDescriptor } from '@/i18n'
import { m } from './composer.messages'
import { views, withEllipsis, type PlatformKey, type PlatformView } from './cuts'
import './composer.css'

/** What a notification shows: its text, the site, its icon and image, and its buttons. */
export interface PushPreview {
  title: string
  message: string
  host: string
  iconUrl?: string
  imageUrl?: string
  buttons: string[]
}

export const platformNames: Record<PlatformKey, { name: MessageDescriptor; sub: MessageDescriptor }> = {
  chrome_windows: { name: m.chromeWindows, sub: m.chromeWindowsSub },
  chrome_mac: { name: m.chromeMac, sub: m.chromeMacSub },
  safari_mac: { name: m.safariMac, sub: m.safariMacSub },
  chrome_android: { name: m.chromeAndroid, sub: m.chromeAndroidSub },
  safari_ios: { name: m.safariIos, sub: m.safariIosSub },
}

/** The site's icon: its image, or its initial on ink. */
export function Mark({ n, size, radius, font }: { n: PushPreview; size: number; radius: number; font: number }) {
  const initial = (n.host.replace(/^www\./, '')[0] ?? 'A').toUpperCase()
  return (
    <span className="mk" style={{ width: size, height: size, borderRadius: radius, fontSize: font }} aria-hidden="true">
      {n.iconUrl ? <img src={n.iconUrl} alt="" /> : initial}
    </span>
  )
}

function ChromeLogo({ size }: { size: number }) {
  return (
    <span className="mk" style={{ width: size, height: size, borderRadius: '50%', background: '#4B5563' }} aria-hidden="true">
      <svg width={size * 0.6} height={size * 0.6} viewBox="0 0 16 16" fill="none" stroke="#fff" strokeWidth="1.6">
        <circle cx="8" cy="8" r="6" />
        <circle cx="8" cy="8" r="2.4" />
        <path d="M8 5.6h5.6M5.9 9.2 3.1 4.4M10.1 9.2 7.3 14" />
      </svg>
    </span>
  )
}

/** One platform's preview with its caption: where it cuts the text. */
export function Preview({ v, n }: { v: PlatformView; n: PushPreview }) {
  const t = useT()
  const p = v.platform
  const title = withEllipsis(v.title.shown, v.title.hidden) || t(m.untitled)
  const message = withEllipsis(v.message.shown, v.message.hidden) || t(m.noMessage)
  const buttons = n.buttons.filter(Boolean)
  const image = Boolean(n.imageUrl)
  const msgCut = v.message.hidden ? 'cut' : 'full'
  let caption: string
  if (v.title.hidden) caption = t(m.capTitleCut, { shown: withEllipsis(v.title.shown, v.title.hidden) })
  else
    switch (p.key) {
      case 'chrome_windows':
        caption = t(m.capWindows, {
          msg: msgCut,
          n: [...n.title].length,
          max: p.titleChars,
          lines: p.messageLines,
          hidden: v.message.hidden,
          image: image ? 'yes' : 'no',
          buttons: buttons.length,
        })
        break
      case 'chrome_mac':
        caption = t(m.capChromeMac, { msg: msgCut, lines: p.messageLines, hidden: v.message.hidden, buttons: buttons.length })
        break
      case 'safari_mac':
        caption = t(m.capSafariMac, { msg: msgCut, lines: p.messageLines, hidden: v.message.hidden })
        break
      case 'chrome_android':
        caption = t(m.capAndroid, { image: image ? 'yes' : 'no', buttons: buttons.length })
        break
      default:
        caption = t(m.capIos, { msg: msgCut, lines: p.messageLines, hidden: v.message.hidden })
    }
  let body: ReactNode
  switch (p.key) {
    case 'chrome_windows':
      body = (
        <div className="px-pv px-pvwin">
          <div className="h">
            <ChromeLogo size={16} />
            <span>{t(m.googleChrome)}</span>
            <span className="x" aria-hidden="true">
              <span>···</span>
              <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
                <path d="M4 4l8 8M12 4l-8 8" />
              </svg>
            </span>
          </div>
          {image ? (
            <div className="px-3">
              <img className="img" src={n.imageUrl} alt="" />
            </div>
          ) : null}
          <div className="b">
            <Mark n={n} size={32} radius={7} font={14} />
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <b className="overflow-hidden whitespace-nowrap">{title}</b>
              <span className="m">{message}</span>
              <span className="s text-[12px]">{n.host}</span>
            </div>
          </div>
          {buttons.length ? (
            <div className="bt" style={buttons.length === 1 ? { gridTemplateColumns: '1fr' } : undefined}>
              {buttons.slice(0, 2).map((b, i) => (
                <span key={i}>{b}</span>
              ))}
            </div>
          ) : (
            <div className="h-3" />
          )}
        </div>
      )
      break
    case 'chrome_mac':
    case 'safari_mac':
      body = (
        <div className="px-pv px-pvmac">
          {p.key === 'chrome_mac' ? <ChromeLogo size={32} /> : <Mark n={n} size={32} radius={7} font={14} />}
          <div className="bd">
            <div className="r1">
              <b className="min-w-0 overflow-hidden whitespace-nowrap">{title}</b>
              <span className="tm">{t(m.now)}</span>
            </div>
            <span className="s">{n.host}</span>
            <span className="m">{message}</span>
          </div>
          {p.key === 'chrome_mac' ? (
            <span className="self-center">
              <Mark n={n} size={30} radius={6} font={13} />
            </span>
          ) : null}
        </div>
      )
      break
    case 'chrome_android':
      body = (
        <div className="px-pv px-pvand">
          <div className="h">
            <Mark n={n} size={16} radius={4} font={9} />
            <span className="min-w-0 truncate">{t(m.androidHead, { host: n.host })}</span>
            <span className="ml-auto" aria-hidden="true">
              <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <path d="M4 10l4-4 4 4" />
              </svg>
            </span>
          </div>
          <div className="bd">
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <b className="overflow-hidden whitespace-nowrap">{title}</b>
              <span className="m one">{n.message || t(m.noMessage)}</span>
            </div>
            <Mark n={n} size={36} radius={18} font={15} />
          </div>
          {image ? (
            <div className="im">
              <img className="img" src={n.imageUrl} alt="" style={{ borderRadius: 0 }} />
            </div>
          ) : null}
          {buttons.length ? (
            <div className="ac">
              {buttons.slice(0, 2).map((b, i) => (
                <span key={i}>{b}</span>
              ))}
            </div>
          ) : (
            <div className="h-2" />
          )}
        </div>
      )
      break
    default:
      body = (
        <div className="px-pv px-pvios">
          <Mark n={n} size={38} radius={9} font={16} />
          <div className="bd" style={{ gap: 2 }}>
            <div className="r1">
              <b className="min-w-0">{title}</b>
              <span className="tm">{t(m.now)}</span>
            </div>
            <span className="m">{message}</span>
          </div>
        </div>
      )
  }
  return (
    <figure className="px-pvf">
      <figcaption>
        <b>{t(platformNames[p.key].name)}</b>
        <span className="k">{t(platformNames[p.key].sub)}</span>
      </figcaption>
      <div className="px-pvstage">{body}</div>
      <span className="cut">{caption}</span>
    </figure>
  )
}

/** "At a glance": each platform's message, image and buttons in one table. */
export function Glance({ n, maxMessage }: { n: PushPreview; maxMessage: number }) {
  const t = useT()
  const vs = views(n.title, n.message)
  const cutOn = vs.filter((v) => v.title.hidden).map((v) => t(platformNames[v.platform.key].name))
  const buttons = n.buttons.filter(Boolean).length
  const image = Boolean(n.imageUrl)
  return (
    <section className="px-pvf" aria-label={t(m.glance)}>
      <div className="flex flex-col gap-0.5">
        <span className="text-[13px] font-bold">{t(m.glance)}</span>
        <span className="k">
          {cutOn.length
            ? t(m.glanceCut, { n: [...n.title].length, platforms: cutOn.join(', '), m: [...n.message].length, max: maxMessage })
            : t(m.glanceAll, { n: [...n.title].length, m: [...n.message].length, max: maxMessage })}
        </span>
      </div>
      <div className="w-full overflow-x-auto">
        <table className="px-gl">
          <thead>
            <tr>
              <th scope="col">{t(m.colPlatform)}</th>
              <th scope="col">{t(m.colMessage)}</th>
              <th scope="col">{t(m.colImage)}</th>
              <th scope="col">{t(m.colButtons)}</th>
            </tr>
          </thead>
          <tbody>
            {vs.map((v) => {
              const p = v.platform
              const msg = p.key === 'chrome_android' ? t(m.line1) : !v.message.hidden && p.key === 'chrome_windows' ? t(m.inFull) : t(m.lines, { n: p.messageLines })
              const img = !image || !p.image ? t(m.none) : p.key === 'chrome_android' ? t(m.expanded) : t(m.shown)
              const btn = !buttons || !p.buttons ? t(m.none) : p.key === 'chrome_mac' ? t(m.onHover) : String(Math.min(2, buttons))
              return (
                <tr key={p.key}>
                  <th scope="row">{t(platformNames[p.key].name)}</th>
                  <td>{msg}</td>
                  <td>{img}</td>
                  <td>{btn}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </section>
  )
}

/** The previews of all platforms, or the desktop or phone ones. */
export function PreviewColumns({ n, show, maxMessage }: { n: PushPreview; show: 'all' | 'desktop' | 'phones'; maxMessage: number }) {
  const t = useT()
  const vs = views(n.title, n.message)
  const desktop = vs.filter((v) => v.platform.desktop)
  const phones = vs.filter((v) => !v.platform.desktop)
  return (
    <div className="pc-pcols">
      {show !== 'phones' ? (
        <div className="pc-pcol">
          <span className="k8">{t(m.desktop)}</span>
          {desktop.map((v) => (
            <Preview key={v.platform.key} v={v} n={n} />
          ))}
        </div>
      ) : null}
      {show !== 'desktop' ? (
        <div className="pc-pcol">
          <span className="k8">{t(m.phones)}</span>
          {phones.map((v) => (
            <Preview key={v.platform.key} v={v} n={n} />
          ))}
          <Glance n={n} maxMessage={maxMessage} />
        </div>
      ) : (
        <div className="pc-pcol">
          <Glance n={n} maxMessage={maxMessage} />
        </div>
      )}
    </div>
  )
}

