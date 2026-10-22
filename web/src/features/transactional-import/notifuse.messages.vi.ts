import { translate } from '@/i18n'
import { m } from './notifuse.messages'

export default translate(m, {
  sourceWhat: 'Khóa API · liên hệ, danh sách, mẫu',
  templatesOption: 'Nhập mẫu',
  templatesDetail:
    '{transactional, plural, =0 {không có mẫu giao dịch} other {# mẫu giao dịch được nhập thành bản nháp}}, {saved, plural, =0 {không có mẫu khác} other {# mẫu khác vào mẫu đã lưu}}',
  templatesSkipped:
    '{skipped, plural, =1 {Email xác nhận hai bước hoặc email hủy đăng ký của Notifuse không được nhập: ReputeCast tự gửi email của mình.} other {# email xác nhận hai bước và email hủy đăng ký của Notifuse không được nhập: ReputeCast tự gửi email của mình.}}',
  doneTemplates: '{transactional, plural, other {# mẫu giao dịch}} đã được nhập thành bản nháp và {saved, plural, other {# mẫu}} đã được thêm vào mẫu đã lưu.',
  doneTemplatesManual: '{manual, plural, other {# mẫu trong số đó cần sửa thủ công}} trước khi dùng được.',
  doneTemplatesLink: 'Mở mẫu giao dịch',
})
