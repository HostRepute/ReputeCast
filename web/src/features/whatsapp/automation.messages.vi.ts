import { translate } from '@/i18n'
import { m } from './automation.messages'

/**
 * Vietnamese for the Send WhatsApp step in automations (docs/i18n/glossary-vi.md
 * › Channels and SMS), worded like the Send SMS step. The step has two
 * windows: WhatsApp's 24-hour customer service window ("khung 24 giờ", in
 * full "khung chăm sóc khách hàng") and the quiet hours' sending window
 * ("khung giờ gửi").
 */
export default translate(m, {
  palette: 'Gửi WhatsApp',
  sub: 'WhatsApp Cloud API của Meta · {number}',
  headTemplate: 'mẫu',
  headText: 'văn bản tự do',
  name: 'Tên',
  namePlaceholder: 'ví dụ: Liên kết phát trực tiếp',
  message: 'Tin nhắn',
  modeTemplate: 'Mẫu đã được phê duyệt',
  modeTemplateHint: 'Dùng được bất cứ lúc nào, dù trong hay ngoài khung 24 giờ.',
  modeText: 'Văn bản tự do',
  modeTextHint: 'Chỉ trong khung chăm sóc khách hàng 24 giờ; ngoài khung đó, bỏ qua hoặc gửi một mẫu thay thế.',
  template: 'Mẫu',
  templatePick: 'Chọn một mẫu đã được phê duyệt',
  templateOption: '{name} · {category} · {language}',
  templateOther: '{name} · {status}',
  synced: 'Đồng bộ lúc {time}',
  syncedDay: 'Đồng bộ {date}, {time}',
  noTemplates: 'Chưa có mẫu nào được phê duyệt. Mẫu đến từ WhatsApp Manager và được đồng bộ trong Nhà cung cấp gửi thư › WhatsApp.',
  variables: 'Biến',
  noVariables: 'Mẫu này không có biến.',
  varSourceFor: "Nguồn điền vào '{{'{key}'}}'",
  textFor: "Văn bản cho '{{'{key}'}}'",
  fallbackFor: "Giá trị dự phòng cho '{{'{key}'}}'",
  fixedText: 'Văn bản cố định',
  fieldOption: 'Trường liên hệ · {label}',
  fallbackPlaceholder: 'Giá trị dự phòng khi trường trống',
  text: 'Nội dung',
  textPlaceholder: 'Cảm ơn bạn đã nhắn tin. Cửa mở lúc 8:30.',
  outsideWindow: 'Ngoài khung 24 giờ',
  outsideWindowSkip: 'Bỏ qua tin nhắn',
  outsideWindowTemplate: 'Gửi một mẫu thay thế',
  fallbackTemplate: 'Mẫu gửi khi ngoài khung 24 giờ',
  consent: 'Đồng ý',
  pickList: 'Chọn một danh sách',
  consentNote: 'Chỉ gửi cho liên hệ đã đồng ý nhận WhatsApp trên {list}: hiện có {n}. Những người khác bị bỏ qua. Không thể bỏ quy tắc này.',
  consentPick: 'Chỉ gửi cho liên hệ đã đồng ý nhận WhatsApp trên danh sách bạn chọn. Những người khác bị bỏ qua. Không thể bỏ quy tắc này.',
  outside: 'Ngoài khung giờ gửi',
  outsideWait: 'Chờ đến khung giờ gửi (mặc định)',
  outsideSkip: 'Bỏ qua tin nhắn',
  outsideHint: '{window} theo múi giờ của từng liên hệ.',
  quietOff: 'Giờ yên lặng đang tắt: tin nhắn được gửi ngay khi liên hệ đến bước này.',
  preview: 'Xem trước',
  previewAs: 'Xem trước · như {name} thấy',
  previewEmpty: 'Hãy chọn một mẫu và một danh sách để xem tin nhắn như một người trong danh sách đó sẽ đọc.',
  previewText: 'Văn bản tự do được gửi đúng như đã viết, cho những người đang trong khung 24 giờ.',
  previewCaption: 'Mẫu {category} · do {business} gửi',
  cost: 'Chi phí ước tính',
  costLine: '{amount} cho {n, plural, other {# tin nhắn}}',
  costNote: 'Theo bảng giá của Meta ngày {date}. Meta tính phí mỗi tin nhắn mẫu gửi thành công, theo danh mục và quốc gia của người nhận.',
  costNoteText: 'Văn bản tự do trong khung 24 giờ không mất phí. Mẫu gửi ngoài khung đó bị Meta tính phí, theo danh mục và quốc gia của người nhận.',
  windowTitle: 'Khung chăm sóc khách hàng',
  windowText:
    'Văn bản tự do chỉ đến được những người đã nhắn tin trong 24 giờ qua, như {name} cho đến {until}. Mẫu thì dùng được cho mọi người đã đồng ý nhận WhatsApp.',
  windowTextNobody:
    'Văn bản tự do chỉ đến được những người đã nhắn tin trong 24 giờ qua; chưa ai trong danh sách làm vậy. Mẫu thì dùng được cho mọi người đã đồng ý nhận WhatsApp.',
  notConnected: 'WhatsApp chưa được kết nối: bước này bỏ qua mọi người cho đến khi được kết nối.',
  // The canvas node.
  nodeConsent: '{n} người đồng ý nhận WhatsApp',
  nodeCost: 'ước tính {cost} · bảng giá của Meta',
  nodeWindow: 'Chỉ trong khung 24 giờ',
  // Step stats.
  statsTitle: 'WhatsApp · 30 ngày',
  sSent: 'Đã gửi',
  sDelivered: 'Gửi thành công',
  sRead: 'Đã đọc',
  sFailed: 'Thất bại',
  sSkipped: 'Bị bỏ qua',
  sCost: 'Chi phí',
  noMessages: 'Chưa có tin nhắn nào. Bước này nhắn tin cho liên hệ đã đồng ý nhận WhatsApp khi họ đến bước.',
  waiting: '{count, plural, other {# tin nhắn}} đang chờ gửi đi.',
  skippedNote:
    'Bị bỏ qua: liên hệ chưa đồng ý nhận WhatsApp hoặc không có số điện thoại, số bị chặn gửi, văn bản tự do ngoài khung 24 giờ, hoặc ngoài khung giờ gửi khi bước được đặt là bỏ qua.',
})
