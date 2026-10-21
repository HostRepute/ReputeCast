import { translate } from '@/i18n'
import { m } from './activity.messages'

/**
 * A text's status heads its line on the timeline, before the campaign's
 * name: "Đã gửi tin nhắn Summit reminder". "Outside quiet hours" means
 * outside the window texts may go out in: "ngoài khung giờ gửi".
 */
export default translate(m, {
  sent: 'Đã gửi tin nhắn',
  delivered: 'Đã gửi thành công tin nhắn',
  undelivered: 'Không gửi được tin nhắn',
  failed: 'Lỗi gửi tin nhắn',
  skipped: 'Đã bỏ qua tin nhắn',
  held: 'Đang tạm giữ tin nhắn',
  cancelled: 'Đã hủy tin nhắn',
  from: 'từ {sender}',
  clicked: 'Đã nhấp liên kết trong tin nhắn',
  linkPreview: 'bản xem trước liên kết, không phải người đọc',
  texted: 'Đã nhắn {keyword}',
  optOutNote: 'đã rút lại đồng ý nhận SMS trên mọi danh sách',
  optInNote: 'đã đồng ý nhận SMS trở lại',
  helpNote: 'đã gửi tin trả lời trợ giúp',
  yesNote: 'đã xác nhận đồng ý',
  replied: 'Đã trả lời “{text}”',
  replyNote: 'lưu trên dòng thời gian; không có hộp thư chung',
  consentSubscribed: 'Đồng ý nhận SMS trên {list}',
  consentPending: 'Đã hỏi đồng ý nhận SMS trên {list}',
  consentConfirmed: 'Đã xác nhận đồng ý nhận SMS trên {list}',
  consentOptedOut: 'Đã rút lại đồng ý nhận SMS trên {list}',
  consentOptedIn: 'Đã đồng ý nhận SMS trở lại trên {list}',
  consentUnsubscribed: 'Đã gỡ đồng ý nhận SMS trên {list}',
  consentImported: 'Đã nhập đồng ý nhận SMS trên {list}',
  confirmationSent: 'Đã gửi tin xác nhận SMS cho {list}',
  confirmationFailed: 'Gửi tin xác nhận SMS cho {list} thất bại',
  everyList: 'mọi danh sách',
  version: 'nội dung đồng ý v{version}',
  byActor: 'bởi {name}',
  ip: 'IP {ip}',
  // Why a text was skipped, held or cancelled.
  rNo_consent: 'chưa đồng ý nhận SMS',
  rNo_phone: 'không có số điện thoại',
  rSuppressed: 'nằm trong danh sách chặn gửi SMS',
  rRecent: 'vừa nhận một tin nhắn gần đây',
  rQuiet_hours: 'ngoài khung giờ gửi',
  rNo_route: 'không có tuyến đến quốc gia đó',
  rSender_pending: 'người gửi chưa được phê duyệt ở đó',
  rPending_sender: 'người gửi chưa được phê duyệt ở đó',
  rProvider_down: 'nhà cung cấp bị gián đoạn',
  rBudget: 'ngân sách của tháng đã hết',
  rCancelled: 'lần gửi đã bị hủy',
})
