import { translate } from '@/i18n'
import { m } from './health.messages'

/** Vietnamese for List health (docs/i18n/glossary-vi.md). The Sunset tier is "Ngừng gửi dần", its policy "chính sách ngừng gửi dần". */
export default translate(m, {
  // ── Page ─────────────────────────────────────────────────────────────
  documentTitle: 'Sức khỏe danh sách',
  recalculated: 'Nhóm tương tác được tính lại mỗi đêm · lần chạy gần nhất {time}',
  loadFailed: 'Không tải được sức khỏe danh sách',
  nothingChanged: 'Không có gì bị thay đổi.',
  loading: 'Đang tải sức khỏe danh sách',

  // ── Tier cards ───────────────────────────────────────────────────────
  tierWindow: '{tier} · {window}',
  pts: '<c>{delta} điểm</c>',
  ptsVsLastMonth: '<c>{delta} điểm</c> so với tháng trước',
  entering: '{count} liên hệ vào nhóm này trong tuần',

  // ── Trend ────────────────────────────────────────────────────────────
  trendTitle: 'Xu hướng nhóm tương tác · 12 tuần qua',
  trendDesc: 'Tỷ lệ các nhóm tương tác theo tuần trong {weeks} tuần; {active}, {sunset}',
  trendDescPolicy: 'Tỷ lệ các nhóm tương tác theo tuần trong {weeks} tuần; {active}, {sunset} sau khi chính sách ngừng gửi dần bắt đầu vào {date}',
  tierStays: '{tier} giữ ở mức {to} phần trăm',
  tierGrows: '{tier} tăng từ {from} lên {to} phần trăm',
  tierShrinks: '{tier} giảm từ {from} xuống {to} phần trăm',
  noHistory: 'Chưa có lịch sử nhóm tương tác',
  policyEnabledOn: '{date} · bật chính sách ngừng gửi dần',
  trendNote:
    'Mỗi cột là một tuần; chiều cao là tỷ lệ trong danh sách. Giá trị của tuần này: Tích cực {active}, Nguội dần {cooling}, Có nguy cơ {atRisk}, Ngừng gửi dần {sunset}.',

  // ── Sunset policy ────────────────────────────────────────────────────
  daysWithoutOpen: '{n} ngày không mở thư',
  days: '{n} ngày',
  thresholdsNumbers: 'Hãy nhập ngưỡng của các nhóm tương tác bằng số ngày.',
  policySaved: 'Đã lưu chính sách ngừng gửi dần',
  policySavedBody: 'Nhóm tương tác theo ngưỡng mới sau lần chạy lúc {time} đêm nay.',
  winBack: 'đã kéo lại {rate} đến nay',
  policyTitle: 'Chính sách ngừng gửi dần',
  onSince: 'Bật từ {date}',
  onSinceToday: 'Bật từ hôm nay',
  activeUntil: 'Tích cực cho đến',
  atRiskFrom: 'Có nguy cơ từ',
  sunsetFrom: 'Ngừng gửi dần từ',
  countMachine: 'Tính lượt mở tự động?',
  humanOnly: 'Không, chỉ người đọc mở và nhấp',
  reengage: 'Gửi một email kéo tương tác lại khi liên hệ vào nhóm Ngừng gửi dần',
  reengageAutomation: '· tự động hóa "{name}"',
  reengageDefault: 'Kéo tương tác lại',
  stopBroadcasts: 'Ngừng gửi hàng loạt {n} ngày sau đó nếu vẫn không mở thư',
  stopNote: '· Tự động hóa và thư Hệ thống vẫn tiếp tục',
  deleteAfter: 'Xóa liên hệ sau {n} tháng ở nhóm Ngừng gửi dần',
  deleteOn: '· xóa vĩnh viễn cùng lịch sử đồng ý; danh sách chặn gửi vẫn giữ',
  deleteOff: '· tắt; thời gian lưu giữ theo GDPR được đặt trong Cài đặt',
  savePolicy: 'Lưu chính sách',
  editFlow: 'Sửa quy trình kéo tương tác lại',
  thisWeek: '<b>Tuần này:</b> {text}.',
  weekEnter:
    '{day, select, none {{reengage, select, yes {{count} liên hệ vào nhóm Ngừng gửi dần và nhận email kéo tương tác lại} other {{count} liên hệ vào nhóm Ngừng gửi dần}}} other {{reengage, select, yes {{count} liên hệ vào nhóm Ngừng gửi dần vào {day} và nhận email kéo tương tác lại} other {{count} liên hệ vào nhóm Ngừng gửi dần vào {day}}}}}',
  weekStop:
    '{day, select, none {{count} liên hệ không phản hồi tháng trước sẽ ngừng nhận thư Gửi hàng loạt} other {{count} liên hệ không phản hồi tháng trước sẽ ngừng nhận thư Gửi hàng loạt vào {day}}}',

  // ── Health per list ──────────────────────────────────────────────────
  perList: 'Sức khỏe theo danh sách',
  figures: 'Số liệu 30 ngày · {link}',
  colList: 'Danh sách',
  colContacts: 'Liên hệ',
  colMix: 'Tỷ lệ nhóm tương tác',
  colNet: 'Tăng ròng · 30 ngày',
  colHumanOpens: 'Lượt mở bởi người đọc',
  colUnsub: 'Tỷ lệ hủy đăng ký',
  colComplaint: 'Tỷ lệ khiếu nại',
  colBounce: 'Tỷ lệ trả thư',
  colHealth: 'Sức khỏe',
  health: '{health, select, excellent {Rất tốt} good {Tốt} attention {Cần chú ý} repermission {Cần xin phép lại} other {{raw}}}',
  savedView: '(chế độ xem đã lưu)',
  actionsFor: 'Thao tác cho {name}',
  deleteList: 'Xóa danh sách',
  tierShare: '{tier} {percent}',
  noLists: 'Chưa có danh sách nào. <link>Thêm liên hệ</link> để xem từng danh sách đang ra sao.',
  startFlow: 'Bắt đầu quy trình xin phép lại',
})
