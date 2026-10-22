import { translate } from '@/i18n'
import { ma, mr, mw } from './elsewhere.messages'

/**
 * Vietnamese for transactional mail in other screens: the contact profile,
 * the Overview card and Instance admin's SMTP relay (docs/i18n/glossary-vi.md).
 */
const contact = translate(ma, {
  row: '<b>{status}</b> <link>{name}</link> <badge>Giao dịch</badge>',
  rowVia: '<b>{status}</b> <link>{name}</link> <badge>Giao dịch</badge> qua <badge>{provider}</badge>',
  afterCall: '{time} sau lệnh gọi',
  contentVisible: 'xem được nội dung với quyền transactional:content',
  contentHidden: 'cần quyền transactional:content để xem nội dung',
  notifications: 'Thông báo từ {product}',
  categoryOn: '{name} bật',
  categoryOff: '{name} tắt',
  notificationsLine: 'Danh mục Tùy chọn mà liên hệ có thể tắt trong trung tâm tùy chọn · email {essential} luôn được gửi đến',
  notificationsLineNoEssential: 'Danh mục Tùy chọn mà liên hệ có thể tắt trong trung tâm tùy chọn',
})

const card = translate(mw, {
  title: 'Giao dịch · 7 ngày qua',
  label: 'Giao dịch, 7 ngày qua',
  line: 'Thư sản phẩm của {product} · {route}',
  route: '{primary}, sau đó {fallback}',
  noRoute: 'chưa có tuyến',
  sent: 'Đã gửi',
  delivered: 'Gửi thành công',
  median: 'Thời gian gửi trung vị',
  rejected: 'Bị từ chối',
  noAlerts: 'Không có cảnh báo nào',
  alerts: '{count, plural, other {# cảnh báo đang hoạt động}}',
  failover: 'Đang chuyển dự phòng sang {provider}',
  open: 'Mở Giao dịch',
})

const relay = translate(mr, {
  title: 'SMTP relay',
  caption: 'Dành cho ứng dụng gửi thư giao dịch qua SMTP thay vì HTTP API',
  turnedOn: '{caption} · {name} đã bật ngày {date}',
  turnedOnNoName: '{caption} · đã bật ngày {date}',
  on: 'Bật',
  off: 'Tắt',
  unreachable: 'Không truy cập được',
  hostname: 'Tên máy chủ',
  ports: 'Cổng',
  starttls: 'STARTTLS',
  tls: 'TLS',
  starttlsPort: 'Cổng STARTTLS',
  tlsPort: 'Cổng TLS',
  certificate: 'Chứng chỉ',
  acme: "Let's Encrypt",
  acmeRenews: "Let's Encrypt · gia hạn {date}",
  files: 'Tệp chứng chỉ và khóa',
  certFile: 'Chuỗi chứng chỉ (PEM)',
  keyFile: 'Khóa riêng (PEM)',
  keyHint: 'Chỉ ghi: khóa được niêm phong và không bao giờ hiển thị lại. Để trống cả hai để giữ các tệp đang dùng.',
  help: 'Không gian làm việc tạo thông tin xác thực SMTP trong Giao dịch › Thiết lập; mỗi thông tin xác thực là một khóa chỉ gửi. Thư đến đây đi qua tuyến Giao dịch của không gian làm việc.',
  offBody:
    'Bật relay để các ứng dụng chỉ dùng được SMTP gửi qua tuyến Giao dịch của không gian làm việc. ReputeCast lắng nghe trên các cổng bên dưới và kiểm tra DNS, các cổng và chứng chỉ.',
  checks: 'Kiểm tra',
  checkedAt: 'Đã kiểm tra {date}',
  notChecked: 'Chưa kiểm tra: hãy chạy kiểm tra khi DNS đã trỏ đến đây.',
  pass: 'Đạt',
  fail: 'Không đạt',
  runChecks: 'Chạy kiểm tra',
  turnOn: 'Bật',
  turnOff: 'Tắt',
  save: 'Lưu',
  saved: 'Đã lưu SMTP relay',
  turnedOnToast: 'Đã bật SMTP relay',
  turnedOffToast: 'Đã tắt SMTP relay',
  turnedOffDetail: 'Ứng dụng gửi qua SMTP sẽ bị từ chối kết nối; HTTP API vẫn hoạt động.',
  renewFailed: 'Chứng chỉ không gia hạn được: {error}',
  error: 'Không tải được SMTP relay',
  retry: 'Thử lại',
})

export default { ...contact, ...card, ...relay }
