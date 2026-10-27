import { translate } from '@/i18n'
import { mr } from './messages'

/**
 * Vietnamese for the review link page (docs/i18n/glossary-vi.md): a
 * comment is "nhận xét"; someone reviewing without an account is "người
 * góp ý"; Subject · Preheader are "Tiêu đề · Dòng xem trước".
 */
export default translate(mr, {
  docTitle: 'Góp ý: {campaign} · {workspace}',
  docTitleLoading: 'Góp ý · ReputeCast',
  skip: 'Chuyển đến email',
  badge: 'Bản để góp ý',
  sharedBy: '{workspace} chia sẻ',
  caption: 'Phiên bản {version} · liên kết hết hạn {date}',
  from: 'Từ',
  subject: 'Tiêu đề',
  preheader: 'Dòng xem trước',
  sample: 'Cá nhân hóa cho một người đọc mẫu, {name}.',
  frame: 'Email',
  comments: 'Nhận xét',
  commentsLead: 'Nhận xét của bạn được gửi đến nhóm của {workspace}. Họ thấy tên bạn ở mỗi nhận xét.',
  yourName: 'Tên của bạn',
  namePlaceholder: 'Để nhóm biết ai đã nhận xét',
  comment: 'Nhận xét',
  commentPlaceholder: 'Nên thay đổi điều gì?',
  post: 'Nhận xét',
  posting: 'Đang đăng…',
  reply: 'Trả lời',
  replyTo: 'Trả lời {name}',
  threadBy: 'Nhận xét của {name}',
  cancel: 'Hủy',
  noComments: 'Chưa có nhận xét nào.',
  reviewer: 'Người góp ý',
  resolved: 'Đã giải quyết',
  postFailed: 'Không đăng được nhận xét',
  goneBadge: 'Liên kết không dùng được',
  goneTitle: 'Liên kết góp ý này không còn dùng được',
  goneBody: 'Liên kết đã hết hạn hoặc bị thu hồi. Hãy xin người đã chia sẻ một liên kết mới.',
  limitedBadge: 'Quá nhiều yêu cầu',
  limitedTitle: 'Có quá nhiều yêu cầu từ mạng của bạn',
  limitedBody: 'Hãy đợi vài phút rồi thử lại.',
  errorBadge: 'Đã xảy ra lỗi',
  errorTitle: 'Không mở được email',
  errorBody: 'Hãy kiểm tra kết nối rồi thử lại.',
  madeWith: 'Được tạo bằng ReputeCast, nền tảng bản tin tự lưu trữ.',
})
