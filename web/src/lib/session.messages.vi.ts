import { translate } from '@/i18n'
import { m } from './session.messages'

export default translate(m, {
  denied: 'Để {action}, bạn cần quyền {permission}.',
  deniedAsk: 'Để {action}, bạn cần quyền {permission}. Hãy hỏi {helpers}.',
  this: 'thực hiện thao tác này',
  'campaigns:send': 'gửi chiến dịch',
  'campaigns:pause': 'tạm dừng lần gửi',
  'campaigns:approve': 'phê duyệt chiến dịch',
  'campaigns:edit': 'sửa chiến dịch',
  'subscribers:export': 'xuất liên hệ',
  'subscribers:import': 'nhập liên hệ',
  'subscribers:delete': 'xóa dữ liệu cá nhân',
  'providers:manage': 'quản lý nhà cung cấp',
  'domains:manage': 'quản lý tên miền',
  'team:invite': 'mời thành viên',
  'team:manage': 'quản lý nhóm',
  'segments:sql': 'viết phân khúc SQL',
  'apikeys:manage': 'quản lý khóa API',
  'mcp:manage': 'quản lý tác tử MCP',
  'automations:publish': 'xuất bản tự động hóa',
  'workspace:manage': 'thay đổi cài đặt không gian làm việc',
  'webhooks:manage': 'quản lý webhook',
  'integrations:hostrepute': 'kết nối HostRepute',
  'forms:edit': 'sửa biểu mẫu và trang công khai',
  'reports:share': 'chia sẻ báo cáo',
})
