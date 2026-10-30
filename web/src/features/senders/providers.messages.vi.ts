import { translate } from '@/i18n'
import { mp } from './providers.messages'

/** Vietnamese for Settings › Providers (docs/i18n/glossary-vi.md). */
export default translate(mp, {
  documentTitle: 'Nhà cung cấp · Cài đặt',
  loadFailed: 'Không tải được cài đặt nhà cung cấp',
  loading: 'Đang tải cài đặt nhà cung cấp…',
  readOnlyText: 'Bạn có thể xem các nhà cung cấp và cài đặt gửi mặc định. Vai trò của bạn không thể thay đổi chúng.',

  connectedProviders: 'Nhà cung cấp đã kết nối',
  noneConnected: 'Chưa có nhà cung cấp gửi thư nào được kết nối.',
  pageNote: 'Tình trạng, định tuyến, khởi động, sự kiện và adapter nằm ở trang Nhà cung cấp gửi thư.',
  addProvider: 'Thêm nhà cung cấp',
  openProviders: 'Mở Nhà cung cấp gửi thư',

  streams: 'Luồng',
  notRouted: 'Chưa được định tuyến',
  connected: 'Ngày kết nối',
  connectedValue: '{date} · {via}',
  costMonth: 'Chi phí · {month}',
  costPlan: 'gói {amount}',
  costMetered: '{amount} theo mức dùng',

  defaultsTitle: 'Cài đặt gửi mặc định cho cả không gian làm việc',
  defaultsLine: 'Chiến dịch mới bắt đầu từ các cài đặt này; mỗi chiến dịch có thể đổi chúng trong Kiểm tra và gửi.',
  bounce: 'Tự động tạm dừng · tỷ lệ trả thư',
  complaint: 'Tự động tạm dừng · tỷ lệ khiếu nại',
  mailboxComplaint: 'Khiếu nại theo nhà cung cấp hộp thư',
  providerError: 'Tỷ lệ lỗi nhà cung cấp · 3 phút',
  pauseScope: 'Phạm vi tạm dừng',
  pauseRoute: 'Chỉ tuyến hoặc nhà cung cấp hộp thư bị ảnh hưởng',
  pauseCampaign: 'Toàn bộ chiến dịch',
  confirmOver: 'Xác nhận khi gửi cho hơn',
  confirmOverValue: '{count} người nhận · nhập số để xác nhận',

  throttlesTitle: 'Giới hạn tốc độ theo nhà cung cấp hộp thư',
  throttlesLine: 'Áp dụng trên mọi tuyến. ReputeCast tự hạ các giới hạn này khi nhà cung cấp hoãn nhận thư.',
  perSecond: '{n} mỗi giây',
  noLimit: 'không giới hạn',
  suppressionSync: 'Chu kỳ đồng bộ danh sách chặn gửi',
  sync15m: 'Mỗi 15 phút, và ngay sau mỗi thư trả lại',
  sync1h: 'Mỗi giờ',
  secrets: 'Khóa bí mật ký webhook',
  rotatedOn: 'Đã xoay vòng ngày {date}',
  notRotated: 'Chưa xoay vòng từ khi kết nối các nhà cung cấp',
  rotate: 'Xoay vòng',
  connectFirst: 'Hãy kết nối một nhà cung cấp trước.',
  discard: 'Hủy thay đổi',
  saveDefaults: 'Lưu cài đặt mặc định',

  needNumber: 'Mỗi ngưỡng cần một con số, ví dụ {example}.',
  throttleFormat: 'Hãy ghi giới hạn tốc độ cho {provider} theo dạng "{example}" hoặc "{noLimit}".',
  saved: 'Đã lưu cài đặt gửi mặc định',
  savedDetail: 'Chiến dịch mới sẽ bắt đầu từ các cài đặt này.',

  rotateSecrets: 'Xoay vòng khóa bí mật',
  rotateTitle: 'Xoay vòng khóa bí mật để ký webhook của mọi nhà cung cấp?',
  rotateBody:
    'ReputeCast tạo khóa bí mật mới cho từng nhà cung cấp và đăng ký khóa đó ở những nơi API của nhà cung cấp cho phép. Với nhà cung cấp có webhook bạn thiết lập thủ công, bạn cũng cần nhập khóa mới ở đó; trước khi làm vậy, sự kiện của họ sẽ bị từ chối.',
  rotating: 'Đang xoay vòng…',
  rotated: '{count, plural, other {Đã xoay vòng # khóa bí mật}}',
  rotatedByHand: 'Hãy cập nhật webhook thủ công trong {providers}.',
})
