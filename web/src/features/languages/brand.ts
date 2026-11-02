/** Trailing words a workspace name often carries that the copy leaves out. */
const generic = new Set(['newsletter', 'newsletters', 'news', 'team', 'workspace', 'inc', 'inc.', 'ltd', 'ltd.', 'llc'])

/**
 * "Acme Newsletter" → "Acme": how the boards name the brand in page copy.
 * On its own, so the contact screens that name the brand don't load the
 * Settings languages card.
 */
export function brandOf(name: string): string {
  const words = name.trim().split(/\s+/)
  while (words.length > 1 && generic.has((words[words.length - 1] ?? '').toLowerCase())) words.pop()
  return words.join(' ')
}
