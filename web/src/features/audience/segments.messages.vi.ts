import { translate } from '@/i18n'
import { m } from './segments.messages'

/**
 * Vietnamese for the segment builder (docs/i18n/glossary-vi.md). The group
 * words AND, ALL and ANY are VÀ, TẤT CẢ and BẤT KỲ; "SQL" and permission
 * keys stay as they are.
 */
export default translate(m, {
  // ── Page ─────────────────────────────────────────────────────────────
  documentTitle: 'Phân khúc',
  newSegment: 'Phân khúc mới',
  loadFailed: 'Không tải được phân khúc',
  nothingChanged: 'Không có gì bị thay đổi.',
  savedNew: 'Đã lưu phân khúc "{name}"',
  saved: 'Đã lưu "{name}"',
  discard: 'Bỏ thay đổi',
  save: 'Lưu phân khúc',
  unsaved: 'Bạn đã thay đổi phân khúc này kể từ lần lưu. Rời đi bây giờ sẽ giữ phiên bản đã lưu.',
  segment: 'Phân khúc',
  name: 'Tên phân khúc',
  namePlaceholder: 'Đặt tên cho phân khúc này',
  refresh: 'Cập nhật',
  dynamic: 'Động · tính lại lúc gửi',
  static: 'Ảnh chụp cố định',

  // ── Conditions ───────────────────────────────────────────────────────
  conditions: 'Điều kiện',
  combine: 'Các nhóm kết hợp bằng VÀ; trong một nhóm, chọn TẤT CẢ hoặc BẤT KỲ',
  and: 'VÀ',
  group: 'Nhóm {n}',
  matchRule: 'Quy tắc khớp cho Nhóm {n}',
  matchAll: 'Khớp TẤT CẢ điều kiện này',
  matchAny: 'Khớp BẤT KỲ điều kiện nào',
  removeGroup: 'Xóa nhóm',
  addCondition: 'Thêm điều kiện',
  addGroup: 'Thêm nhóm',
  alwaysExcluded: 'Địa chỉ bị chặn gửi, bị trả thư và địa chỉ chức danh luôn bị loại lúc gửi, dù phân khúc nói gì.',
  field: 'Trường điều kiện',
  operator: 'Toán tử điều kiện',
  value: 'Giá trị điều kiện',
  valueInput: 'Giá trị',
  period: '{amount} {unit, select, weeks {tuần} other {ngày}}',
  weeks: 'Số tuần',
  days: 'Số ngày',
  unit: 'Đơn vị thời gian',
  unitDays: 'ngày',
  unitWeeks: 'tuần',
  nothingToPick: 'Chưa có gì để chọn',
  removeCondition: 'Xóa điều kiện',

  // ── SQL ──────────────────────────────────────────────────────────────
  sql: 'SQL',
  sqlLimits: 'Chỉ đọc, giới hạn 10 giây · view: {views}',
  sqlLabel: 'SQL của phân khúc',
  backToBuilder: 'Quay lại trình tạo',
  sqlNote: 'Truy vấn phải trả về id liên hệ (c.id). Truy vấn chạy ở chế độ chỉ đọc với giới hạn 10 giây.',
  generated: 'SQL được tạo',
  readOnly: 'chỉ đọc',
  copied: 'Đã sao chép SQL',
  needsSql: 'Cần quyền segments:sql',
  sqlOff: 'Phân khúc SQL đang tắt trên cơ sở dữ liệu này; quản trị viên có thể bật.',
  openSql: 'Mở trong trình soạn SQL',
  needsSqlNote: 'cần quyền segments:sql',
  countAndSaved: 'Số lượng và phân khúc đã lưu',

  // ── Live count ───────────────────────────────────────────────────────
  timedOut: 'Quá thời gian',
  timedOutTitle: '{named, select, yes {Đếm trực tiếp cho "{name}" mất hơn 10 giây} other {Đếm trực tiếp cho "phân khúc này" mất hơn 10 giây}}',
  slowCounts: 'Đếm phân khúc trên {count} liên hệ với điều kiện sự kiện có thể chậm khi dùng cấu hình Postgres mặc định.',
  lastKnown: '{today, select, yes {Hiện số đếm gần nhất ({count}, lúc {time} hôm nay) để thay thế.} other {Hiện số đếm gần nhất ({count}, lúc {time} {date}) để thay thế.}}',
  stillWorks: 'Phân khúc vẫn hoạt động lúc gửi.',
  retryCount: 'Đếm lại',
  tuningGuide: 'Hướng dẫn tinh chỉnh Postgres',
  liveCount: 'Đếm trực tiếp · cập nhật khi bạn sửa',
  matchLine: 'liên hệ khớp · {share} trên {total} · đếm trong {ms} ms',
  tierMix: 'Tỷ lệ nhóm tương tác của phân khúc: {tiers}',
  tierPercent: '{tier} {n} phần trăm',
  counting: 'Đang đếm',
  previewContacts: 'Xem trước liên hệ',
  saveFirst: 'Hãy lưu phân khúc để xem trước các liên hệ của nó.',
  useInCampaign: 'Dùng trong chiến dịch',

  // ── Saved segments ───────────────────────────────────────────────────
  savedSegments: 'Phân khúc đã lưu',
  loadingSegments: 'Đang tải phân khúc',
  addNew: '+ Phân khúc mới',
  loading: 'Đang tải phân khúc',
})
