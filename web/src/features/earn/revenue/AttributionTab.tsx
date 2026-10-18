import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Input, Select, Toggle } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { ago, clock, count, dayMonthDate, dayMonthYear } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { CreateKeyDialog } from '@/features/keys/CreateKeyDialog'
import { KeySecretDialog } from '@/features/keys/KeySecretDialog'
import { CodeBox, type CodeLang } from '@/features/transactional/shared'
import { highlight } from '@/features/transactional/codeSamples'
import { EARN } from '../EarnPage'
import { money } from '../format'
import { CampaignLink } from '../sponsors/parts'
import { ma } from './attribution.messages'
import { RevenueCard } from './RevenueOverview'
import './revenue.css'

type Settings = Schemas['AttributionSettings']
type TT = ReturnType<typeof useT>

const CONVERSIONS = '/api/v1/workspaces/{workspace}/conversions' as const
const presets = [1, 3, 7, 14, 30]
const mono = (c: ReactNode) => <span className="mono">{c}</span>

/**
 * Earn › Settings › Attribution (EarnSettingsTabs): the attribution window,
 * the click token on the publisher's own domains, the conversions log and
 * the conversion API with its keys, code samples and errors.
 */
export function AttributionTab() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${EARN}/attribution`, { params: { path: { workspace: slug } } })
  if (q.error) {
    return (
      <Note tone="bad" label={t(ma.errorLoad)}>
        {errorMessage(q.error)}{' '}
        <button type="button" className="underline" onClick={() => void q.refetch()}>
          {t(ma.retry)}
        </button>
      </Note>
    )
  }
  if (!q.data) {
    return (
      <div className="card er-pad" aria-busy="true">
        <Skeleton w="40%" h={18} />
        <Skeleton w="100%" h={220} className="mt-4" />
      </div>
    )
  }
  const s = q.data
  return (
    <div className="flex flex-col gap-4">
      <div className="er-grid">
        <WindowCard key={`${s.days}|${s.clickToken}|${s.domains.join(',')}`} s={s} />
        <ConversionsLog s={s} />
      </div>
      <ApiCard s={s} />
    </div>
  )
}

/** "app.acme.io" from what was typed, or nothing when it isn't a domain. */
function cleanDomain(raw: string): string | undefined {
  let d = raw.trim().toLowerCase()
  if (d.includes('://')) {
    try {
      d = new URL(d).host
    } catch {
      return undefined
    }
  }
  d = d.replace(/^\*\./, '').replace(/[/?#].*$/, '').replace(/:\d+$/, '').replace(/\.$/, '')
  return /^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z][a-z0-9-]{0,61}[a-z0-9]$/.test(d) ? d : undefined
}

/** The window and the click token, saved together from a footer that shows while they're changed. */
function WindowCard({ s }: { s: Settings }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const save = $api.useMutation('put', `${EARN}/attribution`)
  const [days, setDays] = useState(String(s.days))
  const [custom, setCustom] = useState(!presets.includes(s.days))
  const [token, setToken] = useState(s.clickToken)
  const [domains, setDomains] = useState<string[]>(s.domains)
  const [typed, setTyped] = useState('')
  const [domainError, setDomainError] = useState<string>()
  const [saveError, setSaveError] = useState<string>()
  const ro = !s.canManage
  const n = Number(days)
  const daysOk = /^\d+$/.test(days.trim()) && n >= 1 && n <= 30
  const dirty = days !== String(s.days) || token !== s.clickToken || domains.join(',') !== s.domains.join(',')
  const needsDomain = token && domains.length === 0
  const add = () => {
    if (!typed.trim()) return
    const d = cleanDomain(typed)
    if (!d) {
      setDomainError(t(ma.domainInvalid, { domain: typed.trim() }))
      return
    }
    setDomainError(undefined)
    if (!domains.includes(d)) setDomains([...domains, d])
    setTyped('')
  }
  const submit = () => {
    setSaveError(undefined)
    save.mutate(
      { params: { path: { workspace: slug } }, body: { days: n, clickToken: token, domains } },
      {
        onSuccess: (next) => {
          qc.setQueryData(['get', `${EARN}/attribution`, { params: { path: { workspace: slug } } }], next)
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/revenue`] })
          toast({ kind: 'success', title: t(ma.saved), detail: t(ma.savedText) })
        },
        onError: (e) => setSaveError(errorMessage(e)),
      },
    )
  }
  return (
    <RevenueCard title={t(ma.title)}>
      <div className="er-at-row">
        <div className="er-at-text">
          <span className="er-at-ttl" id="at-window">
            {t(ma.windowTitle)}
          </span>
          <span className="k">{t(ma.windowText)}</span>
        </div>
        <div className="er-at-ctl flex flex-col items-end gap-1">
          {custom ? (
            <>
              <Input
                type="text"
                inputMode="numeric"
                className="w-[140px]"
                aria-labelledby="at-window"
                aria-describedby={daysOk ? undefined : 'at-window-err'}
                invalid={!daysOk}
                value={days}
                disabled={ro}
                onChange={(e) => setDays(e.target.value)}
              />
              {daysOk ? (
                <button type="button" className="k underline" disabled={ro} onClick={() => setCustom(false)} hidden={!presets.includes(n)}>
                  {t(ma.windowPresets)}
                </button>
              ) : (
                <span className="err text-right" id="at-window-err" role="alert">
                  {t(ma.windowError)}
                </span>
              )}
            </>
          ) : (
            <Select
              className="w-[140px]"
              aria-labelledby="at-window"
              value={days}
              disabled={ro}
              onChange={(e) => {
                if (e.target.value === 'custom') setCustom(true)
                else setDays(e.target.value)
              }}
            >
              {presets.map((d) => (
                <option key={d} value={String(d)}>
                  {t(ma.windowDays, { days: d })}
                </option>
              ))}
              <option value="custom">{t(ma.windowCustom)}</option>
            </Select>
          )}
        </div>
      </div>
      <div className="er-at-row">
        <div className="er-at-text">
          <span className="er-at-ttl" id="at-token">
            {t(ma.tokenTitle)}
          </span>
          <span className="k">
            <T msg={ma.tokenText} values={{ mono }} />
          </span>
        </div>
        <Toggle label={t(ma.tokenLabel)} hideLabel checked={token} disabled={ro} onChange={(e) => setToken(e.target.checked)} />
      </div>
      <div className="er-at-row er-at-domains">
        <span className="k font-semibold" id="at-domains">
          {t(ma.domains)}
        </span>
        <ul className="m-0 flex list-none flex-wrap gap-1.5 p-0" aria-labelledby="at-domains">
          {domains.map((d) => (
            <li key={d} className="chip pr-1">
              <span className="mono">{d}</span>
              {ro ? null : (
                <button type="button" aria-label={t(ma.removeDomain, { domain: d })} onClick={() => setDomains(domains.filter((x) => x !== d))}>
                  <Icon name="close" size={12} />
                </button>
              )}
            </li>
          ))}
          {domains.length === 0 ? <li className="k">{t(ma.domainsNone)}</li> : null}
        </ul>
        {ro ? null : (
          <div className="flex flex-wrap items-center gap-2">
            <Input
              type="text"
              className="w-[180px]"
              placeholder={t(ma.addDomain)}
              aria-label={t(ma.addDomain)}
              aria-invalid={domainError ? true : undefined}
              aria-describedby={domainError ? 'at-domain-err' : undefined}
              value={typed}
              onChange={(e) => setTyped(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  add()
                }
              }}
            />
            <Button size="sm" onClick={add}>
              {t(ma.add)}
            </Button>
          </div>
        )}
        {domainError ? (
          <span className="err basis-full" id="at-domain-err" role="alert">
            {domainError}
          </span>
        ) : null}
      </div>
      {dirty && !ro ? (
        <div className="er-at-foot">
          <span className={saveError ? 'err' : 'k'} role={saveError ? 'alert' : undefined}>
            {saveError ?? (!daysOk ? t(ma.notSaved) : needsDomain ? t(ma.notSavedDomains) : t(ma.unsaved))}
          </span>
          <Button variant="primary" size="sm" loading={save.isPending} disabled={!daysOk || needsDomain} onClick={submit}>
            {t(ma.save)}
          </Button>
        </div>
      ) : null}
    </RevenueCard>
  )
}

/** "09:48" today, else "14 Oct 17:02". */
function logWhen(d: string, now: number): string {
  const same = dayMonthYear(d) === dayMonthYear(now)
  return same ? clock(d) : `${dayMonthDate(d)} ${clock(d)}`
}

function ConversionsLog({ s }: { s: Settings }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [limit, setLimit] = useState(5)
  const [now] = useState(() => Date.now())
  const q = $api.useQuery('get', CONVERSIONS, { params: { path: { workspace: slug }, query: { limit } } }, { placeholderData: keepPreviousData })
  const sum = s.summary
  const cur = s.currency
  const sub = sum.name
    ? sum.sender
      ? t(ma.logAll, { count: sum.count, name: sum.name, value: money(sum.typicalCents ?? 0, cur), sender: sum.sender })
      : t(ma.logAllNoSender, { count: sum.count, name: sum.name, value: money(sum.typicalCents ?? 0, cur) })
    : sum.count
      ? t(ma.logSome, { count: sum.count })
      : t(ma.logNone)
  return (
    <RevenueCard title={t(ma.logTitle)} sub={sub}>
      {q.error ? (
        <div className="er-pad">
          <Note tone="bad" label={t(ma.logError)}>
            {errorMessage(q.error)}{' '}
            <button type="button" className="underline" onClick={() => void q.refetch()}>
              {t(ma.retry)}
            </button>
          </Note>
        </div>
      ) : !q.data ? (
        <div className="er-pad">
          <Skeleton w="100%" h={160} />
        </div>
      ) : q.data.items.length ? (
        <div className="overflow-x-auto">
          <table className="tbl er-tbl" aria-label={t(ma.logTitle)}>
            <thead>
              <tr>
                <th scope="col">{t(ma.colTime)}</th>
                <th scope="col" className="text-right">
                  {t(ma.colValue)}
                </th>
                <th scope="col">{t(ma.colMatched)}</th>
                <th scope="col">{t(ma.colCredited)}</th>
              </tr>
            </thead>
            <tbody>
              {q.data.items.map((c) => (
                <tr key={c.id}>
                  <td className="mono whitespace-nowrap">{logWhen(c.receivedAt, now)}</td>
                  <td className="text-right tabular-nums">{money(c.valueCents, c.currency, 'always')}</td>
                  <td>
                    <span className="inline-flex flex-wrap gap-1">
                      <Badge tone="neutral">{c.matched === 'click_token' ? t(ma.byToken) : c.matched === 'email' ? t(ma.byEmail) : t(ma.byNone)}</Badge>
                      {c.test ? <Badge tone="purple">{t(ma.test)}</Badge> : null}
                    </span>
                  </td>
                  <td>
                    <Credited c={c} window={q.data.window} slug={slug} t={t} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      <div className="er-pad flex flex-col gap-2" style={{ paddingTop: 12 }}>
        <span className="k">
          {sum.count && sum.since
            ? t(ma.logFoot, {
                count: count(sum.count),
                date: dayMonthDate(sum.since),
                value: money(sum.valueCents, cur, 'never'),
                credited: count(sum.credited),
                creditedValue: money(sum.creditedCents, cur, 'never'),
                rest: count(sum.count - sum.credited),
                restValue: money(sum.valueCents - sum.creditedCents, cur, 'never'),
              })
            : t(ma.logFootNone)}
        </span>
        {q.data?.next && limit < 200 ? (
          <span>
            <Button size="sm" loading={q.isFetching} onClick={() => setLimit(Math.min(200, limit < 50 ? 50 : limit + 50))}>
              {t(ma.showMore)}
            </Button>
          </span>
        ) : null}
      </div>
    </RevenueCard>
  )
}

function Credited({ c, window, slug, t }: { c: Schemas['ConversionItem']; window: number; slug: string; t: TT }) {
  if (c.credited?.kind === 'campaign') return <CampaignLink id={c.credited.id} name={c.credited.name} status="sent" />
  if (c.credited?.kind === 'automation') {
    return (
      <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: c.credited.id }}>
        {c.credited.name}
      </Link>
    )
  }
  return <span className="k">{c.matched === 'none' ? t(ma.noReader) : t(ma.notCredited, { days: window })}</span>
}

/** The conversion API: its address, the keys that can call it, a sample per language and its errors. */
function ApiCard({ s }: { s: Settings }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [lang, setLang] = useState<CodeLang>('node')
  const [creating, setCreating] = useState(false)
  const [secret, setSecret] = useState<Schemas['APIKeyCreated']>()
  const test = $api.useMutation('post', `${EARN}/attribution/test`)
  const me = useMe().data?.user
  const keys = $api.useQuery('get', '/api/v1/workspaces/{workspace}/api-keys', { params: { path: { workspace: slug } } }, { enabled: creating && s.canCreateKey })
  const [now] = useState(() => Date.now())
  const path = new URL(s.endpoint, window.location.origin).pathname
  const code = conversionSample(lang, s.endpoint, s.currency.toUpperCase(), t(ma.sampleName))
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/attribution`] })
    void qc.invalidateQueries({ queryKey: ['get', CONVERSIONS] })
  }
  const sendTest = () =>
    test.mutate(
      { params: { path: { workspace: slug } } },
      {
        onSuccess: (r) => {
          refresh()
          toast({
            kind: 'success',
            title: t(ma.testSent),
            detail: r.credited ? t(ma.testSentCredited, { matched: r.matched, name: r.credited.name }) : r.matched === 'none' ? t(ma.testSentNone, { email: me?.email ?? '' }) : t(ma.testSentNot),
          })
        },
        onError: (e) => toast({ kind: 'error', title: t(ma.sendTest), detail: errorMessage(e), retry: sendTest }),
      },
    )
  const errors: [string, ReactNode, ReactNode][] = [
    ['missing_identity', <T key="w" msg={ma.errMissingWhen} values={{ mono }} />, t(ma.errMissingDo)],
    ['invalid_value', <T key="w" msg={ma.errValueWhen} values={{ mono }} />, t(ma.errValueDo)],
    ['invalid_currency', t(ma.errCurrencyWhen), t(ma.errCurrencyDo)],
    ['occurred_at_out_of_range', t(ma.errRangeWhen), t(ma.errRangeDo)],
    ['rate_limited', t(ma.errRateWhen), <T key="d" msg={ma.errRateDo} values={{ mono }} />],
  ]
  return (
    <RevenueCard title={t(ma.apiTitle)} sub={t(ma.apiText)}>
      <div className="er-pad flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone="info">POST</Badge>
          <span className="mono" style={{ fontSize: 12.5, overflowWrap: 'anywhere' }}>
            {path}
          </span>
        </div>
        {s.keys.length ? (
          s.keys.map((k) => (
            <div key={k.id} className="er-at-key">
              <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-0.5">
                <span className="flex flex-wrap items-center gap-2">
                  <b className="font-semibold">{k.name}</b>
                  {k.only ? <Badge tone="info">{t(ma.keyBadge)}</Badge> : null}
                </span>
                <span className="k">
                  <T msg={keyLine(k)} values={{ who: k.createdBy, date: dayMonthDate(k.createdAt), ago: ago(k.lastUsedAt, new Date(now)), ip: k.lastUsedIp ?? '', mono }} />
                </span>
              </div>
              <ButtonLink size="sm" to="/w/$workspace/settings/keys" params={{ workspace: slug }}>
                {t(ma.manageKeys)}
              </ButtonLink>
            </div>
          ))
        ) : (
          <Note tone="neutral">
            <T msg={ma.noKey} values={{ mono }} />
          </Note>
        )}
        <CodeBox lang={lang} onLang={setLang} code={code}>
          {highlight(code, lang)}
        </CodeBox>
        <div className="flex flex-wrap items-center gap-2.5">
          {s.canManage ? (
            <Button variant="primary" size="sm" loading={test.isPending} onClick={sendTest}>
              {t(ma.sendTest)}
            </Button>
          ) : null}
          {s.canCreateKey ? (
            <Button size="sm" onClick={() => setCreating(true)}>
              {t(ma.createKey)}
            </Button>
          ) : null}
          <span className="k">{t(ma.otherCurrency)}</span>
        </div>
      </div>
      <div className="overflow-x-auto max-md:hidden">
        <table className="tbl er-tbl" aria-label={t(ma.apiTitle)}>
          <thead>
            <tr>
              <th scope="col">{t(ma.colError)}</th>
              <th scope="col">{t(ma.colWhen)}</th>
              <th scope="col">{t(ma.colDo)}</th>
            </tr>
          </thead>
          <tbody>
            {errors.map(([codeName, when, todo]) => (
              <tr key={codeName}>
                <td className="mono">{codeName}</td>
                <td>{when}</td>
                <td>{todo}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {/* Phones: a row per error. */}
      <ul className="m-0 hidden list-none p-0 max-md:block" aria-label={t(ma.apiTitle)}>
        {errors.map(([codeName, when, todo]) => (
          <li key={codeName} className="er-lrow">
            <span className="mono font-semibold">{codeName}</span>
            <span className="k">{when}</span>
            <span className="k">{todo}</span>
          </li>
        ))}
      </ul>
      {creating && keys.data ? (
        <CreateKeyDialog
          kind="api_key"
          list={keys.data}
          initialPreset="conversions"
          onClose={() => setCreating(false)}
          onCreated={(c) => {
            setCreating(false)
            setSecret(c)
            refresh()
          }}
        />
      ) : null}
      {secret ? <KeySecretDialog created={secret} action="created" onClose={() => setSecret(undefined)} /> : null}
    </RevenueCard>
  )
}

function keyLine(k: Schemas['ConversionKey']) {
  if (!k.only) return ma.keyLineMore
  if (!k.lastUsedAt) return ma.keyLineUnused
  return k.lastUsedIp ? ma.keyLine : ma.keyLineNoIp
}

/** The conversion API in each language, with the sample's values. */
function conversionSample(lang: CodeLang, endpoint: string, currency: string, name: string): string {
  const reply = '202 Accepted → { id: "0192f3a0-…-7d1e", matched: "click_token" }'
  switch (lang) {
    case 'curl':
      return [
        `$ curl -X POST ${endpoint} \\`,
        '  -H "Authorization: Bearer $REPUTECAST_CONVERSIONS_KEY" \\',
        '  -H "Idempotency-Key: upgrade-4821-in_1Q9" \\',
        '  -H "Content-Type: application/json" \\',
        "  -d '{",
        '    "click_token": "rc_click from the link, if present",',
        '    "email": "sam@example.com",',
        `    "name": "${name}",`,
        '    "value": "29.00",',
        `    "currency": "${currency}"`,
        "  }'",
        `# ${reply}`,
      ].join('\n')
    case 'python':
      return [
        'import os, requests',
        '',
        'r = requests.post(',
        `    "${endpoint}",`,
        '    headers={',
        '        "Authorization": f"Bearer {os.environ[\'REPUTECAST_CONVERSIONS_KEY\']}",',
        '        "Idempotency-Key": f"upgrade-{account.id}-{invoice.id}",',
        '    },',
        '    json={',
        '        "click_token": request.cookies.get("rc_click"),  # from ?rc_click=…, if present',
        '        "email": user.email,',
        `        "name": "${name}",`,
        '        "value": "29.00",',
        `        "currency": "${currency}",`,
        '    },',
        ')',
        `# ${reply}`,
      ].join('\n')
    case 'go':
      return [
        'body, _ := json.Marshal(map[string]any{',
        '\t"click_token": clickToken, // from ?rc_click=…, if present',
        '\t"email":       user.Email,',
        `\t"name":        "${name}",`,
        '\t"value":       "29.00",',
        `\t"currency":    "${currency}",`,
        '})',
        `req, _ := http.NewRequest("POST", "${endpoint}", bytes.NewReader(body))`,
        'req.Header.Set("Authorization", "Bearer "+os.Getenv("REPUTECAST_CONVERSIONS_KEY"))',
        'req.Header.Set("Idempotency-Key", "upgrade-"+account.ID+"-"+invoice.ID)',
        'req.Header.Set("Content-Type", "application/json")',
        'res, err := http.DefaultClient.Do(req)',
        `// ${reply}`,
      ].join('\n')
    case 'php':
      return [
        `$ch = curl_init("${endpoint}");`,
        'curl_setopt_array($ch, [',
        '    CURLOPT_POST => true,',
        '    CURLOPT_RETURNTRANSFER => true,',
        '    CURLOPT_HTTPHEADER => [',
        '        "Authorization: Bearer " . getenv("REPUTECAST_CONVERSIONS_KEY"),',
        '        "Idempotency-Key: upgrade-{$account->id}-{$invoice->id}",',
        '        "Content-Type: application/json",',
        '    ],',
        '    CURLOPT_POSTFIELDS => json_encode([',
        '        "click_token" => $_COOKIE["rc_click"] ?? null, // from ?rc_click=…, if present',
        '        "email" => $user->email,',
        `        "name" => "${name}",`,
        '        "value" => "29.00",',
        `        "currency" => "${currency}",`,
        '    ]),',
        ']);',
        '$response = curl_exec($ch);',
        `// ${reply}`,
      ].join('\n')
  }
  return [
    `await fetch("${endpoint}", {`,
    '  method: "POST",',
    '  headers: {',
    '    Authorization: `Bearer ${process.env.REPUTECAST_CONVERSIONS_KEY}`,',
    '    "Idempotency-Key": `upgrade-${account.id}-${invoice.id}`,',
    '    "Content-Type": "application/json",',
    '  },',
    '  body: JSON.stringify({',
    '    click_token: req.cookies.rc_click,   // from ?rc_click=…, if present',
    '    email: user.email,',
    `    name: "${name}",`,
    '    value: "29.00",',
    `    currency: "${currency}",`,
    '  }),',
    '});',
    `// ${reply}`,
  ].join('\n')
}
