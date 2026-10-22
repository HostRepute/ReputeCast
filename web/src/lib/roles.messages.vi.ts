import { translate } from '@/i18n'
import { m } from './roles.messages'

export default translate(m, {
  owner: 'Chủ sở hữu',
  admin: 'Quản trị viên',
  editor: 'Biên tập viên',
  viewer: 'Người xem',
  // Client roles (glossary › Agency accounts).
  clientApprover: 'Người duyệt phía khách hàng',
  clientViewer: 'Người xem phía khách hàng',
})
