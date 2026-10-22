import { translate } from '@/i18n'
import { m } from './messages'

export default translate(m, {
  title: 'Quản trị hệ thống',
  documentTitle: '{page} · Quản trị hệ thống',
  adminsOnly: 'chỉ quản trị viên',
  workspaceView: 'Xem ở cấp không gian làm việc',
  workspaceSettings: 'Cài đặt không gian làm việc',
  tabsLabel: 'Các mục của Quản trị hệ thống',

  notAdminBadge: '403 · chỉ quản trị viên hệ thống',
  notAdminTitle: 'Chỉ quản trị viên hệ thống mới mở được Quản trị hệ thống',
  backToOverview: 'Quay lại Tổng quan',
  account: 'Tài khoản và bảo mật',
  notAdminFootnote: 'Hãy nhờ quản trị viên hệ thống nếu cần thay đổi đăng nhập, SSO hoặc không gian làm việc.',
  notAdminBody:
    'Quản trị hệ thống quản lý người dùng, phương thức đăng nhập, SSO, chính sách bảo mật và không gian làm việc cho toàn hệ thống. Phương thức đăng nhập của riêng bạn nằm trong Tài khoản và bảo mật.',
})
