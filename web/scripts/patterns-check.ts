// Checks the patterns the shared components replaced (review 2026-11
// FE-3, FE-5), as part of `pnpm lint`:
//
//   - a switch is drawn with SwitchInput or Toggle (components/ui/Field),
//     never a hand-made <input role="switch">;
//   - useDebounced comes from lib/hooks, never a local copy;
//   - no file gains hex colours: colours come from the tokens in
//     styles/app.css. scripts/hex-baseline.json holds what each file had
//     when the check began; a file may only go down. `--update` writes the
//     lower counts back.
//
//   node scripts/patterns-check.ts [--update]
import { readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs'
import { join, relative } from 'node:path'

const root = new URL('..', import.meta.url).pathname
const src = join(root, 'src')
const baselinePath = join(root, 'scripts', 'hex-baseline.json')

function files(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) return files(p)
    return /\.(ts|tsx)$/.test(name) && !/\.test\.tsx?$/.test(name) && !name.endsWith('.d.ts') ? [p] : []
  })
}

const hex = /#[0-9A-Fa-f]{6}\b|['"`]#[0-9A-Fa-f]{3}['"`]/g
const problems: string[] = []
const counts: Record<string, number> = {}
for (const p of files(src)) {
  const rel = relative(root, p)
  const text = readFileSync(p, 'utf8')
  if (/role="switch"/.test(text) && rel !== 'src/components/ui/Field.tsx') problems.push(`${rel}: draw switches with SwitchInput or Toggle from components/ui/Field`)
  if (/function useDebounced\b/.test(text) && rel !== 'src/lib/hooks.ts') problems.push(`${rel}: import useDebounced from lib/hooks instead of a local copy`)
  if (/messages(\.vi)?\.ts$/.test(rel)) continue
  const n = (text.match(hex) ?? []).length
  if (n) counts[rel] = n
}

const baseline: Record<string, number> = JSON.parse(readFileSync(baselinePath, 'utf8'))
if (process.argv.includes('--update')) {
  const next: Record<string, number> = {}
  for (const [f, n] of Object.entries(counts)) next[f] = Math.min(n, baseline[f] ?? n)
  writeFileSync(baselinePath, JSON.stringify(next, Object.keys(next).sort(), 2) + '\n')
}
for (const [f, n] of Object.entries(counts)) {
  const allowed = baseline[f] ?? 0
  if (n > allowed) problems.push(`${f}: ${n} hex colours, ${allowed} allowed: use a token from styles/app.css`)
}
const total = Object.values(counts).reduce((a, b) => a + b, 0)
if (problems.length) {
  console.error(problems.join('\n'))
  process.exit(1)
}
console.log(`Patterns: switches and useDebounced shared; ${total} hex colours left in ${Object.keys(counts).length} files (none added).`)
