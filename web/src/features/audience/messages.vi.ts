import { translate } from '@/i18n'
import { m } from './messages'

/**
 * Vietnamese for the audience screens' shared words (docs/i18n/glossary-vi.md).
 * Engagement tiers: Tích cực · Đang nguội · Có nguy cơ · Ngừng gửi.
 */
export default translate(m, {
  // ── Section tabs ─────────────────────────────────────────────────────
  title: 'Người đăng ký',
  sections: 'Các mục người đăng ký',
  tabContacts: 'Liên hệ',
  tabContactsCount: 'Liên hệ · {count}',
  tabSegments: 'Phân khúc',
  tabSegmentsCount: 'Phân khúc · {count}',
  tabHealth: 'Sức khỏe danh sách',
  tabSuppressions: 'Danh sách chặn gửi',
  tabSuppressionsCount: 'Danh sách chặn gửi · {count}',

  // ── Engagement tiers ─────────────────────────────────────────────────
  tierActive: 'Tích cực',
  tierCooling: 'Đang nguội',
  tierAtRisk: 'Có nguy cơ',
  tierSunset: 'Ngừng gửi',
  windowActive: 'có mở thư trong 30 ngày',
  windowCooling: '31 đến 90 ngày',
  windowAtRisk: '91 đến 180 ngày',
  windowSunset: 'không mở thư trong 180 ngày',

  // ── Consent ──────────────────────────────────────────────────────────
  consentDouble: 'Xác nhận hai bước',
  consentSingle: 'Xác nhận một bước',
  consentPending: 'Đã nhập · chờ xác nhận',
  consentPendingFilter: 'Đã nhập, chờ xác nhận',

  // ── Mailbox providers ────────────────────────────────────────────────
  mailboxOther: 'Khác',
})
