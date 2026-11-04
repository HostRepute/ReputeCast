import type { Catalog } from '../define'

/** The French translation files, each loaded when French is picked (store.ts). */
export default import.meta.glob<Catalog>('../../**/*messages.fr.ts', { import: 'default' })
