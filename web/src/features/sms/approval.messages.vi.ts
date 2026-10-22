import { translate } from '@/i18n'
import { m } from './approval.messages'

export default translate(m, {
  title: 'Tin nhắn',
  caption: '{chars} ký tự · {segments, plural, other {# đoạn tin}} · {encoding}',
  bubble: 'Tin nhắn như người nhận thấy',
  countries: 'Người gửi và chi phí theo quốc gia',
  country: 'Quốc gia',
  sender: 'Người gửi',
  recipients: 'Người nhận',
  price: 'Giá mỗi đoạn tin',
  cost: 'Chi phí',
  skipped: 'Bị bỏ qua',
  total: '{recipients, plural, other {# tin nhắn}} · {cost}',
  schedule: 'Lịch gửi: {schedule}',
})
