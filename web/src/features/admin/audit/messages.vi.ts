import { translate } from '@/i18n'
import { m } from './messages'

export default translate(m, {
  documentTitle: 'Nhật ký kiểm tra',
  eventsCaption: 'Các sự kiện bị từ chối được tô nổi bật. Lần đăng nhập bị từ chối có kèm mã tham chiếu mà người dùng thấy trên trang lỗi.',
  tableCaption: 'Các mục nhật ký kiểm tra của hệ thống',
  noMatch: 'Không có sự kiện nào khớp với các bộ lọc này. Hãy mở rộng khoảng thời gian hoặc bỏ bộ lọc người thực hiện.',

  instanceLevel: 'Cấp hệ thống',
  instanceLevelCaption: 'Đăng nhập, SSO, chính sách, người dùng và sự kiện của mọi không gian làm việc',
  retained: 'Lưu giữ {days, plural, other {# ngày}} · không thể sửa đổi',
  retention: 'Thời gian lưu giữ',
  exportDownloaded: 'Đã tải xuống bản xuất {format}',
  exportFailed: 'Xuất thất bại',
  exportTimedOut: 'Bản xuất {format} đã hết thời gian chờ sau {seconds} giây. Hãy thu hẹp khoảng thời gian hoặc dùng endpoint truyền luồng để truy cập hàng loạt.',
  exportError: 'Xuất {format} thất bại: {error} Hãy thu hẹp khoảng thời gian hoặc thử lại.',

  searchPlaceholder: 'Tìm tài khoản, đối tượng hoặc mã tham chiếu',
  filterWorkspace: 'Lọc theo không gian làm việc',
  allWorkspaces: 'Mọi không gian làm việc',
  instanceOnly: 'Chỉ cấp hệ thống',

  cardTitle: 'Thời gian lưu giữ và truyền sự kiện',
  kvRetention: 'Thời gian lưu giữ',
  retentionValue:
    '{days, plural, other {# ngày}}{isDefault, select, yes { (mặc định)} other {}} · các mục cấp hệ thống cũ hơn sẽ bị xóa ở lần dọn dẹp hằng giờ; mỗi không gian làm việc có thời gian lưu giữ riêng',
  kvStreaming: 'Truyền sự kiện',
  streamingValue:
    'Sự kiện của không gian làm việc → {workspace, select, yes {{hosts} (ký bằng HMAC)} other {không truyền}} · sự kiện hệ thống → {instance, select, on {{host} (ký bằng HMAC)} failing {{host} (ký bằng HMAC), đang lỗi} other {không truyền}}',
  kvExport: 'Xuất dữ liệu',
  exportValue: 'CSV hoặc JSON, theo bộ lọc hoặc toàn bộ',
  changeRetention: 'Đổi thời gian lưu giữ',
  streamButtonOn: 'Luồng sự kiện hệ thống',
  streamButtonOff: 'Truyền sự kiện hệ thống',

  retentionTitle: 'Thời gian lưu giữ nhật ký kiểm tra của hệ thống',
  retentionBody:
    'Thời gian lưu giữ các mục cấp hệ thống (đăng nhập, SSO, chính sách và người dùng). Mỗi không gian làm việc có thời gian lưu giữ riêng trong phần cài đặt nhật ký kiểm tra của mình.',
  pickAnother: 'Hãy chọn thời gian lưu giữ khác để lưu',
  retentionSaved: 'Các mục cấp hệ thống được lưu giữ {days, plural, other {# ngày}}',
  saveRetention: 'Lưu thời gian lưu giữ',
  keepFor: 'Lưu giữ các mục cấp hệ thống trong',
  retentionOption: '{days, plural, other {# ngày}}{isDefault, select, yes { (mặc định)} other {}}',
  retentionKept: '{count, plural, other {Hiện đang lưu giữ {total} mục cấp hệ thống.}} Các mục không thể sửa đổi cho đến khi hết thời gian lưu giữ.',
  shorter: 'Ngắn hơn',
  shorterNote: 'Các mục cũ hơn {days, plural, other {# ngày}} sẽ bị xóa ở lần dọn dẹp hằng giờ tiếp theo. Hãy xuất chúng trước nếu bạn cần.',

  streamTitle: 'Truyền sự kiện hệ thống',
  streamBody:
    'Gửi các lần đăng nhập và thay đổi về SSO, chính sách, người dùng đến SIEM hoặc kho dữ liệu của bạn ngay khi chúng xảy ra, theo từng lô có chữ ký. Các không gian làm việc tự truyền sự kiện của mình từ phần cài đặt nhật ký kiểm tra.',
  streamOn: 'Sự kiện hệ thống đang được truyền',
  streamOff: 'Sự kiện hệ thống không còn được truyền',
  secretInput: 'Khóa bí mật để ký của luồng nhật ký kiểm tra hệ thống',
  streamToggle: 'Truyền các mục cấp hệ thống đến một webhook',
  workspaceStreams: 'Đang tự truyền sự kiện của mình: {list}.',
  workspaceStream: '{name} → {host}{failing, select, yes { (đang lỗi)} other {}}',
})
