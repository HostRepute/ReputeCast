import { translate } from '@/i18n'
import { ms } from './setup.messages'

/** Vietnamese for starting a setup and its summary (docs/i18n/glossary-vi.md). */
export default translate(ms, {
  // ── Starting and loading a setup ─────────────────────────────────────
  backToCatalog: 'Quay lại danh mục',
  unknownAdapter: 'Bản dựng này không có nhà cung cấp này',
  unknownAdapterBody: 'Hãy chọn nhà cung cấp khác trong danh mục, hoặc nhờ quản trị viên hệ thống cài adapter của nó.',
  startFailed: 'Không bắt đầu được thiết lập',
  starting: 'Đang bắt đầu thiết lập…',
  notFound: 'Thiết lập này không tồn tại',
  loadFailed: 'Không tải được thiết lập',
  loading: 'Đang tải thiết lập…',
  cancelledReason: 'Thiết lập này đã bị hủy. Hãy bắt đầu lại từ danh mục.',
  cancelledNote: 'Thiết lập này đã bị hủy. Hãy bắt đầu lại từ danh mục để kết nối nhà cung cấp.',
  savedForLater: 'Đã lưu. Hãy chọn {provider} trong danh mục để làm tiếp.',
  saveFailed: 'Không lưu được thiết lập',

  // ── The summary ──────────────────────────────────────────────────────
  doneBadge: 'Hoàn tất',
  took: '{minutes, plural, other {Thiết lập mất # phút. Mọi thứ bên dưới đã hoạt động.}}',
  tookDomain: '{minutes, plural, other {Thiết lập mất # phút. Mọi thứ bên dưới đã hoạt động, và trang Tổng quan giờ hiển thị tình trạng người gửi của {domain}.}}',
  configuredTitle: 'Những gì đã được cấu hình',
  configuredLine: 'đều hoàn tác được trong Nhà cung cấp gửi thư',
  importNow: 'Nhập ngay',
  recommendedTitle: 'Nên làm trước chiến dịch đầu tiên',
  addProviderTitle: 'Thêm nhà cung cấp khác',
  addProviderLine: 'Một nhà cung cấp thứ hai cho thư Hệ thống, hoặc một máy chủ Postal để chia theo tỷ trọng. Có thể chạy nhiều nhà cung cấp cùng lúc.',
  addProvider: 'Thêm nhà cung cấp khác',
  hostReputeTitle: 'Kết nối HostRepute',
  hostReputeConnected: 'Đã kết nối: 84 danh sách chặn, Google Postmaster và Microsoft SNDS hiển thị trong Độ sẵn sàng gửi.',
  hostReputeOptional: 'Không bắt buộc, miễn phí: 84 danh sách chặn, Google Postmaster và Microsoft SNDS trong Độ sẵn sàng gửi.',
  open: 'Mở',
  connect: 'Kết nối',
  formTitle: 'Tạo biểu mẫu đăng ký',
  formLine: 'Xác nhận kép (double opt-in) được bật theo mặc định; thư xác nhận gửi qua luồng Hệ thống.',
  buildForm: 'Tạo biểu mẫu',
  goToOverview: 'Đến Tổng quan',
})
