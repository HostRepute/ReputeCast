import { translate } from '@/i18n'
import { m } from './messages'

/** Vietnamese for the Overview (docs/i18n/glossary-vi.md). */
export default translate(m, {
  // ── Page ─────────────────────────────────────────────────────────────
  title: 'Tổng quan',
  subtitleNoDomain: '{day} · chưa có tên miền gửi',
  newCampaign: 'Chiến dịch mới',
  connectFirst: 'Hãy kết nối một nhà cung cấp gửi thư trước: trình soạn thảo mở khi có nhà cung cấp được gán cho luồng Gửi hàng loạt.',
  retryNow: 'Thử lại ngay',
  viewEventLog: 'Xem nhật ký sự kiện',
  restCurrent: 'Phần còn lại của trang Tổng quan vẫn là dữ liệu mới nhất.',
  loadFailed:
    '{card, select, health {Không tải được sức khỏe người gửi} next {Không tải được chiến dịch đã lên lịch tiếp theo} providers {Không tải được nhà cung cấp gửi thư} growth {Không tải được tăng trưởng người đăng ký} tiers {Không tải được nhóm tương tác} recent {Không tải được chiến dịch gần đây} other {Không tải được thẻ này}}',
  checking: 'Đang kiểm tra danh sách chặn và API của nhà cung cấp… thường mất dưới 5 giây.',
  manageProviders: 'Quản lý nhà cung cấp và định tuyến',

  // ── Card titles ──────────────────────────────────────────────────────
  senderHealth: 'Sức khỏe người gửi',
  nextCampaign: 'Chiến dịch đã lên lịch tiếp theo',
  sendingProviders: 'Nhà cung cấp gửi thư',
  subscriberGrowth: 'Tăng trưởng người đăng ký',
  engagementTiers: 'Nhóm tương tác',
  recentCampaigns: 'Chiến dịch gần đây',

  // ── Sender health ────────────────────────────────────────────────────
  deliverabilityDetails: 'Chi tiết khả năng vào hộp thư',
  scoreLabel: 'Điểm sức khỏe người gửi {score} trên 100, {verdict}',
  verdict: '{label, select, Good {Tốt} Fair {Khá} Poor {Kém} other {{label}}}',
  reasonGood: 'Tốt:',
  reasonProblem: 'Vấn đề:',
  reasonWarning: 'Cảnh báo:',
  healthPending: 'Hiện ra khi một tên miền gửi đã được xác minh. Danh sách chặn, xác thực và giới hạn của nhà cung cấp được kiểm tra mỗi giờ.',

  // ── Next scheduled campaign ──────────────────────────────────────────
  nothingScheduled: 'Chưa có gì được lên lịch.',
  draftsWaiting: '{n, plural, other {{count} bản nháp đang chờ; hãy lên lịch một bản để xem độ sẵn sàng, tuyến gửi và chi phí ở đây.}}',
  scheduleOne: 'Hãy lên lịch một chiến dịch để xem độ sẵn sàng, tuyến gửi và chi phí ở đây.',
  openDrafts: 'Mở bản nháp',
  recipients: 'Người nhận',
  route: 'Tuyến gửi',
  estCost: 'Chi phí ước tính',
  estDuration: 'Thời lượng ước tính',
  issueBlock: 'Chặn',
  issueWarn: 'Cảnh báo',
  openReadiness: 'Mở Độ sẵn sàng gửi',
  viewCampaign: 'Xem chiến dịch',
  editCampaign: 'Sửa chiến dịch',
  nextPending: 'Chưa có chiến dịch nào. Trình soạn thảo mở khi có nhà cung cấp được gán cho luồng Gửi hàng loạt.',
  notChecked: 'Chưa kiểm tra',
  blockedBy: '{n, plural, other {Bị chặn bởi # mục kiểm tra}}',
  blocked: 'Bị chặn',
  readyWith: '{n, plural, other {Sẵn sàng, có # cảnh báo}}',
  readyWarnings: 'Sẵn sàng, có cảnh báo',
  ready: 'Sẵn sàng',

  // ── Provider cards ───────────────────────────────────────────────────
  stale: 'Dữ liệu cũ',
  staleSince: 'Dữ liệu cũ · {time}',
  bounceLimit: 'Tỷ lệ trả thư · giới hạn {limit}',
  bounceLimitReview: 'Tỷ lệ trả thư · giới hạn {limit} (ngưỡng xem xét của {provider})',
  bounceWarmup: 'Tỷ lệ trả thư · giới hạn khởi động {limit}',
  complaintLimit: 'Tỷ lệ khiếu nại · giới hạn {limit}',
  complaintWarmup: 'Tỷ lệ khiếu nại · giới hạn khởi động {limit}',
  quota24h: 'Hạn mức 24 giờ · {max}',
  quotaPlan: 'Gói · {max}/tháng',
  quotaWarmup: 'Giới hạn khởi động hôm nay · {max}',
  quotaUsed: 'đã dùng {used} ({percent})',
  quotaSent: 'đã gửi {sent} ({weekday})',
  maxRate: 'Tốc độ tối đa',
  rampRate: 'Tốc độ (tăng dần)',
  noLimit: 'Chưa đặt giới hạn',
  lastWebhook: 'Webhook gần nhất',
  costMonth: 'Chi phí · {month}',
  costPlan: '{cost} theo gói',
  costMetered: '{cost} theo lượng dùng',
  noEvents: 'chưa có sự kiện',
  justNow: 'vừa xong',
  eventsVia:
    '{webhook, select, yes {{unaffected, select, yes {Sự kiện qua webhook: nhận lần cuối {ago}, nên việc theo dõi gửi thư không bị ảnh hưởng.} other {Sự kiện qua webhook: nhận lần cuối {ago}.}}} other {{unaffected, select, yes {Sự kiện qua {feed}: nhận lần cuối {ago}, nên việc theo dõi gửi thư không bị ảnh hưởng.} other {Sự kiện qua {feed}: nhận lần cuối {ago}.}}}}',
  noFeed: '{unaffected, select, yes {Không có nguồn sự kiện, nên việc theo dõi gửi thư không bị ảnh hưởng.} other {Không có nguồn sự kiện.}}',

  // ── A failing account-health refresh (OverviewStates) ────────────────
  refreshTitle: 'Không làm mới được tình trạng tài khoản {name}',
  refreshReturned:
    '{known, select, yes {{provider} {op} trả về {what} {n, plural, =1 {một lần} =2 {hai lần} =3 {ba lần} other {# lần}}} other {{provider} {op} trả về lỗi {n, plural, =1 {một lần} =2 {hai lần} =3 {ba lần} other {# lần}}}}{shown, select, yes { ({times}).} other {.}}',
  refreshLastGood: 'Tỷ lệ trả thư và khiếu nại bên dưới lấy từ lần tải thành công gần nhất lúc {time}.',
  refreshContinue: 'Việc định tuyến và gửi thư vẫn tiếp tục.',
  refreshContinueWebhook: 'Việc định tuyến và gửi thư vẫn tiếp tục; sự kiện của {provider} vẫn đến qua webhook.',
  refreshContinueFeed: 'Việc định tuyến và gửi thư vẫn tiếp tục; sự kiện của {provider} vẫn đến qua {feed}.',

  // ── Subscriber growth and engagement tiers ───────────────────────────
  lastDays: '{days} ngày qua · {link}',
  listsUnique: '{lists, plural, other {# danh sách}} · {n, plural, other {{total} liên hệ không trùng lặp}}',
  noListsUnique: 'Chưa có danh sách nào · {n, plural, other {{total} liên hệ không trùng lặp}}',
  sparkRising: 'Người đăng ký trong {days} ngày qua, tăng từ {from} lên {to}',
  sparkFalling: 'Người đăng ký trong {days} ngày qua, giảm từ {from} xuống {to}',
  net: '{net} ròng',
  netPercent: '{net} ròng · {percent}',
  tiersLabel: 'Nhóm tương tác: {tiers}',
  tierShare: '{tier} {percent}',
  listHealthLink: 'Sức khỏe danh sách và chính sách ngừng gửi dần',
  noContacts: 'Chưa có liên hệ nào',
  sunsetPolicy: 'Chính sách ngừng gửi dần',

  // ── Recent campaigns ─────────────────────────────────────────────────
  allCampaigns: 'Tất cả chiến dịch',
  colCampaign: 'Chiến dịch',
  colSent: 'Đã gửi',
  colRecipients: 'Người nhận',
  colDelivered: 'Gửi thành công',
  colHumanOpens: 'Lượt mở bởi người đọc',
  colClicks: 'Lượt nhấp',
  colProviders: 'Nhà cung cấp gửi thư',
  colStatus: 'Trạng thái',
  importedFrom: 'nhập từ {origin}',
  imported: '{origin} · đã nhập',
  sent: 'Đã gửi',
  recentEmpty: 'Chiến dịch hiện ở đây khi đã gửi đi, cùng số thư gửi thành công, lượt mở bởi người đọc và lượt nhấp.',

  // ── Search ───────────────────────────────────────────────────────────
  searchPlaceholder: 'Tìm chiến dịch, liên hệ, sự kiện',
  searchResults: 'Kết quả tìm kiếm',
  groupCampaigns: 'Chiến dịch',
  groupEvents: 'Sự kiện',
  scheduledOn: 'lên lịch {date}',
  campaignStatus:
    '{status, select, draft {bản nháp} approval {chờ phê duyệt} approved {đã phê duyệt} scheduled {đã lên lịch} sending {đang gửi} paused {tạm dừng} sent {đã gửi} cancelled {đã hủy} failed {thất bại} other {{status}}}',
  eventType:
    '{type, select, delivered {gửi thành công} deferred {bị hoãn} hard_bounce {trả thư vĩnh viễn} soft_bounce {trả thư tạm thời} complaint {khiếu nại} rejected {bị từ chối} other {{label}}}',
  searching: 'Đang tìm…',
  searchNothing: 'Không có kết quả nào khớp với “{term}”. Tìm theo tên chiến dịch và địa chỉ người nhận trong sự kiện của nhà cung cấp.',

  // ── First run (OverviewEmpty) ────────────────────────────────────────
  step: '{current, select, yes {Bước {n}, đang thực hiện:} other {Bước {n}:}}',
  selfHosted: 'Tự lưu trữ · {version} · đầy đủ mọi tính năng',
  welcome: 'Chào mừng bạn đến với {name}',
  welcomeLead:
    'ReputeCast gửi thư qua một nhà cung cấp gửi thư do bạn kiểm soát. Hãy kết nối một nhà cung cấp ngay, trang này sẽ hiện sức khỏe người gửi, giới hạn của nhà cung cấp và độ sẵn sàng của chiến dịch đầu tiên.',
  stepConnect: 'Kết nối một nhà cung cấp gửi thư',
  stepConnectLine: 'Amazon SES, Postmark, Mailgun, SendGrid, Resend, Brevo, máy chủ Postal của riêng bạn hoặc bất kỳ máy chủ SMTP nào. Mất khoảng 10 phút.',
  chooseProvider: 'Chọn nhà cung cấp',
  browseAdapters: 'Xem bộ điều hợp của cộng đồng',
  stepVerify: 'Xác minh tên miền gửi',
  stepVerifyLine: 'Bản ghi DKIM, return-path, SPF và DMARC, xác minh trực tiếp. Mở sau bước 1.',
  stepImport: 'Nhập liên hệ',
  importNow: 'Nhập ngay',
  stepImportLine: 'Từ Brevo, Mailchimp, MailerLite, listmonk, Keila hoặc CSV. Liên hệ đã hủy đăng ký hoặc bị trả thư vẫn bị chặn gửi.',
  subscribers: 'Người đăng ký',
  subscribersNoLists: 'Hãy tạo một danh sách để thu thập liên hệ, hoặc nhập liên hệ cùng danh sách của họ.',
  subscribersWaiting:
    '{n, plural, =1 {Trong 1 danh sách: {lists}. Chưa có gì được gửi cho họ cho đến khi kết nối một nhà cung cấp gửi thư.} other {Trong # danh sách: {lists}. Chưa có gì được gửi cho họ cho đến khi kết nối một nhà cung cấp gửi thư.}}',
  listsReady: '{n, plural, other {# danh sách đã sẵn sàng nhận liên hệ: {lists}.}}',

  // ── Loading (OverviewStates) ─────────────────────────────────────────
  loadingHealth: 'Đang tải sức khỏe người gửi',
  loadingNext: 'Đang tải chiến dịch tiếp theo',
  loadingProviders: 'Đang tải nhà cung cấp gửi thư',
  loading: 'Đang tải',
})
