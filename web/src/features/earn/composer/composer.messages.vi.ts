import { translate } from '@/i18n'
import { mc } from './composer.messages'

/**
 * Vietnamese for paid content in the composer: Visibility · Paid access, the
 * Upgrade block and paid-only campaigns (docs/i18n/glossary-vi.md › Earn).
 */
export default translate(mc, {
  // Visibility · Paid access.
  kind: 'Quyền truy cập trả phí',
  group: 'Quyền truy cập trả phí',
  kindNote: 'Mọi gói trả phí, từ một gói trở lên, hoặc chỉ người đăng ký miễn phí',
  row: 'Quyền truy cập trả phí',
  opAny: 'mọi gói trả phí',
  opTier: 'từ một gói trở lên',
  opFree: 'chỉ người đăng ký miễn phí',
  show: 'Quyền truy cập trả phí',
  anyTier: 'Mọi gói trả phí',
  tierAndAbove: 'Từ một gói trở lên',
  freeOnly: 'Chỉ người đăng ký miễn phí',
  tierChoice: 'Từ {tier} trở lên',
  tierLabel: 'Gói',
  pickTier: 'Chọn một gói',
  noTiers: 'Hãy tạo một gói trong Kiếm tiền › Gói và ưu đãi trước.',
  includes: 'Quyền truy cập trả phí bao gồm người đang dùng thử, người được miễn phí, người nhận quà tặng, và cả người quá hạn trong lúc Stripe thử lại.',
  includesTier: '{tier} bao gồm nội dung trả phí của {lower}.',
  freeDefault: 'Quyền truy cập trả phí · Chỉ người đăng ký miễn phí, mặc định',
  // The token menu's Paid group: paid.* and the reader's two links.
  tokensPaid: 'Trả phí',
  // The Upgrade block.
  blockLabel: 'Nâng cấp',
  blocksGroup: 'Kiếm tiền',
  blocksHint: 'Khối Tài trợ tự điền nội dung từ lượt đặt chỗ của số này. Khối Nâng cấp chỉ hiện với người đăng ký miễn phí, trừ khi bạn đổi phần Hiển thị của nó.',
  upgradeTitle: 'Khối Nâng cấp',
  // The Upgrade block's offer (X17).
  offer: 'Ưu đãi',
  offerNone: 'Không có',
  offerEnds: '{offer} kết thúc ngày {date}. Sau đó khối hiện giá thường.',
  offerNoneLive: 'Không có ưu đãi đăng ký nào áp dụng cho gói này. Ưu đãi nằm trong Kiếm tiền › Gói và ưu đãi.',
  upgradeSub: 'Thẻ tường phí · liên kết đến trang nâng cấp',
  headline: 'Dòng tiêu đề',
  line: 'Dòng mô tả',
  tierPrice: 'Gói, giá và dùng thử',
  tierPriceHint: 'Hiện “{price}”, lấy từ gói. Dòng này đổi theo khi gói thay đổi.',
  tierPriceNone: 'Gói này chưa có giá. Hãy thêm giá trong Kiếm tiền › Gói và ưu đãi.',
  button: 'Nút',
  buttonHint: 'Liên kết đến {token}, liên kết này nhận ra người đọc và ghi nhận cho email này.',
  notConnected: 'Khối Nâng cấp bị ẩn cho đến khi bạn kết nối nhà cung cấp thanh toán.',
  connect: 'Kết nối Stripe',
  defaultHeadline: 'Nhận nhiều hơn với {tier}',
  defaultLine: 'Hội viên trả phí được đọc trọn vẹn mọi số, kho lưu trữ và nội dung bổ sung.',
  defaultButton: 'Bắt đầu dùng thử miễn phí',
  defaultButtonNoTrial: 'Nâng cấp lên {tier}',
  // Paid-only campaigns: Who on this list.
  who: 'Ai trong danh sách này',
  whoEveryone: 'Mọi người',
  whoPaid: 'Chỉ hội viên trả phí',
  whoFree: 'Chỉ người đăng ký miễn phí',
  paidTier: 'Gói trả phí',
  anyPaidTier: 'Mọi gói trả phí',
  recipients: '{count, plural, other {# người nhận}}',
  withAccess: '{count, number} người có quyền truy cập trả phí.',
  leftOut: 'Không bao gồm: {list}.',
  leftUnsubscribed: '{count, number} người đã hủy đăng ký email của {list}',
  leftNotOn: '{count, number} người không có trong {list}',
  leftSuppressed: '{count, number} người bị chặn gửi',
  leftInactive: '{count, number} người không nhận được email',
  includesAudience:
    'Bao gồm người đang dùng thử, người được miễn phí và người nhận quà tặng{pastDue, plural, =0 {} other {, cùng # người quá hạn trong lúc Stripe thử lại}}. Không bao giờ bao gồm hội viên trả phí thử nghiệm.',
  oneList: '“Chỉ hội viên trả phí” chỉ dùng được với một danh sách mỗi lần.',
  paidOnly: 'Chỉ hội viên trả phí',
  paidOnlyFilter: 'Chỉ hội viên trả phí · {count}',
})
