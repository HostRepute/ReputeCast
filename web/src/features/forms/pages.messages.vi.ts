import { translate } from '@/i18n'
import { m } from './pages.messages'

/**
 * Vietnamese for the built-in public pages' editors (docs/i18n/glossary-vi.md).
 * The words match what subscribers read on those pages
 * (internal/pagelabels): an issue is "số", the archive is "trang lưu trữ".
 */
export default translate(m, {
  // ── The page being edited ────────────────────────────────────────────
  loadFailed:
    '{page, select, subscribe {Không tải được trang đăng ký} preferences {Không tải được trung tâm tùy chọn} unsubscribe {Không tải được trang hủy đăng ký} other {Không tải được trang lưu trữ}}',
  keepsWorking: 'Không có gì bị thay đổi; trang vẫn hoạt động với người đăng ký.',
  loading:
    '{page, select, subscribe {Đang tải trang đăng ký} preferences {Đang tải trung tâm tùy chọn} unsubscribe {Đang tải trang hủy đăng ký} other {Đang tải trang lưu trữ}}',
  pageName:
    '{page, select, subscribe {trang đăng ký} confirm {trang xác nhận} preferences {trung tâm tùy chọn} unsubscribe {trang hủy đăng ký} other {trang lưu trữ}}',
  previewOf:
    '{page, select, subscribe {Bản xem trước trang đăng ký} confirm {Bản xem trước trang xác nhận} preferences {Bản xem trước trung tâm tùy chọn} unsubscribe {Bản xem trước trang hủy đăng ký} other {Bản xem trước trang lưu trữ}}',
  saved: 'Đã lưu',
  savedBody: 'Người đăng ký thấy thay đổi ở lần truy cập tiếp theo.',
  saveFailed: 'Không lưu được trang',
  saving: 'Đang lưu…',
  pageOptions: 'Tùy chọn trang',
  unsaved: 'Bạn đã thay đổi các tùy chọn của trang này kể từ lần lưu.',
  saveLeave: 'Lưu và rời đi',
  previewFailed: 'Không hiển thị được bản xem trước',

  // ── Shared sections ──────────────────────────────────────────────────
  pageAddress: 'Địa chỉ trang',
  urlOf: '{title} (URL)',
  previewAs: 'Xem trước với tư cách',
  previewAsLabel: 'Xem trước với tư cách người đăng ký',
  previewAsPlaceholder: 'Người đăng ký đã xác nhận gần nhất',
  previewAsNote: 'Bản xem trước hiển thị danh sách và cài đặt của người đăng ký này. Không có gì thay đổi với họ.',
  listLines: 'Danh sách · dòng bên dưới mỗi danh sách',

  // ── Subscribe and confirmation ───────────────────────────────────────
  page: 'Trang',
  subscribe: 'Đăng ký',
  confirmation: 'Xác nhận',
  form: 'Biểu mẫu',
  formOnPage: 'Biểu mẫu trên trang đăng ký',
  builtinForm: 'Biểu mẫu có sẵn · email, tên, danh sách công khai',
  draftForm: '{name} · bản nháp',
  formNote: 'Trang hiển thị các trường, danh sách và nội dung đồng ý của biểu mẫu này. Nội dung bên dưới thay cho đề mục của biểu mẫu.',
  copy: 'Nội dung',
  heading: 'Đề mục',
  description: 'Mô tả',
  underButton: 'Bên dưới nút',
  subscribersHint: "'{subscribers}' hiển thị số người trong danh sách, ví dụ {example}.",
  afterClick: 'Sau khi nhấp xác nhận · mỗi danh sách mang lại gì',
  nextIssueHint: 'Số tiếp theo đã lên lịch được thêm vào sau dòng này.',
  confirmationPage: 'Trang xác nhận',
  inboxTip:
    '{named, select, yes {Nhắc người đăng ký mới thêm {sender} vào danh bạ} other {Nhắc người đăng ký mới thêm địa chỉ gửi của bạn vào danh bạ}}',
  searchEngines: 'Công cụ tìm kiếm',
  indexSubscribe: 'Cho phép công cụ tìm kiếm liệt kê trang đăng ký',

  // ── Preference center ────────────────────────────────────────────────
  prefsNote: 'Mọi email đều có liên kết đến trang riêng của từng người đăng ký. Địa chỉ này gửi liên kết đó qua email cho bất kỳ ai yêu cầu.',
  listsOffered: 'Danh sách được hiển thị',
  alwaysSee: 'Người đăng ký luôn thấy các danh sách họ đang tham gia.',
  choices: 'Lựa chọn',
  howOften: 'Tần suất · mọi số, tổng hợp hằng tuần, tuyển chọn hằng tháng',
  needBreak: 'Cần nghỉ một thời gian · tạm dừng 30 hoặc 90 ngày',
  firstName: 'Tên',

  // ── Unsubscribe page ─────────────────────────────────────────────────
  fewerInstead: 'Ít email hơn thay vì rời đi',
  offerBestOf: 'Đề nghị bản tuyển chọn hằng tháng trước khi họ rời đi',
  text: 'Văn bản',
  button: 'Nút',
  reason: 'Lý do',
  askWhy: 'Hỏi vì sao họ rời đi · họ có thể bỏ qua',
  answersShow: 'Câu trả lời hiển thị trong báo cáo của từng chiến dịch.',
  oneClick: 'Hủy đăng ký một lần nhấp',
  oneClickBody:
    'Mọi email đều có List-Unsubscribe và List-Unsubscribe-Post (RFC 8058). Ứng dụng email hủy đăng ký chỉ với một lần nhấp, không qua trang này, và có hiệu lực ngay.',
  lastDays: '{days} ngày qua',
  leftList: 'Rời một danh sách',
  withOneClick: 'Bằng một lần nhấp',
  choseFewer: 'Chọn nhận ít email hơn',
  saidWhy: 'Có nêu lý do',

  // ── Archive ──────────────────────────────────────────────────────────
  archive: 'Lưu trữ',
  publishAt: 'Xuất bản các số đã gửi tại',
  archiveUrl: 'URL trang lưu trữ',
  rssFeed: 'Nguồn cấp RSS',
  listsPublished: 'Danh sách được xuất bản',
  appearOnceSent: 'Các chiến dịch của những danh sách này xuất hiện khi đã gửi xong.',
  options: 'Tùy chọn',
  subscribeBox: 'Hiện "Nhận số tiếp theo trong hộp thư"',
  indexArchive: 'Cho phép công cụ tìm kiếm liệt kê · sitemap.xml',
  backInArchive: '"{title}" đã trở lại trang lưu trữ',
  hidden: 'Đã ẩn "{title}"',
  hiddenBody: 'Số này rời khỏi trang lưu trữ, nguồn cấp và sitemap. Bản thân email không thay đổi.',
  issueFailed: 'Không thay đổi được số này',
  issues: 'Các số',
  issuesPublished: 'Các số · {count} đã xuất bản',
  searchIssues: 'Tìm trong các số',
  noMatch: 'Không có số nào khớp với “{search}”.',
  noIssues: 'Chưa có số nào. Các chiến dịch đã gửi của những danh sách được xuất bản sẽ xuất hiện ở đây.',
  hiddenBadge: 'Đã ẩn',
  needsEdit: 'Vai trò của bạn không thể sửa trang lưu trữ.',
  show: 'Hiện',
  hide: 'Ẩn',
  range: '{from} đến {to} trên {total}',
  newer: 'Mới hơn',
  older: 'Cũ hơn',
  issueUpdated: 'Đã cập nhật số này',
  issueUpdatedBody: 'Trang lưu trữ, nguồn cấp và trang của số này giờ hiển thị thay đổi.',
  editIssue: 'Sửa số đã lưu trữ',
  sentAs: 'Đã gửi với tiêu đề "{subject}". Bản thân email không thay đổi.',
  saveIssue: 'Lưu số này',
  title: 'Tựa đề',
  titleHint: 'Để trống thì dùng tiêu đề email.',
  excerpt: 'Đoạn trích',
  excerptHint: 'Để trống thì dùng dòng xem trước.',
  label: 'Nhãn',
  labelHint: 'Hiển thị sau ngày, ví dụ "{example}".',
  filedUnder: 'Thuộc danh sách',
})
