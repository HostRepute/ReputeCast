import { translate } from '@/i18n'
import { m } from './lists.messages'

/** Vietnamese for deleting and restoring a list (docs/i18n/glossary-vi.md). */
export default translate(m, {
  // ── Delete list dialog ───────────────────────────────────────────────
  badge: 'Xóa danh sách',
  title: 'Xóa "{name}"?',
  stillCounting: 'Vẫn đang đếm số liên hệ trong {name}. Hãy thử lại sau giây lát.',
  counting: 'Đang đếm số liên hệ trong {name}…',
  confirm: 'Xóa danh sách',
  deleting: 'Đang xóa…',
  typeName: 'Nhập tên danh sách để xác nhận',
  typeHint: 'Danh sách đã xóa có thể khôi phục từ nhật ký kiểm tra trong 30 ngày; các lần gửi trong thời gian đó sẽ bỏ qua danh sách này.',
  affects: 'Ảnh hưởng',
  deleted: 'Đã xóa "{name}" · liên hệ vẫn được giữ',
  restored: 'Đã khôi phục "{name}"',
  restoreFailed: 'Không khôi phục được "{name}"',

  // ── What deleting it does (membersText) ──────────────────────────────
  kept: 'Địa chỉ bị chặn gửi không bao giờ bị gỡ.',
  nobody: 'Không ai đăng ký danh sách này. Địa chỉ bị chặn gửi không bao giờ bị gỡ.',
  onList: '{n, plural, other {Danh sách này có {count} liên hệ.}}',
  allElsewhere: '{n, plural, =1 {Liên hệ đó cũng có trong {lists} và vẫn đăng ký ở đó.} other {Tất cả đều có trong {lists} và vẫn đăng ký ở đó.}}',
  allAlone: '{n, plural, =1 {Liên hệ đó không có trong danh sách nào khác và sẽ thành liên hệ không đăng ký danh sách nào.} other {Tất cả đều không có trong danh sách nào khác và sẽ thành liên hệ không đăng ký danh sách nào.}}',
  split:
    '{elsewhere, plural, other {{elsewhereCount} liên hệ cũng có trong {lists} và vẫn đăng ký ở đó}}; {alone, plural, other {{aloneCount} liên hệ không có trong danh sách nào khác và sẽ thành liên hệ không đăng ký danh sách nào.}}',
  otherLists: '{n, plural, other {{count} danh sách khác}}',

  // ── What uses it (affectsText) ───────────────────────────────────────
  forms: '{n, plural, other {{count} biểu mẫu đăng ký}}',
  automations: '{n, plural, other {{count} trình kích hoạt tự động hóa}}',
  automationsNamed: '{n, plural, other {{count} trình kích hoạt tự động hóa ({names})}}',
  campaigns: '{n, plural, other {{count} chiến dịch đã lên lịch}}',
  usesIt: '{one, select, yes {{things} đang dùng danh sách này và sẽ tạm dừng.} other {{things} đang dùng danh sách này và sẽ tạm dừng.}}',

  // ── What a restore turned back on (restoredText) ─────────────────────
  restoredForms: '{n, plural, other {{count} biểu mẫu đăng ký}}',
  restoredAutomations: '{n, plural, other {{count} quy trình tự động hóa}}',
  liveAgain: '{one, select, yes {{things} đã hoạt động lại} other {{things} đã hoạt động lại}}',
  backOnSchedule: '{n, plural, other {{campaigns} đã trở lại lịch gửi}}',
  membersBack: 'Các thành viên của danh sách đã trở lại.',
  stayAsIs: '{n, plural, other {{names} vẫn giữ nguyên.}}',
})
