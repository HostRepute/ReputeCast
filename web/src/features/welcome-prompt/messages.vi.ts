import { translate } from '@/i18n'
import { m } from './messages'

export default translate(m, {
  label: 'Gợi ý chuỗi chào mừng',
  title: 'Người đăng ký mới của {list} chưa nhận được email chào mừng nào',
  body: 'Mẫu khởi đầu Chuỗi chào mừng gửi ba email trong tuần đầu tiên.',
  covered: '{count, plural, other {{lists} đã có chuỗi chào mừng.}}',
  start: 'Bắt đầu chuỗi chào mừng',
  notNow: 'Để sau',
  dismissed: 'Sẽ không gợi ý chuỗi chào mừng cho {list} nữa',
  dismissedDetail: 'Bạn có thể bắt đầu một chuỗi bất cứ lúc nào từ Tự động hóa › Mẫu khởi đầu.',
  dismissFailed: 'Không ẩn được gợi ý',
})
