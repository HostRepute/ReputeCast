import { GitHubMark, LogoMark } from '../components/Marks'
import { links } from '../content/links'
import { togglePaused } from '../lib/motion'
import { githubStars } from '../content/stars'

const nav = [
  { href: '#features', label: 'Features' },
  { href: '#providers', label: 'Providers' },
  { href: '#automations', label: 'Automations' },
  { href: '#compare', label: 'Compare' },
  { href: '#cost', label: 'Cost' },
  { href: links.docs, label: 'Docs' },
]

export function Header() {
  const stars = githubStars()
  return (
    <header className="relative z-[2] border-b border-line-2 bg-white">
      <div className="wrap flex h-auto flex-wrap items-center gap-5 pt-3.5 pb-3.5">
        <a href="/" className="flex items-center gap-2.5 text-ink hover:text-ink" aria-label="ReputeCast home">
          <LogoMark size={30} />
          <span className="text-[17px] font-bold tracking-[-0.01em]">ReputeCast</span>
        </a>
        <nav className="flex flex-wrap gap-4.5 text-[14.5px] font-medium" aria-label="Site">
          {nav.map((l) => (
            <a key={l.href} href={l.href} className="text-ink-nav hover:text-ink-nav">
              {l.label}
            </a>
          ))}
        </nav>
        <div className="ml-auto flex flex-wrap gap-2.5">
          {/* Pauses every animation on the page (WCAG 2.2.2). The label follows
              <html data-motion>, which is restored before first paint. */}
          <button type="button" className="btn motion-toggle min-h-10" onClick={togglePaused}>
            <span className="when-playing">Pause motion</span>
            <span className="when-paused">Play motion</span>
          </button>
          <a href={links.discord} className="btn min-h-10">
            Discord
          </a>
          <a href={links.repo} className="btn dark min-h-10">
            <GitHubMark />
            {stars ? (
              <span>
                GitHub <span aria-hidden="true">·</span> {stars}
                <span className="vh"> stars</span>
              </span>
            ) : (
              'GitHub'
            )}
          </a>
        </div>
      </div>
    </header>
  )
}
