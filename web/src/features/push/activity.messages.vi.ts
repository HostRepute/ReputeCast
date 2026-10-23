import { translate } from '@/i18n'
import { m } from './activity.messages'

/**
 * Vietnamese for notifications and WhatsApp messages on a contact's
 * timeline, worded like the SMS lines (features/sms/activity.messages.vi.ts):
 * the status heads the line. A WhatsApp message is "tin WhatsApp", and a
 * free-text reply in place of a template reads "dạng trả lời tự do".
 */
export default translate(m, {
  pushSent: '<b>Đã gửi thông báo</b> “{title}”',
  pushShown: '<b>Đã hiển thị thông báo</b> “{title}”',
  pushClicked: '<b>Đã nhấp vào thông báo</b> “{title}”',
  pushFailed: '<b>Lỗi gửi thông báo</b> “{title}”',
  pushSkipped: '<b>Đã bỏ qua thông báo</b> “{title}”',
  pushHeld: '<b>Đang tạm giữ thông báo</b> “{title}”',
  pushSubscribed: '<b>Đã bật thông báo</b> trên {site}',
  pushUnsubscribed: '<b>Đã tắt thông báo</b> trên {site}',
  from: 'từ {name}',
  linkedMember: 'liên kết với tư cách thành viên đã đăng nhập',
  linkedLink: 'liên kết qua đường dẫn có mã nhận diện',
  linkedAccount: 'từ trang tài khoản',
  linkedPreferences: 'từ trung tâm tùy chọn',
  removedUnsubscribed: 'trong trình duyệt',
  removedTurnedOff: 'từ trang tài khoản',
  removedGone: 'dịch vụ đẩy trả về 410 Gone',
  removedRotated: 'khóa đã được xoay vòng',
  prompt: 'lời nhắc “{text}”',
  reasonCap: 'vượt giới hạn tần suất',
  reasonQuiet: 'ngoài khung giờ gửi',
  reasonWorker: 'trang thiếu service worker',
  reasonOther: '{reason}',

  waSent: '<b>Đã gửi tin WhatsApp</b> {template}',
  waDelivered: '<b>Đã gửi thành công tin WhatsApp</b> {template}',
  waDeliveredRead: '<b>Đã đọc tin WhatsApp</b> {template}',
  waUndelivered: '<b>Không gửi được tin WhatsApp</b> {template}',
  waFailed: '<b>Lỗi gửi tin WhatsApp</b> {template}',
  waSkipped: '<b>Đã bỏ qua tin WhatsApp</b> {template}',
  waHeld: '<b>Đang tạm giữ tin WhatsApp</b> {template}',
  waCancelled: '<b>Đã hủy tin WhatsApp</b> {template}',
  waText: 'dạng trả lời tự do',
  waReply: '<b>Đã nhắn trên WhatsApp</b> “{text}”',
  waButton: '<b>Đã nhấn</b> “{text}” trên WhatsApp',
  waOptOut: '<b>Đã hủy nhận WhatsApp</b> bằng {keyword}',
  waOptIn: '<b>Đã đăng ký nhận WhatsApp trở lại</b> bằng {keyword}',
  waHelp: '<b>Đã nhờ trợ giúp trên WhatsApp</b> bằng {keyword}',
  waKeyword: '<b>Đã nhắn {keyword}</b> trên WhatsApp',
  waConsentSubscribed: '<b>Đồng ý nhận WhatsApp</b> trên {list}',
  waConsentWithdrawn: '<b>Đã rút lại đồng ý nhận WhatsApp</b> trên {list}',
  waConsentOther: '<b>Đã thay đổi đồng ý nhận WhatsApp</b> trên {list}',
  byActor: 'bởi {name}',
  consentVersion: 'nội dung đồng ý v{version}',
  ip: 'IP {ip}',
})
