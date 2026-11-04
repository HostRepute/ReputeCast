import { translate } from '@/i18n'
import { mbk } from './brandkit.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(mbk, {
  banner: '<b>Mit Ihrem Brand-Kit beginnen</b> Logo, Farben und Schriften von Ihrer Website, angewendet auf jede Vorlage und jeden neuen Block.',
  setUp: 'Brand-Kit einrichten',
  defaultColours: 'In Standardfarben angezeigt, bis Sie das Brand-Kit einrichten',
  synced: 'Synchronisiert',
  syncedHeader: 'Kopfzeile + Logo',
  syncedFooter: 'Fußzeile + Abmeldung',
  syncedHeaderNotice: 'Synchronisierte Kopfzeile.',
  syncedFooterNotice: 'Synchronisierte Fußzeile.',
  syncedNoticeBody: 'Änderungen erfolgen im Brand-Kit, und jeder Entwurf und jede geplante E-Mail mit diesem Block übernimmt sie.',
  editInBrandKit: 'Im Brand-Kit bearbeiten',
  syncedBlock: 'Synchronisierter Block',
  syncedNamedNotice: 'Synchronisierter Block „{name}“.',
  syncedGone: 'Dieser synchronisierte Block wurde aus dem Brand-Kit gelöscht.',
  syncedGoneBody: 'Nichts ändert ihn mehr. Heben Sie die Verknüpfung auf, um ihn in dieser E-Mail zu bearbeiten.',
  unlink: 'Verknüpfung aufheben',
  unlinked: 'Verknüpfung mit dem synchronisierten Block aufgehoben',
  unlinkedDetail: 'Der Abschnitt gehört jetzt nur zu dieser E-Mail: Bearbeiten Sie ihn hier. Rückgängig stellt die Verknüpfung wieder her.',
  saveSyncedTitle: 'Synchronisierter Block',
  saveSyncedLead: 'Verwenden Sie diesen Abschnitt in anderen E-Mails wieder. Bearbeiten Sie ihn einmal im Brand-Kit, und jeder Entwurf damit übernimmt die Änderung.',
  saveSynced: 'Als synchronisierten Block speichern…',
  saveSyncedDialog: 'Als synchronisierten Block speichern',
  saveSyncedDialogLead: 'Der Abschnitt kommt ins Brand-Kit, und diese E-Mail enthält an seiner Stelle den synchronisierten Block. Fügen Sie ihn in anderen E-Mails über Blöcke › Ihre Blöcke ein.',
  syncedName: 'Name',
  saveSyncedConfirm: 'Synchronisierten Block speichern',
  saveSyncedFailed: 'Der synchronisierte Block konnte nicht gespeichert werden: {error}',
  savedSynced: '„{name}“ als synchronisierter Block gespeichert',
  savedSyncedDetail: 'Fügen Sie ihn über Blöcke › Ihre Blöcke ein. Bearbeiten Sie ihn im Brand-Kit: Jeder Entwurf damit übernimmt die Änderung.',
})
