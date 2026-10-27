import { translate } from '@/i18n'
import { m } from './messages'

/** Vietnamese for the words Sending providers & routing shares (docs/i18n/glossary-vi.md). */
export default translate(m, {
  title: 'Nhà cung cấp gửi thư và định tuyến',
  sectionsLabel: 'Các mục',
  tabProviders: 'Nhà cung cấp',
  tabRouting: 'Định tuyến và chuyển dự phòng',
  tabWarmup: 'Khởi động',
  tabTraffic: 'Quy tắc lưu lượng',
  tabEvents: 'Sự kiện và đồng bộ',
  tabAdapters: 'Adapter',
  channelButton: 'Kênh: {channel}. Đổi kênh',
  channelLabel: 'Kênh',

  streamBroadcast: 'Gửi hàng loạt',
  streamAutomation: 'Tự động hóa',
  streamSystem: 'Hệ thống',
  streamTransactional: 'Giao dịch',

  failoverAutomatic: 'tự động',
  failoverPauseAndAsk: 'tạm dừng và hỏi',
  failoverWarmOnly: 'tự động, chỉ sang nhà cung cấp đã khởi động xong',

  statusHealthy: 'Ổn định',
  statusWarmingUp: 'Đang khởi động',
  statusWarmingUpDay: 'Đang khởi động · ngày {day}/{days}',
  statusDown: 'Gián đoạn',
  statusDownSince: 'Gián đoạn · {since}',
  statusDownAuth: 'Gián đoạn · xác thực thất bại',
  statusDegraded: 'Suy giảm',
  statusDegradedSilent: 'Suy giảm · webhook im lặng {since}',
  statusSandbox: 'Sandbox · chỉ địa chỉ đã xác minh',
  statusPaused: 'Đã tạm dừng',

  notSupported: '<vh>Không hỗ trợ:</vh> {capability}',
  secondsAgo: '{n} giây trước',

  limitGuard: '{kind, select, bounce {Tỷ lệ trả thư · bảo vệ khởi động {limit}} other {Tỷ lệ khiếu nại · bảo vệ khởi động {limit}}}',
  limit: '{kind, select, bounce {Tỷ lệ trả thư · giới hạn {limit}} other {Tỷ lệ khiếu nại · giới hạn {limit}}}',
  limitNamed: '{kind, select, bounce {Tỷ lệ trả thư · giới hạn {limit} ({provider})} other {Tỷ lệ khiếu nại · giới hạn {limit} ({provider})}}',
  limitReview:
    '{kind, select, bounce {Tỷ lệ trả thư · giới hạn {limit} ({provider} xem xét ở mức {review})} other {Tỷ lệ khiếu nại · giới hạn {limit} ({provider} xem xét ở mức {review})}}',
  limitPause:
    '{kind, select, bounce {Tỷ lệ trả thư · giới hạn {limit} ({provider} tạm dừng ở mức {pause})} other {Tỷ lệ khiếu nại · giới hạn {limit} ({provider} tạm dừng ở mức {pause})}}',
  limitReviewPause:
    '{kind, select, bounce {Tỷ lệ trả thư · giới hạn {limit} ({provider} xem xét ở mức {review}, tạm dừng ở mức {pause})} other {Tỷ lệ khiếu nại · giới hạn {limit} ({provider} xem xét ở mức {review}, tạm dừng ở mức {pause})}}',
})
