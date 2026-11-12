import { translate } from '@/i18n'
import { mc } from './messages'

/** Vietnamese for the composer's Discount code block (M59). */
export default translate(mc, {
  block: 'Mã giảm giá',
  name: 'Mã giảm giá',
  defaultLabel: 'Mã của bạn',
  defaultButton: 'Mua ngay',
  canvasPick: 'Chọn một kho mã giảm giá trong cài đặt của khối.',
  canvasUnknown: 'Không có kho mã giảm giá nào tên {slug}. Chọn kho khác trong cài đặt của khối.',
  oneCode: 'Mỗi người đọc một mã, được cấp khi email được gửi.',
  pool: 'Kho mã',
  poolOption: '{name} · {free} còn trống',
  poolNone: 'Chọn một kho mã',
  poolStore: '{platform} · {store}',
  poolUpload: 'Mã đã tải lên',
  poolOnePer: 'một mã cho mỗi liên hệ',
  poolLiquid: 'trong Liquid <mono>{token}</mono>',
  noPools: 'Chưa có kho mã nào. <a>Tạo một kho</a> trong Chiến dịch › Mã giảm giá.',
  label: 'Nhãn',
  button: 'Nút',
  buttonOn: 'Hiện nút',
  buttonOpens: 'Mở {store} với mã đã được áp dụng: <mono>{token}</mono>',
  buttonApply: 'Mở địa chỉ áp dụng kèm mã: <mono>{token}</mono>',
  buttonNoUrl: '{name} không có địa chỉ áp dụng, nên khối không hiện nút. Thêm địa chỉ trên trang của kho mã.',
  manage: 'Quản lý mã giảm giá',
  manageNote: 'Các kho mã, số liệu và nơi dùng từng kho, trong Chiến dịch › Mã giảm giá.',
  pass: 'Đạt',
  passNote: 'Kiểm tra và gửi xác nhận có đủ mã cho mọi người nhận.',
  warn: 'Cảnh báo',
  archivedNote: '{name} đã được lưu trữ: người đọc không nhận được mã. Hãy chọn kho khác.',
  tokensGroup: 'Mã giảm giá',
})
