import { translate } from '@/i18n'
import { ps } from './sales.messages'

/** Vietnamese for Earn › Overview's "Product sales" card (docs/i18n/glossary-vi.md). */
export default translate(ps, {
  title: 'Doanh số sản phẩm · 30 ngày',
  products: 'Sản phẩm',
  orders: '{count, plural, other {đơn hàng}}',
  gross: 'tổng thu',
  refunded: 'đã hoàn tiền · {count, plural, other {# đơn hàng}}',
  note: 'Thực thu {net} từ {products, plural, other {# sản phẩm}}, chưa trừ phí Stripe. ReputeCast không thu phí.',
})
