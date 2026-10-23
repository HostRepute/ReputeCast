import { translate } from '@/i18n'
import { ma, mf, mp, mr } from './elsewhere.messages'

/**
 * Vietnamese for digital products in other screens: the contact profile,
 * the campaign report, the composer's Product card and the automation
 * trigger (docs/i18n/glossary-vi.md). An automation's flow is "luồng".
 */
const purchases = translate(mp, {
  title: 'Lượt mua · {count}',
  spent: 'Đã chi {amount}',
  line: '{date} · {amount} · đơn hàng {number}',
  linePwyw: '{date} · {amount}, trả tùy ý · đơn hàng {number}',
  orderLink: '{product}, đơn hàng {number}',
  linkExpired: 'Liên kết hết hạn {date}',
  sendLink: 'Gửi liên kết mới',
  sendLinkAction: 'gửi liên kết mới',
  orderError: 'Không mở được đơn hàng {number}',
  error: 'Không tải được các lượt mua',
  retry: 'Thử lại',
  // The timeline's events.
  bought: '<b>Đã mua</b> {product}',
  refunded: '<b>Đã hoàn tiền</b> {product}',
  // The consent history: a buyer who ticked the list offer at checkout.
  checkoutTicked: '{source}: đã đánh dấu “{text}” vào {when}',
  // The contacts list's filter.
  filterLabel: 'Đã mua sản phẩm',
  filterAll: 'Mọi lịch sử mua hàng',
  filterAny: 'Đã mua bất kỳ sản phẩm nào',
  filterProduct: 'Đã mua {product}',
  filterUnknown: 'Đã mua một sản phẩm',
})

const report = translate(mr, {
  title: 'Doanh số sản phẩm',
  caption: 'Sản phẩm số được mua sau một lượt nhấp trong email này',
  products: 'Sản phẩm',
  colProduct: 'Sản phẩm',
  colOrders: 'Đơn hàng',
  colRevenue: 'Doanh thu',
  orders: '{count, plural, other {# đơn hàng}}',
  total: 'Tổng cộng',
  note: 'Một đơn hàng được tính cho email mà người mua nhấp gần nhất, tối đa {days, plural, other {# ngày}} trước khi thanh toán. Doanh thu đã trừ hoàn tiền và chưa trừ phí Stripe.',
})

const fill = translate(mf, {
  title: 'Điền từ một sản phẩm',
  product: 'Sản phẩm đã xuất bản',
  pick: 'Chọn một sản phẩm đã xuất bản',
  option: '{name} · {price}',
  pwywFrom: 'Trả tùy ý · từ {min}',
  fill: 'Điền vào thẻ',
  filling: 'Đang điền…',
  hint: 'Điền hình ảnh và văn bản thay thế, tên, giá, và liên kết của nút đến trang sản phẩm.',
  none: 'Chưa có sản phẩm nào được xuất bản. Hãy xuất bản một sản phẩm trong Kiếm tiền › Sản phẩm để điền thẻ này từ đó.',
  loading: 'Đang tải sản phẩm…',
  filled: 'Thẻ giờ hiển thị {name}',
  failed: 'Không điền được thẻ',
})

const automation = translate(ma, {
  trigger: 'Mua sản phẩm',
  condition: 'Đã mua sản phẩm',
  product: 'Sản phẩm',
  any: 'Mọi sản phẩm',
  deleted: 'Một sản phẩm đã bị xóa',
  loading: 'Đang tải…',
  anyRef: 'mọi sản phẩm',
  someRef: 'một sản phẩm',
  phrase: 'đã mua {product}',
  phraseAny: 'đã mua một sản phẩm',
  goal: 'Đã mua {product}',
  goalAny: 'Đã mua một sản phẩm',
  triggerNote: 'Chỉ liên hệ mới vào luồng: người mua không phải liên hệ thì không bao giờ vào. Khi đơn hàng được hoàn tiền, liên hệ rời khỏi luồng.',
  liquid: 'Email trong luồng này dùng được {name}, {amount} và {number}.',
  conditionNote: 'Tính các đơn hàng đã thanh toán kể từ khi liên hệ vào luồng. Đơn hàng được hoàn tiền toàn bộ không được tính.',
})

export default { ...purchases, ...report, ...fill, ...automation }
