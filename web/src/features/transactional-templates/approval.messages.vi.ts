import { translate } from '@/i18n'
import { m } from './approval.messages'

export default translate(m, {
  ruleTitle: 'Xuất bản mẫu giao dịch cũng cần phê duyệt',
  ruleDetail: 'Áp dụng cho mọi người có thể xuất bản mẫu giao dịch. Khi tắt: việc xuất bản chỉ được ghi vào nhật ký, và API gửi phiên bản mới ngay.',
  cardTitle: 'Mẫu giao dịch',
  cardCaption: 'Bản nháp mà phê duyệt này cho phép tác giả xuất bản',
  version: 'Phiên bản',
  versionValue: 'v{version} của {key}',
  versionReplaces: 'v{version} của {key}, thay cho v{live} đang chạy',
  category: 'Danh mục',
  subject: 'Tiêu đề',
  variables: 'Biến',
  content: 'Nội dung',
  noContentChange: 'Nội dung không thay đổi',
  breaking: 'Thay đổi gây lỗi',
  testRenders: 'Kết xuất thử',
  renders: 'Kết xuất được',
  fails: 'Lỗi',
  checksLine: '{passed, plural, other {# mục đạt}}{warnings, plural, =0 {} other {, # cảnh báo}}',
  published: 'Đã xuất bản thành v{version}',
  decisionNote: 'Phê duyệt áp dụng cho đúng bản nháp này. Nếu {name} sửa nó trước khi xuất bản, việc xuất bản sẽ cần phê duyệt mới.',
  rejectDetail:
    '{name} sẽ được thông báo và bản nháp không được xuất bản. API tiếp tục gửi phiên bản đang chạy. Hãy từ chối khi nó hoàn toàn không nên được gửi đi; nếu chỉ cần sửa, hãy yêu cầu thay đổi.',
  changesDetail: 'Yêu cầu được trả lại cho {name} kèm nhận xét của bạn. Bản nháp chưa được xuất bản cho đến khi họ yêu cầu lại.',
})
