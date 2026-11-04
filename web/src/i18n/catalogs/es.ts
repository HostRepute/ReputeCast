import type { Catalog } from '../define'

/** The Spanish translation files, each loaded when Spanish is picked (store.ts). */
export default import.meta.glob<Catalog>('../../**/*messages.es.ts', { import: 'default' })
