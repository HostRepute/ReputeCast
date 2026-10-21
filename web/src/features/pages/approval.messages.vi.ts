import { translate } from '@/i18n'
import { m } from './approval.messages'

/** Vietnamese for pages in the approval inbox and review (docs/i18n/glossary-vi.md › Website and pages). */
export default translate(m, {
  // The inbox
  badge: 'Trang',
  badgeLabel: 'Trang: mở “{name}” trong trình dựng trang',
  line: 'Trang · {path} · {visibility}',
  whenApproved: 'Khi được phê duyệt',
  scheduled: '{date}, {time}',
  published: 'Đã xuất bản',

  // The review
  cardTitle: 'Trang',
  cardCaption: 'Những gì phê duyệt sẽ xuất bản: nội dung, địa chỉ, chế độ hiển thị và thời điểm xuất bản',
  page: 'Trang',
  pageValue: '{name} · {template}',
  address: 'Địa chỉ',
  visibility: 'Chế độ hiển thị',
  goesLive: 'Thời điểm xuất bản',
  goesLiveNow: 'Ngay khi được phê duyệt',
  goesLivePublished: 'Đã xuất bản khi được phê duyệt',
  version: 'Phiên bản',
  versionValue: 'v{version}',
  previewTitle: 'Bản xem trước của “{name}”, v{version}',
  previewFullTitle: 'Bản xem trước đầy đủ của “{name}”',
  previewCaption: 'Trang như người đọc sẽ thấy, trong khung của trang web',
  previewUnavailable: 'Không có bản xem trước: {error}',
  previewLoading: 'Bản xem trước đang tải.',
  openPreview: 'Mở bản xem trước đầy đủ',
  openBuilder: 'Mở trong trình dựng trang',

  // The decision
  decisionNote:
    'Phê duyệt áp dụng cho đúng phiên bản này: nội dung, địa chỉ, chế độ hiển thị và thời điểm xuất bản. {when, select, at {Trang được xuất bản vào thời điểm đã đặt} other {Trang được xuất bản ngay khi được phê duyệt}}. Nếu {name} thay đổi trang trước đó, việc xuất bản cần phê duyệt mới.',
  rejectDetail: '{name} sẽ được báo và trang không được xuất bản. Hãy từ chối khi trang hoàn toàn không nên xuất bản; nếu cần sửa, hãy yêu cầu chỉnh sửa.',
  changesDetail: 'Yêu cầu được gửi lại cho {name} kèm nhận xét của bạn. Trang không được xuất bản cho đến khi họ yêu cầu lại.',
})
