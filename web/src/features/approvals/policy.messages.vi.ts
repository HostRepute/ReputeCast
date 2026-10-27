import { translate } from '@/i18n'
import { mp } from './policy.messages'

/** Vietnamese for Settings › Approvals (docs/i18n/glossary-vi.md). */
export default translate(mp, {
  settings: 'Cài đặt',
  documentTitle: 'Phê duyệt · Cài đặt',
  loadFailed: 'Không tải được chính sách phê duyệt',
  nothingChanged: 'Không có gì bị thay đổi.',
  loading: 'Đang tải chính sách phê duyệt',

  numberRange: 'Hãy nhập số người nhận từ {min} đến {max}.',
  saved: 'Đã lưu chính sách phê duyệt',
  savedDetail: 'Chính sách áp dụng từ yêu cầu tiếp theo; các yêu cầu đang mở giữ những quy tắc chúng đã khớp.',
  saveFailed: 'Không lưu được chính sách phê duyệt',

  readOnly: 'Chỉ đọc',
  readOnlyLead: 'Bạn có thể xem chính sách phê duyệt.',
  readOnlyReason: 'Vai trò của bạn không thể thay đổi chính sách này.',

  // ── When is approval required? ────────────────────────────────────────
  whenTitle: 'Khi nào cần phê duyệt?',
  whenCaption: 'Theo không gian làm việc',
  whenIntro: 'Cần phê duyệt khi bất kỳ quy tắc nào đang bật được khớp. Tác giả thấy những quy tắc đã khớp và có thể thêm ghi chú.',
  audienceTitle: 'Một lần gửi hàng loạt có ít nhất',
  audienceDetail: 'Đếm sau khi loại trừ, lúc yêu cầu và một lần nữa lúc gửi.',
  minimumRecipients: 'Số người nhận tối thiểu',
  recipients: 'người nhận',
  warmingTitle: 'Tuyến có một nhà cung cấp Đang khởi động, hoặc một nhà cung cấp được kết nối trong {days} ngày qua',
  matchesToday: 'Đang khớp: {what}',
  warmingNone: 'Hiện không khớp gì: không nhà cung cấp nào trong tuyến Gửi hàng loạt đang khởi động hoặc mới kết nối.',
  warningsTitle: 'Độ sẵn sàng gửi là “Sẵn sàng, có cảnh báo”',
  warningsDetail: 'Lần gửi bị chặn không bao giờ bắt đầu; quy tắc này thêm một bước kiểm tra của con người cho các cảnh báo.',
  lacksSendTitle: 'Tác giả không có quyền gửi',
  lacksSendDetail:
    'Thành viên có thể sửa nhưng không thể gửi chiến dịch vẫn chuẩn bị được lần gửi và yêu cầu phê duyệt. {who, select, roles {{roles} hiện có: {names}.} members {Thành viên hiện có: {names}.} other {Hiện không có ai trong trường hợp đó.}}',
  listsTitle: 'Người nhận gồm một danh sách được đánh dấu “Cần phê duyệt”',
  listsDetail:
    '{marked, select, yes {Đang được đánh dấu: {lists}. Đánh dấu thêm trong <link>Người đăng ký → Danh sách</link>.} other {Chưa có danh sách nào được đánh dấu. Hãy đánh dấu danh sách trong <link>Người đăng ký → Danh sách</link>.}}',
  automationTitle: 'Xuất bản một tự động hóa cũng cần phê duyệt',
  automationDetail: 'Áp dụng cho mọi người có thể xuất bản tự động hóa. Khi tắt: việc xuất bản chỉ được ghi nhật ký.',
  postPublishing: 'việc xuất bản bài viết và trang',

  // ── Approvers ─────────────────────────────────────────────────────────
  approversTitle: 'Người phê duyệt',
  approversNote:
    'Người phê duyệt là những người có vai trò được phép phê duyệt chiến dịch. {has, select, yes {Hiện có: {list}.} other {Hiện chưa ai phê duyệt được: hãy cho phép một vai trò phê duyệt chiến dịch trong Cài đặt › Vai trò.}} Tác giả không bao giờ tự phê duyệt được chiến dịch của mình.',
  approverWithRole: '<b>{name}</b> ({role})',
  approverName: '<b>{name}</b>',
  requiredLegend: 'Số lượt phê duyệt cần có',
  oneApproval: '1 lượt phê duyệt',
  oneApprovalDetail: 'Bất kỳ người phê duyệt nào không phải tác giả',
  twoApprovals: '2 lượt phê duyệt',
  twoApprovalsDetail: 'Hai người phê duyệt khác nhau; với {count} người phê duyệt hiện có, một trong hai người không được là tác giả',
  twoNeeds: 'Cần hai thành viên có thể phê duyệt chiến dịch; hiện có {count}.',
  snapshot: 'Bản chụp',
  snapshotDetail:
    'Một phê duyệt áp dụng cho nội dung và tiêu đề, định nghĩa người nhận, tuyến gửi và lịch gửi. Thay đổi bất kỳ mục nào trong số đó, hoặc số người nhận thay đổi quá 10%, sẽ làm phê duyệt mất hiệu lực; người phê duyệt thấy chính xác những gì đã thay đổi.',
  recheck: 'Kiểm tra lại',
  recheckDetail: 'Độ sẵn sàng gửi được kiểm tra lại khi bắt đầu gửi. Kết quả Bị chặn sẽ dừng lần gửi và thông báo cho tác giả và người phê duyệt.',

  // ── Emergency bypass ──────────────────────────────────────────────────
  bypassTitle: 'Bỏ qua khẩn cấp',
  bypassCaption: 'Chỉ Chủ sở hữu',
  bypassSwitch: 'Chủ sở hữu có thể gửi mà không cần phê duyệt bằng cách nhập lý do',
  bypassLogged: 'Việc bỏ qua được ghi là <mono>campaign.approval_bypassed</mono> và mọi người phê duyệt được thông báo ngay.',
  used: 'Đã dùng',
  usedCount: '{count} lần trong 90 ngày qua',

  discard: 'Bỏ thay đổi',
  savePolicy: 'Lưu chính sách',
  unsaved: 'Các thay đổi của bạn với chính sách phê duyệt chưa được lưu. Các yêu cầu vẫn theo chính sách đã lưu.',

  // ── Pending now ───────────────────────────────────────────────────────
  pendingTitle: 'Đang chờ · {count}',
  approvalInbox: 'Hộp thư phê duyệt',
  pendingCaption: 'Yêu cầu phê duyệt đang chờ',
  colCampaign: 'Chiến dịch',
  colAuthor: 'Tác giả',
  colAudience: 'Người nhận',
  colRequested: 'Yêu cầu lúc',
  colActions: 'Thao tác',
  nothingPending: 'Không có gì đang chờ phê duyệt.',
  review: 'Xem xét',
})
