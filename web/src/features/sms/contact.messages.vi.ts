import { translate } from '@/i18n'
import { m } from './contact.messages'

export default translate(m, {
  title: 'Đồng ý theo danh sách · {name}',
  lead: 'Email và SMS được tính riêng. Mỗi thay đổi đều giữ bằng chứng của nó.',
  list: 'Danh sách',
  subscribed: 'Đã đăng ký',
  confirmed: 'Đã xác nhận',
  pending: 'Đang chờ',
  unsubscribed: 'Đã hủy đăng ký',
  optedOut: 'Đã hủy nhận',
  notSubscribed: 'Chưa đăng ký',
  noPhone: 'Chưa có số điện thoại.',
  addPhone: 'Thêm số điện thoại',
  changePhone: 'Đổi số điện thoại',
  phone: 'Số điện thoại',
  phoneHint: 'Theo định dạng quốc tế, ví dụ +1 312 555 0123.',
  phoneText: 'Quốc gia được xác định từ số điện thoại; loại thuê bao và múi giờ cũng được đọc từ đó. Đồng ý nhận SMS vẫn tính theo từng danh sách.',
  phoneSaved: 'Đã lưu số điện thoại',
  save: 'Lưu',
  cancel: 'Hủy',
  lineMobile: 'di động',
  lineLandline: 'cố định',
  lineTollFree: 'miễn cước',
  lineVoip: 'VoIP',
  lineUnknown: 'không rõ loại thuê bao',
  suppressed:
    'Số này nằm trong danh sách chặn gửi SMS ({reason, select, opted_out {đã hủy nhận} landline {số cố định} not_in_service {ngừng hoạt động} invalid {số không hợp lệ} filtered {bị nhà mạng lọc} provider_list {danh sách hủy nhận của nhà cung cấp} manual {thêm thủ công} other {{reasonText}}}, {date}).',
  byKeyword: 'bằng cách nhắn {keyword}',
  consentVersion: 'nội dung đồng ý v{version}',
  confirmation:
    'tin xác nhận {status, select, queued {đang chờ gửi} held {bị tạm giữ} skipped {bị bỏ qua} sending {đang gửi} sent {đã gửi} delivered {đã gửi thành công} undelivered {không gửi được} failed {thất bại} cancelled {đã hủy} other {{status}}} {time}',
  withdraw: 'Rút lại đồng ý nhận SMS',
  record: 'Ghi nhận đồng ý nhận SMS',
  consentSaved: 'Đã lưu đồng ý nhận SMS trên {list}',
  foot: 'STOP rút lại đồng ý nhận SMS trên mọi danh sách và không bao giờ ảnh hưởng đến email; hủy đăng ký email cũng không bao giờ ảnh hưởng đến tin nhắn.',
})
