import { translate } from '@/i18n'
import { mt } from './templates.messages'

export default translate(mt, {
  search: 'Tìm mẫu và phần đã lưu',
  found: '{count, plural, =0 {Không tìm thấy} other {Tìm thấy {count}}}',
  moreFor: 'Thêm thao tác cho {name}',
  rename: 'Đổi tên…',
  replace: 'Thay bằng email này',
  delete: 'Xóa…',
  renameTitle: 'Đổi tên "{name}"',
  name: 'Tên',
  description: 'Mô tả (không bắt buộc)',
  renamed: 'Đã đổi tên thành "{name}"',
  replaceTitle: 'Thay "{name}" bằng email này?',
  replaceBody: 'Các email mới tạo từ mẫu sẽ bắt đầu từ email đang mở. Các email đã tạo từ mẫu không thay đổi.',
  replaceBadge: 'Thay mẫu',
  replaceConfirm: 'Thay mẫu',
  replacing: 'Đang thay…',
  replaced: '"{name}" giờ chứa email này',
  deleteTitle: 'Xóa "{name}"?',
  deleteTemplateBody: 'Mẫu rời khỏi thư viện. Các email đã tạo từ mẫu không thay đổi.',
  deleteSectionBody: 'Phần này rời khỏi danh sách phần đã lưu. Các email đang dùng nó không thay đổi.',
  deleteBadge: 'Xóa',
  deleteConfirm: 'Xóa',
  deleting: 'Đang xóa…',
  deleted: 'Đã xóa "{name}"',
})
