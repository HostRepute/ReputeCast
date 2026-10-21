import { translate } from '@/i18n'
import { m } from './overview.messages'

export default translate(m, {
  title: 'Nhà cung cấp SMS',
  sub: '{month}: đã gửi {sent} tin nhắn · {delivered} gửi thành công',
  manage: 'Nhà cung cấp SMS và định tuyến',
  budget: 'Chi tiêu {month}',
  of: 'trên {budget}',
  noBudget: 'Chi tiêu {month} {spend} · chưa đặt ngân sách hằng tháng',
})
