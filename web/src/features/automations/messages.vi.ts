import { translate } from '@/i18n'
import { m } from './messages'

/** Vietnamese for the Automations list (docs/i18n/glossary-vi.md). */
export default translate(m, {
  title: 'Tự động hóa',
  route: 'Tuyến Tự động hóa · {route}',
  newAutomation: 'Tự động hóa mới',
  loadFailed: 'Không tải được tự động hóa',
  stillRunning: 'Các quy trình vẫn chạy; chỉ trang này không tải được.',
  createTitle: 'Bắt đầu từ một mẫu hoặc một khung thiết kế trống',
  routeLead: 'Các quy trình gửi qua tuyến Tự động hóa ({route}) và tự chuyển dự phòng, nên sự cố của một nhà cung cấp không bao giờ làm kẹt email chào mừng.',

  filterLabel: 'Bộ lọc',
  filter: '{filter, select, live {Đang chạy} paused {Đã tạm dừng} drafts {Bản nháp} other {Tất cả}}',
  filterCount: '{filter, select, live {Đang chạy} paused {Đã tạm dừng} drafts {Bản nháp} other {Tất cả}} · {count}',
  noneWithStatus:
    '{status, select, live {Không có tự động hóa nào đang chạy} paused {Không có tự động hóa nào đã tạm dừng} drafts {Không có bản nháp nào} other {Không có tự động hóa nào}}',
  showAll: 'Hiện mọi tự động hóa',
  noneWithStatusBody: '{count} tự động hóa trong không gian làm việc này. Hãy chọn bộ lọc khác để xem chúng.',

  inFlowsNow: 'Liên hệ đang trong quy trình',
  acrossLive: 'trên {count} tự động hóa đang chạy',
  emailsSent: 'Email đã gửi · {days} ngày',
  sentByFailover: '{provider} {count} (chuyển dự phòng ngày {date})',
  noEmailsYet: 'chưa gửi email nào',
  humanOpens: 'Lượt mở bởi người đọc',
  clicksRate: 'lượt nhấp {rate}',
  goalsReached: 'Mục tiêu đã đạt · {days} ngày',
  noneYet: 'chưa có',

  pausedLine: 'v{version} · {hasBy, select, yes {{name} tạm dừng ngày {date}} other {tạm dừng ngày {date}}}{hasReason, select, yes {: {reason}} other {}}',
  draftWord: 'bản nháp',
  publishedOn: 'xuất bản ngày {date}',

  tableLabel: 'Tự động hóa',
  colAutomation: 'Tự động hóa',
  colStatus: 'Trạng thái',
  colTrigger: 'Trình kích hoạt',
  colSteps: 'Số bước',
  colEntered: 'Đã vào · 30 ngày',
  colInFlow: 'Trong quy trình',
  colCompleted: 'Hoàn tất',
  colHumanOpens: 'Lượt mở bởi người đọc',
  colGoal: 'Mục tiêu',
  colActions: 'Thao tác',
  edit: 'Sửa',
  open: 'Mở',

  pausedAutomatically: 'Tự tạm dừng',
  stepFailing: 'Bước đang lỗi',
  loadingList: 'Đang tải các tự động hóa và số liệu 30 ngày của chúng',

  statusLive: 'Đang chạy',
  statusPaused: 'Đã tạm dừng',
  statusDraft: 'Bản nháp',

  createFailed: 'Không tạo được tự động hóa',
  starterTemplates: 'Mẫu khởi đầu',
  starterLead: 'Mỗi mẫu mở ra thành một bản nháp trên khung thiết kế; bước nào cũng sửa được',
  templateSteps: '{count} bước',
  blankCanvas: 'Khung thiết kế trống',
  blankLead: 'Chọn một trình kích hoạt và xây từ đầu',
})
