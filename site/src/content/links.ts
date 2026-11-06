/**
 * Every outbound link on the page, in one place. The Landing board links its
 * app buttons to other boards; on reputecast.com they open the docs.
 */
const repo = 'https://github.com/hostrepute/reputecast'
const docs = 'https://docs.reputecast.com'

export const links = {
  repo,
  discussions: `${repo}/discussions`,
  docs: `${docs}/`,
  adapterSdk: `${docs}/adapters/community/`,
  releases: `${repo}/releases`,
  newIssue: `${repo}/issues/new/choose`,
  installGuide: `${docs}/self-hosting/docker/`,
  screenshots: `${repo}#screenshots`,
  flowEditor: `${docs}/automations/overview/`,
  hostrepute: 'https://hostrepute.com',
  withoutHostRepute: `${docs}/deliverability/hostrepute/`,
  composeUrl: 'https://get.reputecast.com/compose.yml',
} as const
