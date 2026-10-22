import { translate } from '@/i18n'
import { mpc } from './contacts.messages'

/** Vietnamese for paid status in Contacts and the contact profile (docs/i18n/glossary-vi.md › Earn). */
export default translate(mpc, {
  paidStatus: 'Trạng thái trả phí',
  anyPaidStatus: 'Mọi trạng thái trả phí',
  withAccess: 'Mọi gói trả phí · có quyền truy cập',
  andAbove: 'Từ {tier} trở lên',
  trialing: 'Đang dùng thử',
  pastDue: 'Quá hạn',
  complimentary: 'Được miễn phí',
  gift: 'Quà tặng',
  ended: 'Đã kết thúc',
  free: 'Chỉ người đăng ký miễn phí',
  tierTitle: 'Gói trả phí',
  colPaid: 'Trả phí',
  // The contact profile's Paid subscription card.
  cardTitle: 'Thuê bao trả phí',
  cardSince: 'Từ {date} · trạng thái trả phí lấy từ Stripe',
  cardSinceGrant: 'Từ {date} · quyền truy cập miễn phí do {publisher} cấp',
  cardSinceGift: 'Từ {date} · quà tặng',
  cardEnded: 'Hết quyền truy cập ngày {date}',
  tier: 'Gói',
  price: 'Giá',
  status: 'Trạng thái',
  renews: 'Gia hạn',
  ends: 'Kết thúc',
  ltv: 'Giá trị trọn đời',
  source: 'Nguồn',
  openStripe: 'Mở trong Stripe',
  openMember: 'Mở hội viên trả phí',
  giveComp: 'Cấp quyền truy cập miễn phí',
  cancel: 'Hủy thuê bao',
  resend: 'Gửi lại email hội viên',
  tierPeriod: '{tier} · {period, select, month {hằng tháng} year {hằng năm} other {}}',
  // The Activity filter.
  activityPaid: 'Trả phí',
  noPaidActivity: 'Chưa có sự kiện trả phí nào.',
  // The segment builder.
  segmentNote: 'Các điều kiện trả phí đọc trạng thái trả phí từ sự kiện của Stripe. Hội viên trả phí thử nghiệm không bao giờ được tính.',
})
