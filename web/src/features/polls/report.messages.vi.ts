import { translate } from '@/i18n'
import { m } from './report.messages'

/**
 * Vietnamese for poll results in the campaign report (docs/i18n/glossary-vi.md).
 * NPS groups: người ủng hộ, người trung lập, người phản đối.
 */
export default translate(m, {
  // Headline tiles.
  tileAnswers: 'Câu trả lời bình chọn',
  tileAnswersSub:
    '{percent} trên số gửi thành công · {nonContacts, plural, =0 {không có từ người không phải liên hệ} other {thêm # từ người không phải liên hệ}}',
  tileAutomated: 'Lượt truy cập tự động bị bỏ qua',
  tileAutomatedSub: 'cổng bảo mật và trình tải trước · không bao giờ là câu trả lời',

  // How answers are counted.
  howTitle: 'Cách tính câu trả lời',
  howRecorded:
    'Câu trả lời được ghi nhận trên trang trả lời, sau khi trang hiển thị một lúc, hoặc khi người trả lời bấm Xác nhận. Mở liên kết không ghi nhận gì.',
  howAutomated:
    '<b>{count, plural, other {# lượt truy cập tự động}}</b> đã bị bỏ qua: trình quét liên kết và trình tải trước do bộ phân loại của mô-đun theo dõi phát hiện.',
  howChanged: '<b>{count, plural, other {# người đã đổi câu trả lời.}}</b> Câu trả lời mới nhất được tính; lịch sử vẫn được giữ lại.',
  howNonContacts:
    '<b>{count, plural, other {# câu trả lời từ người không phải liên hệ}}</b> đến qua bản lưu trữ trên web. Chúng được tính riêng.',
  howTests: 'Câu trả lời thử từ các lần gửi thử không bao giờ được tính.',
  howPage: 'Trang trả lời: {url}',
  openPage: 'Mở trang trả lời',

  // Click map.
  mapRegion: 'Bình chọn · {count, plural, other {# câu trả lời}}',
  mapNote:
    'Cuộc bình chọn là một vùng kèm số câu trả lời, không phải tỷ lệ lượt nhấp: câu trả lời đến từ trang trả lời, nên trình quét liên kết mở các nút cũng không thêm gì vào đây.',

  // The section's header.
  kicker: 'Bình chọn',
  kickerOf: 'Bình chọn {n}/{total}',
  statusOpenCloses: 'Đang mở · đóng {date}',
  statusOpen: 'Đang mở · đến khi có người đóng',
  statusClosedScheduled: 'Đã đóng · {date} · theo lịch',
  statusClosedBy: 'Đã đóng · {date} · do {name}',
  statusClosed: 'Đã đóng · {date}',
  live: 'Trực tiếp · cập nhật mỗi phút',
  answersOf: '<b>{count, plural, other {# câu trả lời}}</b> · {percent} trên số gửi thành công',
  answersOfFinal: '<b>{count, plural, other {# câu trả lời}}</b> · {percent} trên số gửi thành công · kết quả cuối cùng',
  answersSoFar: '<b>{count, plural, other {# câu trả lời đến nay}}</b> · {elapsed} sau khi gửi',
  automatedChip: '{count, plural, other {# lượt truy cập tự động bị bỏ qua}}',
  automatedWhy:
    '{gateways, plural, other {# từ cổng bảo mật mở liên kết trước người đọc}}, {prefetch, plural, other {# từ proxy tải trước}}. Không lượt nào được tính là câu trả lời.',
  closeNow: 'Đóng bình chọn ngay',
  reopen: 'Mở lại bình chọn',
  exportResults: 'Xuất kết quả',
  exportAnswers: 'Xuất câu trả lời kèm liên hệ',
  exportAnswersAction: 'xuất câu trả lời kèm liên hệ',
  closeAction: 'đóng cuộc bình chọn',
  closeBadge: 'Đóng bình chọn',
  closeTitle: 'Đóng cuộc bình chọn này ngay?',
  closeBody: 'Trang trả lời sẽ báo cuộc bình chọn đã đóng và không ghi nhận gì nữa. Việc đóng được ghi vào nhật ký kiểm tra, và bạn có thể mở lại.',
  closeConfirm: 'Đóng bình chọn',
  closed: 'Đã đóng cuộc bình chọn',
  reopened: 'Đã mở lại cuộc bình chọn',
  failed: 'Không thay đổi được cuộc bình chọn',
  exportFailed: 'Không xuất được',
  closedNote:
    'Trang trả lời giờ hiện “Cuộc bình chọn này đã đóng vào {date}.” và không ghi nhận gì. Việc mở lại được ghi vào nhật ký kiểm tra; câu trả lời sau khi mở lại được tính như bình thường.',

  // 0–10 scale.
  nps: 'Net Promoter Score',
  npsSoFar: 'NPS đến nay',
  npsLine: 'người ủng hộ {promoters} trừ người phản đối {detractors}',
  npsEarly: 'câu trả lời sớm thường nghiêng về tích cực; chỉ số ổn định dần trong ngày đầu',
  npsBar: 'Người ủng hộ {promoters} ({pp}), người trung lập {passives} ({pap}), người phản đối {detractors} ({dp})',
  promotersItem: '<b>Người ủng hộ</b> 9–10 · {count} · {percent}',
  passivesItem: '<b>Người trung lập</b> 7–8 · {count} · {percent}',
  detractorsItem: '<b>Người phản đối</b> 0–6 · {count} · {percent}',
  promotersTitle: 'Người ủng hộ · {count}',
  passivesTitle: 'Người trung lập · {count}',
  detractorsTitle: 'Người phản đối · {count}',
  byScore: 'Câu trả lời theo điểm',
  byScoreAria: 'Câu trả lời theo điểm, từ 0 đến 10: {list}',
  scoreTitle: '{score} · {count, plural, other {# câu trả lời}}',
  scoreItem: '{score}: {count}',

  // Choice and emoji polls.
  answers: 'Câu trả lời',
  savedAsTags: 'Lưu dưới dạng thẻ',
  tagsNote: 'Khi câu trả lời thay đổi, thẻ cũng được đổi theo. Phân khúc có thể dùng các thẻ này ngay, ví dụ “có thẻ {tag}”.',
  savedToField: 'Lưu vào một trường',
  fieldNote: 'Mỗi câu trả lời đặt {field} trên liên hệ; khi câu trả lời thay đổi, giá trị của trường cũng đổi theo.',

  // Breakdown.
  breakdown: 'Phân tích',
  breakdownBy: 'Phân tích theo',
  byTier: 'Nhóm tương tác',
  byList: 'Danh sách',
  byLanguage: 'Ngôn ngữ',
  breakdownTable: 'NPS theo {by}',
  answersTable: 'Câu trả lời theo {by}',
  colTier: 'Nhóm tương tác',
  colList: 'Danh sách',
  colLanguage: 'Ngôn ngữ',
  colAnswers: 'Câu trả lời',
  colShare: 'Tỷ lệ',
  colNps: 'NPS',
  colTop: 'Được chọn nhiều nhất',
  tooFew: 'Quá ít để so sánh',
  tooFewNote: 'Hàng có dưới 30 câu trả lời ghi “Quá ít để so sánh”.',
  langSummary: 'Theo ngôn ngữ: {list}.',
  langSummaryFew: 'Theo ngôn ngữ: {list}; {few}.',
  langNps: '{name} NPS {nps}',
  langNpsShort: '{name} {nps}',
  langFew: '{name} có {count, plural, other {# câu trả lời}}',
  noBreakdown: 'Chưa có câu trả lời nào để phân tích.',

  // Over time.
  overTime: 'Câu trả lời trong 72 giờ đầu',
  soFarSix: 'Câu trả lời đến nay · 6 giờ đầu',
  cumulative: 'cộng dồn',
  overTimeNote:
    '{percent} đến trong 6 giờ đầu. {after, plural, =0 {Không có câu trả lời nào đến sau 72 giờ.} other {Thêm # câu trả lời đến sau 72 giờ.}}',
  overTimeAria: 'Câu trả lời cộng dồn theo giờ: {count} sau {hours} giờ',
  bySix: '{percent} sau 6 giờ',
  sentAt: 'Gửi lúc {time}',
  plusHours: '+{n} giờ',
  now: 'bây giờ',

  // Comments.
  comments: 'Nhận xét · {count}',
  filterComments: 'Lọc nhận xét',
  all: 'Tất cả',
  promoters: 'Người ủng hộ',
  passives: 'Người trung lập',
  detractors: 'Người phản đối',
  searchComments: 'Tìm nhận xét',
  commentsTable: 'Nhận xét',
  colScore: 'Điểm',
  colAnswer: 'Câu trả lời',
  colComment: 'Nhận xét',
  colContact: 'Liên hệ',
  colTime: 'Thời gian',
  promoter: 'người ủng hộ',
  passive: 'người trung lập',
  detractor: 'người phản đối',
  notAContact: 'Không phải liên hệ',
  aContact: 'Một liên hệ',
  quote: '“{text}”',
  showing: 'Hiển thị {shown} trên {total} · chỉ người có quyền subscribers:view mới thấy tên',
  previous: 'Trước',
  next: 'Sau',
  noComments: 'Chưa có nhận xét nào.',
  noMatch: 'Không có nhận xét nào khớp với “{query}”.',

  // Footer.
  footChanged: '{count, plural, other {# người đã đổi câu trả lời}}',
  footNonContacts: '{count, plural, other {# câu trả lời từ người không phải liên hệ, được tính riêng}}',
  footTests: 'câu trả lời thử không bao giờ được tính',
  footExports: 'Bản xuất kết quả chỉ có số liệu tổng hợp, và được ghi vào nhật ký kiểm tra',

  // Empty, loading, error.
  emptyTitle: 'Chưa có câu trả lời nào',
  emptyBody:
    '{campaign} đã được gửi lúc {time}, {ago}. Câu trả lời sẽ hiện ở đây khi người nhận chọn trên trang trả lời. Mở email hay các liên kết trong đó không được tính.',
  emptyPoll: 'Bình chọn: “{question}” · {closes}',
  closesOn: 'đóng {date}',
  loading: 'Đang tải kết quả bình chọn',
  errorTitle: 'Không tải được kết quả bình chọn',
  errorBody: 'Phần còn lại của báo cáo đã tải xong. {detail} Không có gì bị mất, và câu trả lời vẫn tiếp tục được ghi nhận.',
  retry: 'Thử lại',
  request: 'mã yêu cầu {id}',
})
