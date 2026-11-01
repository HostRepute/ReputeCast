import { translate } from '@/i18n'
import { mf } from './followup.messages'

export default translate(mf, {
  menu: 'Gửi tiếp',
  nonOpeners: 'Gửi lại cho người chưa mở…',
  clickers: 'Gửi cho người đã nhấp…',
  titleNonOpeners: 'Gửi lại cho người chưa mở',
  titleClickers: 'Gửi cho người đã nhấp',
  bodyNonOpeners: '<b>{count} liên hệ</b> đã nhận "{name}" nhưng chưa mở.',
  machineOnly: '{count} liên hệ chỉ được mở qua một proxy bảo vệ quyền riêng tư như Apple Mail Privacy Protection: họ được tính là chưa rõ và bị loại ra.',
  bodyClickers: '<b>{count} người</b> đã nhấp một liên kết trong "{name}". Lượt nhấp của trình quét liên kết không được tính.',
  nobody: 'Hiện chưa có ai trong nhóm này nên lần gửi tiếp sẽ chưa đến được ai. Danh sách người nhận được cập nhật cho đến khi gửi đi.',
  who: 'Gửi cho',
  whoNonOpeners: 'Người chưa mở thư',
  whoClickers: 'Người đã nhấp',
  subject: 'Tiêu đề mới',
  subjectNonOpeners: 'Có thể bạn đã bỏ lỡ: {subject}',
  subjectClickers: 'Thêm về: {subject}',
  send: 'Gửi',
  after: '{days} ngày sau thư gốc ({when})',
  later: 'Chọn ở bước Xem lại và gửi',
  note: 'Tạo một bản nháp với cùng nội dung. Danh sách chặn gửi, hủy đăng ký và giới hạn email marketing vẫn áp dụng như thường lệ.',
  create: 'Tạo lần gửi tiếp',
  creating: 'Đang tạo…',
  created: 'Đã tạo lần gửi tiếp "{name}"',
  loading: 'Đang đếm số người sẽ nhận…',
  subjectRequired: 'Hãy đặt tiêu đề cho lần gửi tiếp.',
})
