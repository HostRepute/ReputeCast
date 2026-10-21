import { translate } from '@/i18n'
import { m } from './messages'

/**
 * Vietnamese for the approval statuses, the diff lines and the approval
 * inbox (docs/i18n/glossary-vi.md). The wording follows Review & send's
 * approval cards: "bản chụp", "người phê duyệt", "tác giả".
 */
export default translate(m, {
  // ── Request statuses (badges) ─────────────────────────────────────────
  statusPending: 'Đang chờ',
  statusPendingLong: 'Chờ phê duyệt',
  statusApproved: 'Đã phê duyệt',
  statusChangesRequested: 'Yêu cầu thay đổi',
  statusRejected: 'Bị từ chối',
  statusInvalidated: 'Mất hiệu lực',
  statusBypassed: 'Đã bỏ qua',
  statusSent: 'Đã gửi',
  statusCancelled: 'Đã hủy',

  // ── Diff lines and what stayed the same ───────────────────────────────
  whatChanged: 'Những gì thay đổi',
  before: 'Trước:',
  after: 'Sau:',
  diffLabelled: '{label}: {value}',
  unchangedAudience: 'Người nhận {count}',
  unchangedAudienceMoved: 'Người nhận {count} ({change})',
  unchangedRoute: 'tuyến',
  unchangedScheduleNow: 'lịch gửi: gửi ngay',
  unchangedSchedule: 'lịch gửi {when}',
  fieldAudience: 'người nhận',
  fieldRoute: 'tuyến',
  fieldSchedule: 'lịch gửi',
  unchangedSentence: '{fields} không thay đổi.',
  rulesLine: 'quy tắc: {rules}',

  // ── Approval inbox ────────────────────────────────────────────────────
  title: 'Phê duyệt',
  filterLabel: 'Trạng thái phê duyệt',
  filter: '{status, select, pending {Đang chờ} approved {Đã phê duyệt} changes_requested {Yêu cầu thay đổi} rejected {Bị từ chối} invalidated {Mất hiệu lực} other {Tất cả}}',
  filterCount:
    '{status, select, pending {Đang chờ} approved {Đã phê duyệt} changes_requested {Yêu cầu thay đổi} rejected {Bị từ chối} invalidated {Mất hiệu lực} other {Tất cả}} · {count}',
  loadFailed: 'Không tải được các yêu cầu phê duyệt',
  nothingChanged: 'Không có gì bị thay đổi.',
  requestsTitle: 'Yêu cầu phê duyệt',
  requestsCaptionAll: 'Mọi thứ cần người thứ hai duyệt · người phê duyệt thấy mọi yêu cầu, tác giả thấy yêu cầu của mình',
  requestsCaptionOwn: 'Yêu cầu của bạn · người phê duyệt thấy mọi yêu cầu',
  filterByAuthor: 'Lọc theo tác giả',
  anyAuthor: 'Mọi tác giả',
  filterByApprover: 'Lọc theo người phê duyệt',
  anyApprover: 'Mọi người phê duyệt',
  dateRange: 'Khoảng thời gian',
  lastDays: '{days} ngày qua',
  allTime: 'Mọi thời điểm',
  emptyPendingTitle: 'Không có gì đang chờ bạn',
  emptyPendingBody: 'Yêu cầu hiện ở đây và trong thông báo của bạn. Tác giả vẫn xem được yêu cầu của chính họ.',
  emptyTitleYet:
    '{status, select, pending {Chưa có yêu cầu nào đang chờ} approved {Chưa có yêu cầu nào được phê duyệt} changes_requested {Chưa có yêu cầu nào cần thay đổi} rejected {Chưa có yêu cầu nào bị từ chối} invalidated {Chưa có yêu cầu nào mất hiệu lực} other {Chưa có yêu cầu nào}}',
  emptyTitleDays:
    '{status, select, pending {Không có yêu cầu nào đang chờ trong {days} ngày qua} approved {Không có yêu cầu nào được phê duyệt trong {days} ngày qua} changes_requested {Không có yêu cầu nào cần thay đổi trong {days} ngày qua} rejected {Không có yêu cầu nào bị từ chối trong {days} ngày qua} invalidated {Không có yêu cầu nào mất hiệu lực trong {days} ngày qua} other {Không có yêu cầu nào trong {days} ngày qua}}',
  emptyFiltered: 'Hãy xóa bộ lọc hoặc chọn khoảng thời gian dài hơn để xem thêm.',
  emptyNone: 'Yêu cầu sẽ hiện ở đây khi có người yêu cầu phê duyệt.',
  tableCaption: 'Yêu cầu phê duyệt',
  colCampaign: 'Chiến dịch',
  colAuthor: 'Tác giả',
  colAudience: 'Người nhận',
  colRoute: 'Tuyến',
  colScheduled: 'Lịch gửi',
  colRequested: 'Yêu cầu lúc',
  colStatus: 'Trạng thái',
  colActions: 'Thao tác',
  scheduledSent: 'Đã gửi {date}',
  unscheduled: 'Chưa lên lịch',
  noteWithdrawn: 'tác giả đã rút lại',
  noteBypassed: 'Chủ sở hữu đã gửi mà không cần phê duyệt',
  report: 'Báo cáo',
  review: 'Xem xét',
  viewDiff: 'Xem khác biệt',
  view: 'Xem',
  policyTitle: 'Chính sách đang áp dụng',
  editPolicy: 'Sửa chính sách',
  viewPolicy: 'Xem chính sách',
  policyRules: 'Quy tắc',
  policyApprovals: 'Phê duyệt',
  policyMedianWait: 'Thời gian chờ trung vị',
  noRules: 'Không có quy tắc nào được bật: các lần gửi không bao giờ cần phê duyệt',
  approvalsFrom: '{count} lượt, từ {approvers}',
  approvalsFromNobody: '{count} lượt, nhưng chưa có ai phê duyệt được (chưa thành viên nào có quyền campaigns:approve)',
  medianWait: '{wait} trong 30 ngày qua',
  medianWaitNone: 'Không có quyết định nào trong 30 ngày qua',
  loading: 'Đang tải các yêu cầu phê duyệt',
})
