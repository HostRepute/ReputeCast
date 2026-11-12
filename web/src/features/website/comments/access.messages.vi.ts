import { translate } from '@/i18n'
import { ma } from './access.messages'

/** Vietnamese for Website › Access › Comments. */
export default translate(ma, {
  title: 'Bình luận',
  text: 'Hội viên bình luận dưới những bài viết họ đọc được. Mặc định tắt cho mọi trang cho tới khi bạn bật.',
  waiting: '{count} bình luận chờ duyệt',
  who: 'Ai được bình luận',
  off: 'Tắt',
  offText: 'Không có phần bình luận dưới bài viết. Các bình luận đã có được giữ lại cho tới khi bạn bật lại.',
  members: 'Hội viên đã đăng nhập',
  membersText: 'Mọi hội viên đọc được bài viết',
  paid: 'Chỉ hội viên trả phí',
  paidText: 'Hội viên miễn phí đọc được bình luận và thấy liên kết nâng cấp',
  review: 'Duyệt',
  reviewAll: 'Mọi bình luận',
  reviewAllText: 'Không có gì hiển thị cho tới khi người kiểm duyệt bình luận đăng nó',
  reviewFirst: 'Bình luận đầu tiên của hội viên',
  reviewFirstText: 'Sau một bình luận đã đăng, bình luận của hội viên hiển thị ngay',
  reviewNone: 'Không duyệt',
  reviewNoneText: 'Bình luận hiển thị ngay',
  links: 'Dù bạn chọn gì, bình luận có từ 2 liên kết trở lên vẫn phải chờ duyệt.',
  moderatedBy: 'Người kiểm duyệt',
  moderatedByNote: 'những người có thể kiểm duyệt bình luận',
  nobody: 'Chưa có ai',
  perPost: 'Theo bài viết',
  perPostText: 'Mỗi bài viết có thể tắt bình luận trong cài đặt Web của nó',
  limits: 'Giới hạn',
  limitsText: '{chars} ký tự mỗi bình luận; 10 bình luận mỗi hội viên trong 10 phút và 50 mỗi ngày',
})
