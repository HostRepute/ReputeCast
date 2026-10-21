import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Note } from '@/components/ui/Note'
import { formatList, useT } from '@/i18n'
import { m } from './importconsent.messages'

const columnName = { date: m.date, source: m.source, text: m.text } as const

/**
 * The importer's Consent step, once a column maps to the phone: SMS
 * consent comes in only with its date, source and text (brief §3.3).
 */
export function SmsImportConsent({ sms }: { sms: Schemas['ImportSmsConsent'] }) {
  const t = useT()
  const missing = formatList(sms.missing.map((c) => (c in columnName ? t(columnName[c as keyof typeof columnName]) : c)))
  return (
    <section className="card flex flex-col gap-2.5 px-5 py-4.5" aria-labelledby="imp-sms-h">
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="h2" id="imp-sms-h">
          {t(m.title)}
        </h3>
        {sms.evidenceMapped ? <Badge tone="ok">{t(m.withEvidence)}</Badge> : null}
      </div>
      {sms.evidenceMapped ? (
        <span className="k">{t(m.withEvidenceText)}</span>
      ) : sms.consentMapped ? (
        <Note tone="warn">{t(m.missing, { missing, count: sms.missing.length })}</Note>
      ) : (
        <span className="k">{t(m.numbersOnly)}</span>
      )}
    </section>
  )
}
