import { translate } from '@/i18n'
import { m } from './deactivate.messages'

export default translate(m, {
  eyebrow: 'Hộp thoại · vô hiệu hóa thành viên',
  titleOne: 'Vô hiệu hóa {name}?',
  titleMany: 'Vô hiệu hóa {count, plural, other {# thành viên}}?',
  descriptionOne:
    '{name} vẫn giữ lịch sử của mình và có thể được kích hoạt lại sau. Nếu xóa thay vì vô hiệu hóa, tư cách thành viên sẽ bị xóa; nhật ký kiểm tra vẫn ghi tên họ.',
  descriptionMany: 'Họ vẫn giữ lịch sử của mình và có thể được kích hoạt lại sau. Nhật ký kiểm tra vẫn ghi tên họ.',
  loadingChanges: 'Đang tải những gì sẽ thay đổi…',
  confirmOne: 'Vô hiệu hóa {name}',
  confirmMany: 'Vô hiệu hóa {count, plural, other {# thành viên}}',
  doneOne: 'Đã vô hiệu hóa {name}',
  doneMany: '{count, plural, other {Đã vô hiệu hóa # thành viên}}',
  doneDetail: '{count, plural, other {đã đăng xuất khỏi # phiên}}',
  failedOne: 'Không vô hiệu hóa được {name}',
  failedMany: '{count, plural, other {Không vô hiệu hóa được # thành viên}}',

  signedOutEverywhere: '{count, plural, other {Bị đăng xuất khỏi mọi nơi ngay lập tức (# phiên)}}',
  cantSignIn: 'Không đăng nhập được vào {workspace}; các không gian làm việc khác không bị ảnh hưởng',
  scheduled: '{count, plural, other {# chiến dịch đã lên lịch của họ vẫn giữ lịch và chạy dưới danh nghĩa không gian làm việc}}',
  approvals: '{count, plural, other {# yêu cầu phê duyệt đang chờ vẫn mở cho người phê duyệt}}',

  keysLegend: 'Khóa API họ đã tạo · {count}',
  keepKey: 'Giữ {keys} tiếp tục hoạt động <k>({permissions})</k>',
  keepKeys: 'Giữ {keys} tiếp tục hoạt động',
  revoke: '{count, plural, other {Thu hồi ngay}}',

  reason: 'Lý do <k>· được ghi vào nhật ký kiểm tra</k>',
  reasonPlaceholder: 'Hợp đồng đã kết thúc',
})
