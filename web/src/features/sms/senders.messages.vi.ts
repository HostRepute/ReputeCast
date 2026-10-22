import { translate } from '@/i18n'
import { m } from './senders.messages'

export default translate(m, {
  title: 'Tên miền liên kết rút gọn',
  sub: 'Mọi liên kết trong tin nhắn được rút gọn ở đây, với một mã riêng cho từng người nhận. Dịch vụ rút gọn công khai như bit.ly bị chặn vì nhà mạng lọc chúng.',
  caption: 'Tên miền liên kết rút gọn',
  domain: 'Tên miền',
  usedBy: 'Dùng cho',
  smsLinks: 'Liên kết trong tin nhắn',
  record: 'Bản ghi',
  status: 'Trạng thái',
  lastCheck: 'Kiểm tra gần nhất',
  cname: 'CNAME {name} → {target}',
  verifiedOn: 'Đã xác minh {date}',
  notVerified: 'Chưa xác minh',
  checkFailed: 'Kiểm tra thất bại',
  never: 'Chưa kiểm tra',
  checkDns: 'Kiểm tra DNS',
  checked: 'Đã kiểm tra DNS',
  manage: 'Định tuyến và quy tắc',
  manageLabel: 'Đổi tên miền liên kết rút gọn trong Định tuyến và quy tắc',
  none: 'Chưa có tên miền liên kết rút gọn. Tin nhắn có liên kết chưa gửi được cho đến khi một tên miền được xác minh.',
  setUp: 'Thiết lập tên miền',
  untilVerified: 'Cho đến khi tên miền được xác minh, tin nhắn có liên kết chưa gửi được. Tin nhắn không có liên kết vẫn gửi được.',
})
