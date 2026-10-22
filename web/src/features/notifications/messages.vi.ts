import { translate } from '@/i18n'
import { m } from './messages'

export default translate(m, {
  title: 'Thông báo',
  account: 'Tài khoản và bảo mật',
  filterLabel: 'Bộ lọc',
  filter:
    '{filter, select, unread {Chưa đọc} approvals {Phê duyệt} sending {Gửi thư} providers {Nhà cung cấp} team {Nhóm} hostrepute {HostRepute} other {Tất cả}}',
  unreadCount: 'Chưa đọc · {count}',

  listTitle: 'Thông báo',
  listCaption: 'Mỗi thông báo dẫn đến màn hình nơi bạn có thể xử lý',
  markAll: 'Đánh dấu tất cả là đã đọc',
  allRead: 'Mọi thông báo đều đã đọc',
  loadFailed: 'Không tải được thông báo',
  nothingMarked: 'Không có thông báo nào được đánh dấu là đã đọc.',
  loading: 'Đang tải thông báo…',
  caughtUp: 'Bạn đã xem hết',
  nothingUnread: 'Không có thông báo chưa đọc',
  noneInCategory:
    '{filter, select, approvals {Không có thông báo nào về phê duyệt} sending {Không có thông báo nào về gửi thư} providers {Không có thông báo nào về nhà cung cấp} team {Không có thông báo nào về nhóm} hostrepute {Không có thông báo HostRepute nào} other {Không có thông báo nào}}',
  emptyBody: 'Phê duyệt, lần tự tạm dừng, sự cố nhà cung cấp, thay đổi trong nhóm và cảnh báo HostRepute sẽ hiện ở đây.',
  category:
    '{category, select, approvals {Phê duyệt} sending {Gửi thư} providers {Nhà cung cấp} team {Nhóm} hostrepute {HostRepute} integrations {Tích hợp} transactional {Giao dịch} ai {AI} earn {Kiếm tiền} website {Trang web} grow {Tăng trưởng} other {Khác}}',
  new: 'Mới',
  retention: 'Thông báo cũ hơn được giữ trong {days, plural, other {# ngày}}.',

  prefsTitle: 'Tùy chọn của tôi',
  prefsLoadFailed: 'Không tải được tùy chọn của bạn',
  prefsCaptionVia: 'Theo từng loại sự kiện · email được gửi trên luồng Hệ thống qua {provider}',
  prefsCaption: 'Theo từng loại sự kiện · email sẽ bắt đầu được gửi khi có nhà cung cấp gửi thư xử lý luồng Hệ thống',
  prefsTable: 'Tùy chọn thông báo',
  colEvent: 'Sự kiện',
  colInApp: 'Trong ứng dụng',
  colEmail: 'Email',
  inAppFor: 'Trong ứng dụng cho {event}',
  emailFor: 'Email cho {event}',
  quietHours: 'Giờ yên lặng · chỉ email',
  quietStart: 'Bắt đầu giờ yên lặng',
  quietEnd: 'Kết thúc giờ yên lặng',
  quietRange: '{start} <k>đến</k> {end}',
  quietUrgent: 'Thông báo tự tạm dừng và sự cố nhà cung cấp luôn được gửi ngay.',
  digest: 'Email tổng hợp',
  digestImmediate: 'Ngay lập tức',
  digestHourly: 'Tổng hợp mỗi giờ',
  digestDaily: 'Tổng hợp hằng ngày lúc {time}',
  emailOff: 'Email đang tắt',
  emailOffBody:
    'Chưa có nhà cung cấp gửi thư nào xử lý luồng Hệ thống, nên thông báo chỉ hiện trong ứng dụng. Lựa chọn email của bạn sẽ có hiệu lực khi có nhà cung cấp.',
  notSaved: 'Chưa lưu',
  save: 'Lưu tùy chọn',
  saved: 'Đã lưu tùy chọn',
})
