import { translate } from '@/i18n'
import { m } from './messages'

export default translate(m, {
  dialog: 'Hộp thoại',
  dismiss: 'Đóng thông báo',
  undo: 'Hoàn tác',
  hide: 'Ẩn',
  reference: 'Mã tham chiếu <mono>{reference}</mono>',
  optional: 'không bắt buộc',
  typeToConfirm: 'Hãy nhập {name} để xác nhận.',
  reasonTooShort: 'Hãy nêu lý do dài ít nhất {min} ký tự.',
  bulkPartial: 'Thao tác hàng loạt · một phần',
  bulkDone: 'Thao tác hàng loạt · xong',
  undoFor: 'Hoàn tác cho {count} mục',
  leaveTitle: 'Rời đi mà không lưu?',
  saveAndLeave: 'Lưu và rời đi',
  unsavedChanges: 'Chưa lưu thay đổi',
  stay: 'Ở lại',
  leaveWithoutSaving: 'Rời đi mà không lưu',
  partFailed: 'Không hiển thị được phần này',
  partFailedBody: 'Phần còn lại của trang vẫn dùng được. Hãy thử lại, hoặc tải lại trang nếu lỗi tiếp diễn.',
  loadFailedBody: 'Máy chủ không phản hồi. Không có gì bị mất.',
})
