import { translate } from '@/i18n'
import { m } from './roles.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  owner: 'Inhaber',
  admin: 'Admin',
  editor: 'Bearbeiter',
  viewer: 'Betrachter',
  clientApprover: 'Freigeber (Kunde)',
  clientViewer: 'Betrachter (Kunde)',
})
