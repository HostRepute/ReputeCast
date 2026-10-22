import { translate } from '@/i18n'
import { m } from './campaigns.messages'

/** Vietnamese for push and WhatsApp campaigns in the campaigns list and New campaign (docs/i18n/glossary-vi.md › Campaigns). */
export default translate(m, {
  newPush: 'Thông báo đẩy',
  newPushBody: 'Một thông báo đến trình duyệt của người đọc trên các trang dùng thông báo đẩy của bạn.',
  newPushHint: 'Tiêu đề, nội dung và liên kết, kèm bản xem trước cho từng nền tảng.',
  newWhatsApp: 'Tin nhắn WhatsApp',
  newWhatsAppBody: 'Một mẫu đã được phê duyệt, gửi đến những người đồng ý nhận WhatsApp.',
  newWhatsAppHint: 'Biến lấy từ trường của liên hệ; chi phí hiện trước khi bạn gửi.',
  untitledPush: 'Chiến dịch thông báo đẩy chưa đặt tên',
  untitledWhatsApp: 'Chiến dịch WhatsApp chưa đặt tên',
  noOpens: 'Thông báo và tin nhắn WhatsApp không có lượt mở',
  shownTitle: 'Đã hiển thị, theo báo cáo của service worker',
  local: '{date} · {time} theo múi giờ của từng người nhận',
  notSite: 'Chưa có trang',
  siteSubscribers: 'Người đăng ký của trang',
  duplicated: 'Đã nhân bản thành “{name}”',
  duplicateFailed: 'Không nhân bản được',
})
