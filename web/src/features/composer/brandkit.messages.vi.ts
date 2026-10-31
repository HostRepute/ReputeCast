import { translate } from '@/i18n'
import { mbk } from './brandkit.messages'

/** Vietnamese for the brand kit in the composer (docs/i18n/glossary-vi.md). */
export default translate(mbk, {
  banner: '<b>Bắt đầu từ bộ nhận diện thương hiệu</b> Logo, màu sắc và phông chữ từ website của bạn, áp dụng cho mọi mẫu và khối mới.',
  setUp: 'Thiết lập bộ nhận diện',
  defaultColours: 'Hiển thị bằng màu mặc định cho tới khi bạn thiết lập bộ nhận diện thương hiệu',
  synced: 'Đồng bộ',
  syncedHeader: 'Đầu trang + logo',
  syncedFooter: 'Chân trang + hủy đăng ký',
  syncedHeaderNotice: 'Đầu trang đồng bộ.',
  syncedFooterNotice: 'Chân trang đồng bộ.',
  syncedNoticeBody: 'Khối này được sửa trong bộ nhận diện thương hiệu, và mọi bản nháp cũng như email đã lên lịch chứa nó sẽ cập nhật theo.',
  editInBrandKit: 'Sửa trong Bộ nhận diện',
  syncedBlock: 'Khối đồng bộ',
  syncedNamedNotice: 'Khối đồng bộ “{name}”.',
  syncedGone: 'Khối đồng bộ này đã bị xóa khỏi bộ nhận diện.',
  syncedGoneBody: 'Không còn gì thay đổi nó nữa. Hãy hủy liên kết để sửa nó trong email này.',
  unlink: 'Hủy liên kết',
  unlinked: 'Đã hủy liên kết với khối đồng bộ',
  unlinkedDetail: 'Phần này giờ chỉ thuộc về email này: hãy sửa ngay tại đây. Hoàn tác sẽ khôi phục liên kết.',
  saveSyncedTitle: 'Khối đồng bộ',
  saveSyncedLead: 'Dùng lại phần này trong các email khác. Sửa một lần trong bộ nhận diện và mọi bản nháp chứa nó sẽ cập nhật theo.',
  saveSynced: 'Lưu thành khối đồng bộ…',
  saveSyncedDialog: 'Lưu thành khối đồng bộ',
  saveSyncedDialogLead: 'Phần này được đưa vào bộ nhận diện, và email này sẽ chứa khối đồng bộ ở đúng vị trí đó. Chèn nó vào email khác từ Khối › Khối của bạn.',
  syncedName: 'Tên',
  saveSyncedConfirm: 'Lưu khối đồng bộ',
  saveSyncedFailed: 'Không lưu được khối đồng bộ: {error}',
  savedSynced: 'Đã lưu “{name}” thành khối đồng bộ',
  savedSyncedDetail: 'Chèn nó từ Khối › Khối của bạn. Sửa nó trong bộ nhận diện: mọi bản nháp chứa nó sẽ cập nhật theo.',
})
