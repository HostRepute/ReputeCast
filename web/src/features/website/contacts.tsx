import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { useWebsite } from './api'
import { me } from './elsewhere.messages'

// The website's part of the contacts filters, in a file of its own: the
// rest of its screens' parts (elsewhere.tsx) bring the site preview's frame
// and the posts' copy, which Contacts would download for two chips.

/** The contacts filters' members: "Signed in to the website · 2,804" and "Paid members · 1,184". */
export function MembersChips({
  signedIn,
  onSignedIn,
  paid,
  onPaid,
}: {
  signedIn: boolean
  onSignedIn: (on: boolean) => void
  /** Earn's "Paid status: with access" filter (X16). */
  paid: boolean
  onPaid: (on: boolean) => void
}) {
  const t = useT()
  const ov = useWebsite()
  const d = ov.data
  if (!d || d.status !== 'on') return null
  return (
    <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t(me.filterLabel)}>
      <span className="k mr-1">{t(me.filterLabel)}</span>
      <button type="button" className="vchip" aria-pressed={signedIn} onClick={() => onSignedIn(!signedIn)}>
        {t(me.filterSignedIn, { count: count(d.members.signedIn) })}
      </button>
      {d.members.paid ? (
        <button type="button" className="vchip" aria-pressed={paid} onClick={() => onPaid(!paid)}>
          {t(me.filterPaid, { count: count(d.members.paid) })}
        </button>
      ) : null}
    </div>
  )
}
