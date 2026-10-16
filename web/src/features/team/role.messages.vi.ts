import { translate } from '@/i18n'
import { m } from './role.messages'

export default translate(m, {
  eyebrow: 'Hộp thoại · đổi vai trò hoặc phạm vi',
  title: 'Đổi vai trò hoặc phạm vi của {name}',
  description: 'Thay đổi có hiệu lực ngay, từ yêu cầu tiếp theo của {name}. Cả giá trị cũ và mới đều được ghi vào nhật ký kiểm tra.',
  nothingChanged: 'Chưa có gì thay đổi.',
  current: '{role} (hiện tại)',
  updated: 'Đã cập nhật {name}',
  scopeChanged: 'đã đổi phạm vi',
  nowRole: 'giờ là {role}',
  ssoGroup:
    'Vai trò của {name} đến từ ánh xạ nhóm SSO. Thay đổi ở đây chỉ giữ đến lần đăng nhập tiếp theo, khi ánh xạ được áp dụng lại; hãy đổi ánh xạ để thay đổi lâu dài.',
  allScope: 'Mọi danh sách và tên miền người gửi',

  lastOwnerEyebrow: 'Hộp thoại · đổi vai trò · bị chặn',
  lastOwnerTitle: '{name} là Chủ sở hữu cuối cùng',
  onlyOwnerTransfers: 'Chỉ Chủ sở hữu mới chuyển được quyền sở hữu.',
  transfer: 'Chuyển quyền sở hữu',
  cantDemote: 'Không thể hạ vai trò',
  keepsOwner:
    'Một không gian làm việc luôn cần ít nhất một Chủ sở hữu, nên không thể đổi vai trò này thành Quản trị viên, cũng không thể xóa hay vô hiệu hóa thành viên này.',
  transferFirst: 'Để thay đổi, trước tiên hãy chuyển quyền sở hữu cho một thành viên khác đã bật xác thực hai lớp. Sau đó {name} sẽ tự động trở thành Quản trị viên.',
  transferFirstSuggest:
    'Để thay đổi, trước tiên hãy chuyển quyền sở hữu cho một thành viên khác đã bật xác thực hai lớp, ví dụ {suggest}. Sau đó {name} sẽ tự động trở thành Quản trị viên.',
})
