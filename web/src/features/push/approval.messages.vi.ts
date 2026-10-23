import { translate } from '@/i18n'
import { m } from './approval.messages'

/** Vietnamese for push and WhatsApp campaigns in the approval review (docs/i18n/glossary-vi.md › Approvals). */
export default translate(m, {
  pushTitle: 'Thông báo',
  pushCaption: '{n, plural, other {# người đăng ký}} trên {sites} · {schedule}',
  link: 'Liên kết',
  buttons: 'Nút',
  expiry: 'Hết hạn sau',
  hours: '{n, plural, other {# giờ}}',
  minutes: '{n, plural, other {# phút}}',
  tag: 'Thẻ',
  none: '–',
  waTitle: 'Tin nhắn WhatsApp',
  waCaption: '{template} · {category} · {language} · {n, plural, other {# người}} · {schedule}',
  business: 'Gửi dưới tên',
  variables: 'Biến',
  varField: "'{{'{key}'}}' lấy từ {field}",
  varFieldFallback: "'{{'{key}'}}' lấy từ {field}, hoặc “{fallback}”",
  varText: "'{{'{key}'}}' {text}",
  cost: 'Chi phí ước tính',
  costLine: '{amount}, Meta tính phí khi tin được gửi thành công',
})
