import { translate } from '@/i18n'
import { m } from './automation.messages'

/**
 * Vietnamese for the Send push step (docs/i18n/glossary-vi.md › Automations),
 * worded like the Send SMS step: "Outside quiet hours" is outside the
 * window notifications may go out in, "ngoài khung giờ gửi".
 */
export default translate(m, {
  palette: 'Gửi thông báo đẩy',
  name: 'Tên',
  namePlaceholder: 'ví dụ: Trước 15 phút',
  title: 'Tiêu đề',
  message: 'Nội dung',
  counter: '{n}/{max}',
  titleLong: 'Tiêu đề dài hơn sẽ bị cắt trên mọi nền tảng.',
  linkAndButton: 'Liên kết và nút',
  link: 'Liên kết',
  linkPlaceholder: 'news.acme.io/summit/live',
  button: 'Nút',
  buttonPlaceholder: 'ví dụ: Xem phát trực tiếp',
  buttonHint: 'Không bắt buộc. Nút mở cùng liên kết.',
  expiry: 'Thời hạn',
  expiry30: '30 phút',
  expiry240: '4 giờ (mặc định)',
  expiry1440: '1 ngày',
  expiryOther: '{n, plural, other {# phút}}',
  expiryHint: 'Sau thời hạn đó, dịch vụ đẩy sẽ bỏ thông báo: nó chỉ có ích khi đến đúng lúc.',
  outside: 'Ngoài khung giờ gửi',
  outsideWait: 'Chờ đến khung giờ gửi (mặc định)',
  outsideSkip: 'Bỏ qua thông báo',
  outsideHint: '{window} theo múi giờ của từng liên hệ.',
  quietOff: 'Giờ yên lặng đang tắt: thông báo được gửi ngay khi liên hệ đến bước này.',
  who: 'Ai nhận được',
  whoText: 'Gửi đến các lượt đăng ký trên trình duyệt đã liên kết với liên hệ. Bị bỏ qua khi liên hệ không có lượt đăng ký nào đã liên kết.',
  reach: '{linked, plural, other {#}} trong số {contacts, plural, other {# liên hệ}} trong luồng này có lượt đăng ký đã liên kết.',
  reachNone: 'Chưa có liên hệ nào trong luồng này có lượt đăng ký đã liên kết.',
  reachRest: 'Những người còn lại bị bước này bỏ qua, không phải bị luồng bỏ qua: họ tiếp tục sang bước kế tiếp.',
  anonymous: 'Tự động hóa không bao giờ gửi đến người đăng ký ẩn danh.',
  pushOff: 'Thông báo đẩy chưa được thiết lập: bước này bỏ qua mọi người cho đến khi được thiết lập.',
  previews: 'Bản xem trước',
  // The canvas node.
  nodeLinked: '{n, plural, other {# lượt đăng ký đã liên kết}}',
  nodeWait: 'Ngoài khung giờ gửi: chờ',
  nodeSkip: 'Ngoài khung giờ gửi: bỏ qua',
  // Step stats.
  statsTitle: 'Thông báo · 30 ngày',
  sSent: 'Đã gửi',
  sShown: 'Đã hiển thị',
  sClicks: 'Lượt nhấp',
  sSkipped: 'Bị bỏ qua',
  noPush: 'Chưa có thông báo nào. Bước này gửi thông báo cho liên hệ có lượt đăng ký đã liên kết khi họ đến bước.',
  ranDay: 'Đã chạy {date}, {from}–{to}',
  ranDays: 'Đã chạy từ {from} đến {to}',
  skippedNote: 'Bị bỏ qua: liên hệ không có lượt đăng ký đã liên kết, liên hệ ngoài khung giờ gửi khi bước được đặt là bỏ qua, hoặc vượt giới hạn tần suất.',
})
