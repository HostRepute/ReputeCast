import { translate } from '@/i18n'
import { mcard } from './card.messages'

/** Vietnamese for the workspace Overview's Earn card (docs/i18n/glossary-vi.md › Earn). */
export default translate(mcard, {
  title: 'Kiếm tiền · 30 ngày qua',
  label: 'Kiếm tiền, 30 ngày qua',
  line: '{provider} · {account} · {mode, select, test {Thử nghiệm} other {Chính thức}}',
  lineNoAccount: '{provider} · {mode, select, test {Thử nghiệm} other {Chính thức}}',
  mrr: 'MRR',
  members: 'Hội viên trả phí',
  churn: 'Rời bỏ',
  trials: 'Dùng thử',
  nextSponsor: 'Nhà tài trợ tiếp theo',
  nextIssue: '#{issue} · {sponsor}',
  creativeDue: 'Nội dung tài trợ đến hạn vào {date}',
  creativeReady: 'Nội dung tài trợ đã sẵn sàng',
  open: 'Mở Kiếm tiền',
  silentTitle: 'Webhook {provider} không có tin',
  silentText:
    'Không có sự kiện nào từ {provider} trong {hours, plural, other {# giờ}}, trong khi {active, plural, other {# thuê bao}} đang hoạt động. Trạng thái có thể chậm tối đa một giờ.',
  checkWebhook: 'Kiểm tra webhook',
  inviteLabel: 'Kiếm tiền',
  inviteTitle: 'Kiếm tiền từ bản tin của bạn',
  inviteText: 'Gói trả phí, quà tặng và tài trợ. Người đọc trả tiền cho bạn qua tài khoản Stripe của chính bạn; ReputeCast không thu phí.',
  setUp: 'Thiết lập Kiếm tiền',
  notNow: 'Để sau',
})
