/**
 * Every outbound link on the page, in one place. The Landing board links its
 * app buttons to other boards; on reputecast.com they open the live demo.
 */
const repo = 'https://github.com/hostrepute/reputecast'
const demo = 'https://demo.reputecast.com'

export const links = {
  repo,
  discord: 'https://discord.gg/reputecast',
  docs: `${repo}/tree/main/docs`,
  adapterSdk: `${repo}/blob/main/docs/adapters.md#community-adapters`,
  releases: `${repo}/releases`,
  installGuide: `${repo}#quick-start`,
  hostrepute: 'https://hostrepute.com',
  composeUrl: 'https://get.reputecast.com/compose.yml',
  demo,
  demoFlowEditor: `${demo}/w/acme-newsletter/automations`,
  demoAutomations: `${demo}/w/acme-newsletter/automations`,
  demoWithoutHostRepute: `${demo}/w/acme-newsletter/deliverability`,
} as const
