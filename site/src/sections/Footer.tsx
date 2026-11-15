import { Icon } from '../components/Icon'
import { LogoMark } from '../components/Marks'
import { links } from '../content/links'
import { togglePaused } from '../lib/motion'

const footerLinks = [
  { href: links.repo, label: 'GitHub' },
  { href: links.discussions, label: 'Discussions' },
  { href: links.docs, label: 'Docs' },
  { href: links.adapterSdk, label: 'Adapter SDK' },
  { href: links.hostrepute, label: 'HostRepute' },
]

export function Footer() {
  return (
    <footer className="border-t border-line-2 bg-ground-2">
      <div className="wrap flex flex-wrap items-center gap-6 pt-7 pb-7 text-[14px] text-ink-3">
        <span className="flex items-center gap-2 font-bold text-ink">
          <LogoMark size={22} />
          ReputeCast
        </span>
        {footerLinks.map((l) => (
          <a key={l.label} href={l.href} className="text-ink-3 hover:text-ink-3">
            {l.label}
          </a>
        ))}
        {/* Pauses every animation on the page (WCAG 2.2.2). The label follows
            <html data-motion>, which is restored before first paint. */}
        <button type="button" className="ft-motion motion-toggle" onClick={togglePaused}>
          <span className="when-playing">
            <Icon name="pause" size={14} />
            Pause motion
          </span>
          <span className="when-paused">
            <Icon name="play" size={14} />
            Play motion
          </span>
        </button>
        <span className="ml-auto">Open-source licence: see the repository · Built by HostRepute</span>
      </div>
    </footer>
  )
}
