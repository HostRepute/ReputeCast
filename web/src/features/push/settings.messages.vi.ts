import { translate } from '@/i18n'
import { m } from './settings.messages'

/** Vietnamese for push and WhatsApp in Settings › Providers (docs/i18n/glossary-vi.md › Channels and SMS). */
export default translate(m, {
  pushTitle: 'Thông báo đẩy trên web',
  pushBuiltIn: 'Có sẵn · không cần đăng ký nhà cung cấp nào',
  keys: 'Khóa',
  keysCreated: 'Tạo ngày {date} · {name}',
  sites: 'Trang',
  noSites: 'Chưa có trang nào',
  subscribers: 'Người đăng ký',
  subscribersLine: '{total} · {linked} đã liên kết với liên hệ',
  cap: 'Giới hạn tần suất',
  capLine: '{n, plural, other {# thông báo}} mỗi ngày cho mỗi người đăng ký',
  pushFoot: 'Khóa, trang, lời nhắc, giờ yên lặng và giới hạn tần suất nằm ở Nhà cung cấp gửi thư › Thông báo đẩy.',
  pushOpen: 'Mở thiết lập thông báo đẩy',
  waTitle: 'WhatsApp',
  waProvider: 'WhatsApp Cloud API của Meta',
  business: 'Doanh nghiệp',
  businessLine: '{name} · {number}',
  quality: 'Xếp hạng chất lượng',
  limit: 'Giới hạn nhắn tin',
  limitLine: '{n, plural, other {# người}} mỗi ngày',
  templates: 'Mẫu',
  templatesLine: '{approved} đã phê duyệt · {pending} đang chờ · {rejected} bị từ chối',
  consent: 'Đồng ý nhận WhatsApp',
  consentLine: '{n, plural, other {# người}}',
  waFoot: 'Tài khoản, mẫu, bảng giá và sự kiện nằm ở Nhà cung cấp gửi thư › WhatsApp.',
  waOpen: 'Mở WhatsApp',
})
