import { translate } from '@/i18n'
import { mpi } from './entries.messages'

/**
 * Vietnamese for the Payments group of Settings › Integrations and the paid
 * members hint in Import contacts (docs/i18n/glossary-vi.md › Earn).
 */
export default translate(mpi, {
  stripe: 'Stripe',
  paddle: 'Paddle',
  lemonsqueezy: 'Lemon Squeezy',
  byOwnStripe: 'Tài khoản Stripe của chính bạn',
  byMerchant: 'Bên bán hàng thay bạn',
  stripeLine: 'Gói trả phí và thanh toán một lần, kèm cổng khách hàng cho người đọc của bạn.',
  merchantLine: 'Bên bán hàng thay bạn: thay mặt bạn bán cho người đọc, và lo thuế bán hàng cùng VAT.',
  capSubscriptions: 'Thuê bao',
  capOneTime: 'Thanh toán một lần',
  capPortal: 'Cổng khách hàng',
  capTestMode: 'Chế độ thử nghiệm',
  capTax: 'Tính thuế',
  factRestricted: 'Khóa hạn chế, không bao giờ là khóa bí mật',
  factConnected: '{account} · {mode, select, live {Chính thức} other {Thử nghiệm}}',
  factNotInRelease: 'Chưa có trong bản phát hành này',
  notAvailable: 'Chưa có sẵn',
  connect: 'Kết nối',
  manage: 'Quản lý',
  view: 'Xem',
  importHint: 'Chuyển hội viên trả phí từ Substack, Ghost, beehiiv hoặc Kit? <a>Hãy nhập họ trong Kiếm tiền</a>, khớp với các thuê bao Stripe của bạn.',
})
