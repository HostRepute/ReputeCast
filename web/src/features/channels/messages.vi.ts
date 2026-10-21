import { translate } from '@/i18n'
import { m } from './messages'

/** Vietnamese for the channel model's words (docs/i18n/glossary-vi.md › Channels and SMS). */
export default translate(m, {
  switchLabel: 'Kênh',
  email: 'Email',
  sms: 'SMS',
  push: 'Thông báo đẩy',
  whatsapp: 'WhatsApp',
  addChannel: 'Thêm kênh',
  addChannelHint: 'Tin nhắn, thông báo đẩy hoặc WhatsApp, bên cạnh email. Bộ chuyển kênh xuất hiện khi kênh thứ hai được thiết lập.',
  smsTermsTab: 'Điều khoản tin nhắn',
  all: 'Tất cả',
  sendsOn: 'Gửi qua {channels}',
  noChannelCampaigns:
    '{channel, select, sms {Chưa có chiến dịch SMS nào} push {Chưa có chiến dịch thông báo đẩy nào} whatsapp {Chưa có chiến dịch WhatsApp nào} other {Chưa có chiến dịch email nào}}',
})
