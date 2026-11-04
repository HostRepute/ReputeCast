import type { Catalog } from '../define'

/** The German translation files, each loaded when German is picked (store.ts). */
export default import.meta.glob<Catalog>('../../**/*messages.de.ts', { import: 'default' })
