/**
 * The star count on the header's GitHub button ("GitHub · 1.2k"). The build
 * script fetches it from the GitHub API once (scripts/build.ts) and passes it
 * in as VITE_GITHUB_STARS, so visitors' browsers never call GitHub. Without
 * a count the button just says "GitHub".
 */
export function formatStars(n: number): string {
  if (!Number.isFinite(n) || n < 0) return ''
  if (n < 1000) return String(Math.round(n))
  if (n < 10_000) return `${(Math.floor(n / 100) / 10).toFixed(1).replace(/\.0$/, '')}k`
  return `${Math.floor(n / 1000)}k`
}

export function githubStars(): string {
  const raw = import.meta.env.VITE_GITHUB_STARS as string | undefined
  if (!raw) return ''
  const n = Number(raw)
  return Number.isFinite(n) ? formatStars(n) : ''
}
