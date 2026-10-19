import { useEffect, useLayoutEffect, useRef, useState, type CSSProperties } from 'react'

/**
 * A page of the site rendered by the server, shown inert (no scripts; the
 * page's own stylesheets load from the instance, as with the other public
 * page previews) at the height of its content. With `scale`, the page is
 * laid out `width` wide and drawn smaller.
 */
export function SiteFrame({
  html,
  title,
  width,
  scale: fixedScale = 1,
  minHeight = 400,
  maxHeight = 6000,
  className,
  style,
  fit,
}: {
  html: string
  title: string
  /** The width the page is laid out at (default: the frame's own width). */
  width?: number
  scale?: number
  minHeight?: number
  maxHeight?: number
  className?: string
  style?: CSSProperties
  /** Scale the page to the width of the frame's container (with `width`). */
  fit?: boolean
}) {
  const frame = useRef<HTMLIFrameElement>(null)
  const box = useRef<HTMLDivElement>(null)
  const [fitted, setFitted] = useState<number | null>(null)
  useLayoutEffect(() => {
    const el = box.current
    if (!fit || !width || !el) return
    const measure = () => setFitted(Math.min(1, el.clientWidth / width))
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    measure()
    return () => ro.disconnect()
  }, [fit, width])
  const observer = useRef<ResizeObserver | null>(null)
  const [height, setHeight] = useState(minHeight)
  useEffect(() => () => observer.current?.disconnect(), [])
  const scale = fit ? (fitted ?? 0.5) : fixedScale
  // A page drawn smaller is a picture of the page: it doesn't scroll or take clicks.
  const thumb = fit || scale !== 1
  const onLoad = () => {
    const doc = frame.current?.contentDocument
    if (!doc?.body) return
    if (thumb) doc.documentElement.style.overflow = 'hidden'
    observer.current?.disconnect()
    const measure = () => setHeight(Math.min(maxHeight, Math.max(doc.documentElement.scrollHeight, minHeight)))
    measure()
    observer.current = new ResizeObserver(measure)
    observer.current.observe(doc.body)
  }
  const frameStyle: CSSProperties = {
    width: scale === 1 ? (width ?? '100%') : width,
    height,
    border: 0,
    display: 'block',
    background: '#fff',
    ...(scale === 1 ? {} : { transform: `scale(${scale})`, transformOrigin: '0 0' }),
    ...(thumb ? { pointerEvents: 'none' } : {}),
    ...style,
  }
  const iframe = <iframe ref={frame} title={title} srcDoc={html} sandbox="allow-same-origin" tabIndex={thumb ? -1 : undefined} onLoad={onLoad} className={scale === 1 && !fit ? className : undefined} style={frameStyle} />
  if (scale === 1 && !fit) return iframe
  return (
    <div ref={box} className={className} style={{ height: height * scale, overflow: 'hidden', position: 'relative' }}>
      {iframe}
    </div>
  )
}
