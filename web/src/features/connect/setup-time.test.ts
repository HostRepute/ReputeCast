import { describe, expect, it } from 'vitest'
import { m as setup } from '@/features/auth/setup.messages'
import setupVi from '@/features/auth/setup.messages.vi'
import { m as overview } from '@/features/overview/messages'
import overviewVi from '@/features/overview/messages.vi'
import { mo } from '@/features/providers/overview.messages'
import providersVi from '@/features/providers/overview.messages.vi'
import type { MessageDescriptor } from '@/i18n'
import { mses } from './connection.messages'
import connectionVi from './connection.messages.vi'
import { m as connect } from './messages'
import connectVi from './messages.vi'

// R3.U6, F-6: every screen that says how long connecting a provider takes
// says what the connect wizard says, and what its behaviour backs: most
// setups take 10–15 minutes; Amazon SES takes longer when ReputeCast doesn't
// run on AWS (the wizard then recommends IAM keys, created by hand, instead
// of the CloudFormation role); and a new AWS account can wait up to 24 hours
// for production access. Never "about 10 minutes".
const claims: [MessageDescriptor, Record<string, string>][] = [
  [setup.providerSub, setupVi],
  [connect.stepsFoot, connectVi],
  [mses.setupTime, connectionVi],
  [mo.emptyBody, providersVi],
  [overview.stepConnectLine, overviewVi],
]

describe('setup time', () => {
  it.each(claims.map(([d, vi]) => ({ id: d.id, en: d.message, vi: vi[d.id] ?? '' })))('$id states the honest setup time', ({ en, vi }) => {
    expect(en).toContain('Most setups take 10–15 minutes')
    expect(en).toContain("Amazon SES takes longer when ReputeCast doesn't run on AWS")
    expect(en).toContain('a new AWS account can wait up to 24 hours for production access')
    expect(en).not.toMatch(/about 10 minutes/)
    expect(vi).toContain('10–15 phút')
    expect(vi).toContain('Amazon SES mất lâu hơn khi ReputeCast không chạy trên AWS')
    expect(vi).toContain('24 giờ để có quyền gửi chính thức')
    expect(vi).not.toMatch(/khoảng 10 phút/)
  })
})
