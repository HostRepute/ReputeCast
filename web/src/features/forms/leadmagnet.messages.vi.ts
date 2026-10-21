import { translate } from '@/i18n'
import { m } from './leadmagnet.messages'

/** Vietnamese for the form builder's download and pages sections (docs/i18n/glossary-vi.md). */
export default translate(m, {
  // After confirmation: offer a download
  title: 'Sau khi xác nhận',
  offer: 'Tặng một tệp tải xuống',
  offerHint: 'Một tệp dành cho người đã xác nhận, như danh sách kiểm tra hoặc cẩm nang. Tệp nằm trên trang xác nhận, qua một liên kết riêng của từng người.',
  upload: 'Tải tệp lên',
  chooseFile: 'Chọn tệp để tặng',
  fileMeta: '{pages, plural, =0 {{kind} · {size}} other {{kind} · # trang · {size}}}',
  fileKind: 'Tệp',
  replace: 'Thay',
  replaceLabel: 'Thay {name}',
  remove: 'Gỡ',
  removeLabel: 'Ngừng tặng {name}',
  days: 'Liên kết có hiệu lực trong',
  daysOption: '{days, plural, other {# ngày}}',
  daysText: 'Liên kết có hiệu lực trong {days, plural, other {# ngày}}.',
  daysSaved: 'Liên kết tải xuống giờ có hiệu lực trong {days, plural, other {# ngày}}',
  daysFailed: 'Không đổi được thời hạn của liên kết',
  numbers: '{links, plural, other {# liên kết}} · {downloaded} lượt tải xuống',
  needsDoubleOptIn: 'Tệp được tặng trên trang xác nhận, nên cần bật xác nhận hai bước.',
  none: 'Chưa tặng tệp tải xuống nào.',
  uploading: 'Đang tải lên · {done} trên {size}',
  checking: 'Đang kiểm tra tệp…',
  progress: 'Tiến độ tải lên',
  cancel: 'Hủy',
  cancelLabel: 'Hủy tải lên {name}',
  stopped: 'Việc tải lên đã dừng ở {pct}%.',
  tryAgain: 'Thử lại',
  uploaded: '{name} được tặng sau khi xác nhận',
  uploadFailed: 'Không tải lên được {name}',
  loadFailed: 'Không tải được tệp tải xuống: {error}',
  retry: 'Thử lại',
  removeBadge: 'Gỡ tệp tải xuống',
  removeTitle: 'Ngừng tặng {name}?',
  removeBody:
    '{links, plural, =0 {Người xác nhận sẽ không còn nhận được tệp tải xuống.} other {Người xác nhận sẽ không còn nhận được tệp tải xuống, và # liên kết đã gửi sẽ ngừng hoạt động.}}',
  removeConfirm: 'Gỡ tệp',
  removing: 'Đang gỡ…',
  removed: 'Không còn tặng {name}',

  // Used on pages
  pagesTitle: 'Dùng trên các trang',
  pagesNone: 'Chưa có trang nào dùng biểu mẫu này. Hãy thêm khối biểu mẫu đăng ký vào một trang trong <link>Trang</link>.',
  pagesFailed: 'Không tải được các trang: {error}',
  pageOpen: 'Mở {name} trong trình dựng trang',
  statusPublished: 'Đã xuất bản',
  statusScheduled: 'Đã lên lịch',
  statusDraft: 'Bản nháp',
  statusUnpublished: 'Đã ngừng xuất bản',
  draftOnly: '{path} · trong các thay đổi chưa xuất bản',

  // Allowed domains
  ownOrigins: 'Các trang của chính bạn trên {hosts} luôn được phép.',
})
