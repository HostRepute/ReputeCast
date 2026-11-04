import type { Catalog } from '../define'

/** The Vietnamese translation files, each loaded when Vietnamese is picked (store.ts). */
export default import.meta.glob<Catalog>('../../**/*messages.vi.ts', { import: 'default' })
