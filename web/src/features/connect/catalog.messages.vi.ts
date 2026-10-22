import { translate } from '@/i18n'
import { mc } from './catalog.messages'

/** Vietnamese for the provider catalog (docs/i18n/glossary-vi.md). */
export default translate(mc, {
  heading: 'Nhà cung cấp nào sẽ gửi thư của bạn?',
  lead: 'Mỗi adapter cho biết nó làm được gì. Các nhãn lấy từ manifest của từng adapter; điều khoản được xem lại mỗi quý.',
  searchPlaceholder: 'Tên nhà cung cấp',
  filterBulk: 'Cho phép tiếp thị hàng loạt',
  filterWebhooks: 'Có webhook báo thư trả lại',
  filterDedicated: 'IP riêng',
  counts: '{builtin} có sẵn · {community} của cộng đồng',
  loadFailed: 'Không tải được danh sách nhà cung cấp',
  noMatch: 'Không có nhà cung cấp nào phù hợp',
  clearFilters: 'Xóa bộ lọc',
  noMatchSearch: 'Không có nhà cung cấp nào tên “{query}” với các bộ lọc bạn đã chọn.',
  noMatchFilters: 'Không có nhà cung cấp nào có đủ mọi khả năng bạn đã chọn.',
  notSupported: '<vh>Không hỗ trợ:</vh> {capability}',
  connectProvider: 'Kết nối {provider}',
  communityTitle: 'Adapter cộng đồng',
  communityLine: 'Được xây bằng adapter SDK. Hãy xem người duy trì và phiên bản trước khi kết nối.',
  manageAdapters: 'Quản lý adapter',
  configure: 'Cấu hình',
  connect: 'Kết nối',
  loading: 'Đang tải danh sách nhà cung cấp…',
})
