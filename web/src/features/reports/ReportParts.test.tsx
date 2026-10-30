import { fireEvent, screen } from '@testing-library/react'
import { vi } from 'vitest'
import type { Schemas } from '@/api/client'
import { renderInWorkspace } from '@/test/app'
import { ClickMap } from './ClickMap'
import { Insights } from './Insights'
import { HeadlineTiles } from './Tiles'
import type { CampaignReport } from './words'

type Insight = Schemas['ReportInsight']

const report = (insights: Insight[]) => ({ insights }) as unknown as CampaignReport

describe('insights with actions', () => {
  const bounces: Insight = {
    kind: 'mailbox_bounces',
    tone: 'warn',
    title: 'Outlook bounced 1.0%, twice the rate of the other mailbox providers (0.5%).',
    detail: '101 of 10,076 messages. SNDS shows how Outlook sees your sending IPs.',
    action: 'open_snds',
    actionLabel: 'Open Microsoft SNDS',
    link: 'https://sendersupport.olc.protection.outlook.com/snds/',
  }
  const topLink: Insight = {
    kind: 'top_link',
    tone: 'info',
    title: 'The “Read the docs” link drew 38% of human clicks.',
    detail: '1,297 people clicked “Read the docs”, more than any other link.',
    action: 'show_link',
    actionLabel: 'Show on the click map',
    position: 6,
  }

  it("opens Microsoft SNDS in a new tab from Outlook's bounces", async () => {
    await renderInWorkspace(<Insights r={report([bounces])} onShowLink={() => {}} />)
    const snds = screen.getByRole('link', { name: 'Open Microsoft SNDS (opens in a new tab)' })
    expect(snds).toHaveAttribute('href', 'https://sendersupport.olc.protection.outlook.com/snds/')
    expect(snds).toHaveAttribute('target', '_blank')
  })

  it('shows the top link on the click map, in its variant for a test on content', async () => {
    const onShowLink = vi.fn()
    await renderInWorkspace(<Insights r={report([topLink, { ...topLink, position: 0, variant: 'B' }])} onShowLink={onShowLink} />)
    const [plain, inB] = screen.getAllByRole('button', { name: 'Show on the click map' })
    fireEvent.click(plain!)
    expect(onShowLink).toHaveBeenLastCalledWith(6, undefined)
    fireEvent.click(inB!)
    expect(onShowLink).toHaveBeenLastCalledWith(0, 'B')
  })

  it('checks delisting on Deliverability, and leaves actions out of a read-only link', async () => {
    const blocked: Insight = {
      kind: 'blocked',
      tone: 'bad',
      title: '11 messages were blocked by Outlook for the sending IP’s reputation (S3150).',
      detail: '',
      action: 'check_delisting',
      actionLabel: 'Check delisting',
    }
    const { unmount } = await renderInWorkspace(<Insights r={report([blocked, topLink])} onShowLink={() => {}} />)
    expect(screen.getByRole('link', { name: 'Check delisting' })).toHaveAttribute('href', '/w/acme-newsletter/deliverability')
    unmount()
    // A read-only link has no click map actions.
    await renderInWorkspace(<Insights r={report([topLink])} />)
    expect(screen.queryByRole('button', { name: 'Show on the click map' })).not.toBeInTheDocument()
  })
})

describe('the click map of a test on content', () => {
  const link = (position: number, label: string, uniqueClicks: number, share: number) =>
    ({ id: `${position}`, url: `https://acme.io/${label}`, label, position, clicks: uniqueClicks, uniqueClicks, botClicks: 0, share, grouped: false, hot: false }) as Schemas['ReportLink']
  const variant = (key: string, total: number, links: Schemas['ReportLink'][]) =>
    ({ key, sameEmail: key === 'A', html: '', available: false, total, botTotal: 0, links, rows: [] }) as Schemas['ReportVariantClickMap']
  const map = {
    html: '',
    available: false,
    total: 120,
    botTotal: 0,
    links: [link(0, 'a-docs', 120, 100)],
    threshold: 5,
    rows: [],
    perVariant: true,
    variants: [variant('A', 120, [link(0, 'a-docs', 120, 100)]), variant('B', 260, [link(0, 'b-guide', 260, 74.3), link(1, 'b-blog', 90, 25.7)])],
  } as Schemas['ReportClickMap']

  it('has no map of everyone: it starts on variant A, with the variant picker', async () => {
    const onVariant = vi.fn()
    await renderInWorkspace(<ClickMap m={map} clicks={380} imported={false} emailShared={false} view="human" onView={() => {}} onVariant={onVariant} />)
    const picker = screen.getByRole('combobox', { name: 'Recipients' })
    expect(Array.from((picker as HTMLSelectElement).options).map((o) => o.text)).toEqual(['Variant A', 'Variant B'])
    expect(picker).toHaveValue('A')
    // The totals are A's, said to be A's: not "380 people made 120 clicks".
    expect(screen.getByText("Share of 120 human clicks by variant A's recipients, each person once per link.")).toBeInTheDocument()
    expect(screen.queryByText(/people made/)).not.toBeInTheDocument()
    fireEvent.change(picker, { target: { value: 'B' } })
    expect(onVariant).toHaveBeenCalledWith('B')
    expect(screen.getByText("Share of 260 human clicks by variant B's recipients, each person once per link.")).toBeInTheDocument()
    expect(screen.getByRole('rowheader', { name: /b-guide/ })).toBeInTheDocument()
  })

  it('keeps Everyone for a test on the subject, where every variant shares the email', async () => {
    const subject = { ...map, perVariant: false, variants: map.variants.map((x) => ({ ...x, sameEmail: true })) }
    await renderInWorkspace(<ClickMap m={subject} clicks={380} imported={false} emailShared={false} view="human" onView={() => {}} />)
    const picker = screen.getByRole('combobox', { name: 'Recipients' })
    expect(Array.from((picker as HTMLSelectElement).options).map((o) => o.text)).toEqual(['Everyone', 'Variant A', 'Variant B'])
    expect(picker).toHaveValue('')
  })
})

describe('headline tiles', () => {
  const tiles = {
    campaign: { id: 'c', name: 'October Product Update', status: 'sent', final: true, notes: [] },
    headline: {
      sent: 47980,
      delivered: 47548,
      deliveredRate: 99.1,
      undelivered: 0,
      humanOpens: 19590,
      humanOpenRate: 41.2,
      humanOpenRateMeasurable: 52.8,
      measurable: 37112,
      privacyOpens: 10436,
      privacyOpenRate: 21.9,
      machineOpens: 1240,
      clicks: 2996,
      clickRate: 6.3,
      clickToOpen: 15.3,
      botClicks: 0,
      unsubscribes: 112,
      unsubscribeRate: 0.24,
      complaints: 19,
      complaintRate: 0.04,
      bounces: 288,
      bounceRate: 0.6,
      hardBounces: 193,
      softBounces: 79,
      sentBy: [],
      notReportedBy: [],
    },
    providers: { split: false, columns: [] },
    comparisons: {
      issues: 6,
      tiles: [
        { metric: 'sent', change: 2.4, unit: 'percent', median: 46856, tone: 'neutral', trend: [45120, 46000, 46500, 47000, 46400, 47210, 47980] },
        { metric: 'humanOpens', change: 2.1, unit: 'points', median: 50.7, tone: 'good', trend: [49.9, 50.4, 49.7, 50.7, 51.0, 50.6, 52.8] },
        { metric: 'delivered', change: -0.1, unit: 'points', median: 99.2, tone: 'bad', trend: [99.2, 99.1, 99.3, 99.2, 99.0, 99.2, 99.1] },
      ],
    },
  } as unknown as CampaignReport

  it('draws each compared tile’s trend line and reads it out', async () => {
    await renderInWorkspace(<HeadlineTiles r={tiles} />)
    expect(screen.getByRole('img', { name: 'Up over your last 6 issues, from 45,120 to 47,980' })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Up over your last 6 issues, from 49.9% to 52.8%' })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Down over your last 6 issues, from 99.2% to 99.1%' })).toBeInTheDocument()
    // A tile that isn't compared has no line.
    expect(screen.getAllByRole('img')).toHaveLength(3)
  })

  it('draws no trend while the results still come in', async () => {
    await renderInWorkspace(<HeadlineTiles r={{ ...tiles, campaign: { ...tiles.campaign, final: false } }} />)
    expect(screen.queryAllByRole('img')).toHaveLength(0)
  })
})
