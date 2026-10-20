import { prepareCompile } from './mjml'

// The canvas shows a block marked rc-omit (a Sponsor block without a
// booking); the compiled email leaves it out, as the server's PrepareCompile
// does (internal/mjml).
describe('prepareCompile', () => {
  const doc =
    '<mjml><mj-body><mj-section><mj-column><mj-text>Story</mj-text></mj-column></mj-section>' +
    '<!-- rc:sponsor --><mj-section rc-sponsor="1" rc-omit="1" padding="12px 24px"><mj-column><mj-text>No sponsor is booked for #144.</mj-text></mj-column></mj-section>' +
    '<mj-section rc-omit="0"><mj-column><mj-text>Kept</mj-text></mj-column></mj-section></mj-body></mjml>'

  it('leaves out blocks marked rc-omit and strips the attribute elsewhere', () => {
    const out = prepareCompile(doc)
    expect(out).not.toContain('No sponsor is booked')
    expect(out).not.toContain('rc-')
    expect(out).toContain('Story')
    expect(out).toContain('Kept')
  })

  it('keeps marking conditional blocks around an omitted one', () => {
    const rule = JSON.stringify({ id: 'beta01', show: 'only', match: 'all', conditions: [] }).replace(/"/g, '&quot;')
    const out = prepareCompile(doc.replace('<mj-section rc-omit="0">', `<mj-section rc-visible="${rule}">`))
    expect(out).toContain('<mj-raw><!--rc:if beta01--></mj-raw>')
    expect(out).toContain('<mj-raw><!--rc:fi beta01--></mj-raw>')
    expect(out).not.toContain('No sponsor is booked')
  })
})
