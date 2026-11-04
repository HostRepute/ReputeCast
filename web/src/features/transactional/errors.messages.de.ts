import { translate } from '@/i18n'
import { me } from './errors.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(me, {
  template_not_found_when: 'Die ID existiert in diesem Arbeitsbereich nicht',
  template_not_found_do: 'Prüfen Sie die ID unter Vorlagen',
  template_not_published_when: 'Die Vorlage hat einen Entwurf, aber keine Live-Version',
  template_not_published_do: 'Veröffentlichen Sie sie zuerst',
  missing_variables_when: 'Eine erforderliche Variable fehlt in {data}',
  missing_variables_do: 'Die Antwort listet auf, welche',
  invalid_recipient_when: 'Die Adresse ist keine gültige E-Mail-Adresse',
  invalid_recipient_do: 'Validieren Sie sie in Ihrer App',
  recipient_suppressed_when: 'Für die Adresse gab es einen Bounce oder eine Beschwerde, oder sie ist eine Spamfalle',
  recipient_suppressed_do: 'Die Antwort nennt Grund und Datum',
  sender_not_verified_when: '{from} ist kein verifizierter Absender für Transaktional',
  sender_not_verified_do: 'Verifizieren Sie ihn unter Einstellungen › Absender',
  attachments_too_large_when: 'Über dem Limit der Route',
  attachments_too_large_do: '{size} bei {provider}',
  rate_limited_when: 'Über der Rate des Schlüssels',
  rate_limited_do: 'Warten Sie gemäß {header}',
  no_healthy_route_when: 'Jeder Anbieter der Route ist ausgefallen',
  no_healthy_route_do: 'Versuchen Sie es mit demselben Idempotenzschlüssel erneut',
  idempotency_key_reused_when: 'Derselbe Schlüssel kam innerhalb von 24 Stunden mit einer anderen Anfrage',
  idempotency_key_reused_do: 'Verwenden Sie für eine neue Nachricht einen neuen Schlüssel',
  template_not_allowed_when: 'Der Schlüssel darf diese Vorlage oder Kategorie nicht senden',
  template_not_allowed_do: 'Verwenden Sie einen Schlüssel, der das darf, oder erweitern Sie den Bereich des Schlüssels',
})
