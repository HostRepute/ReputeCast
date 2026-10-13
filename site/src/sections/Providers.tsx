const providers = [
  'Amazon SES',
  'Postmark',
  'Mailgun',
  'SendGrid',
  'Resend',
  'SparkPost',
  'Mailjet',
  'Brevo',
  'Azure Communication Services',
  'Postal · self-hosted',
  'Any SMTP server',
]

const routes: { kind: string; segments: { width: string; color: string }[]; note: string; delay: string }[] = [
  {
    kind: 'Broadcasts',
    segments: [
      { width: '90%', color: 'var(--chart-ses)' },
      { width: '10%', color: 'var(--chart-postal)' },
    ],
    note: 'SES 90% · Postal 10% warming up',
    delay: 'g1',
  },
  { kind: 'Automations', segments: [{ width: '100%', color: 'var(--chart-ses)' }], note: 'SES, failover Postmark', delay: 'g2' },
  { kind: 'System', segments: [{ width: '100%', color: 'var(--chart-postmark)' }], note: 'Postmark, failover SES', delay: 'g3' },
]

function Chip({ name, copy, dashed }: { name: string; copy?: boolean; dashed?: boolean }) {
  return (
    <span className={`chip${dashed ? ' border-dashed' : ''}`} role={copy ? undefined : 'listitem'} aria-hidden={copy || undefined}>
      {name}
    </span>
  )
}

/** The provider marquee and the routing card. */
export function Providers() {
  return (
    <section className="wrap flex flex-col gap-5 pt-10 pb-10" id="providers" aria-label="Sending providers">
      <div className="marquee reveal">
        {/* The list is doubled so the loop is seamless; the copy is hidden from
            screen readers. Hover or the Pause motion control stops it. */}
        <div className="track" role="list" aria-label="Supported sending providers">
          {providers.map((p) => (
            <Chip key={p} name={p} />
          ))}
          <Chip name="+ community adapters" dashed />
          {providers.map((p) => (
            <Chip key={`copy-${p}`} name={p} copy />
          ))}
          <Chip name="+ community adapters" dashed copy />
        </div>
      </div>
      <div className="card reveal flex flex-wrap items-center gap-6 p-[22px]">
        <div className="flex flex-[1_1_300px] flex-col gap-1.5">
          <h3 className="m-0 text-[17px] font-bold">Route each kind of mail where it belongs</h3>
          <span className="k">Broadcasts, Automations and System mail each get a primary, a fallback, conditions and weighted splits.</span>
        </div>
        <div className="flex flex-[2_1_420px] flex-col gap-2.5 text-[14px]">
          {routes.map((r) => (
            <div key={r.kind} className="flex items-center gap-3 max-md:flex-wrap">
              <span className="w-[110px] font-semibold">{r.kind}</span>
              <div className={`grow ${r.delay} flex h-3.5 flex-1 gap-0.5 overflow-hidden rounded-[7px] bg-line-2`} aria-hidden="true">
                {r.segments.map((s, i) => (
                  <i key={i} className="block" style={{ width: s.width, background: s.color }} />
                ))}
              </div>
              <span className="k w-[200px] max-md:w-auto">{r.note}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
