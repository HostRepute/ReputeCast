import { translate } from '@/i18n'
import { m } from './settings.messages'

export default translate(m, {
  title: 'Nhà cung cấp SMS',
  senders: '{count, plural, other {# người gửi}} · {kinds}',
  noSenders: 'Chưa có người gửi',
  routes: 'Tuyến',
  notRouted: 'Chưa có tuyến',
  held: 'Tạm giữ: {countries}',
  sendersLabel: 'Người gửi',
  connected: 'Đã kết nối',
  connectedOn: '{date} · {name}',
  cost: 'Chi phí · {month}',
  budget: 'Ngân sách SMS · {month}',
  spentOf: '<b>{spent}</b> trên {budget}',
  alerts: 'Cảnh báo ở {alert}% và 100%. Ở 100%, chiến dịch không bắt đầu được và các bước SMS phải chờ đến khi người có quyền providers:manage nâng ngân sách.',
  noBudget: 'Chưa có ngân sách hằng tháng: đã chi {spend} trong {month}.',
  foot: 'Người gửi, tuyến theo quốc gia, giờ yên lặng, quy tắc và ngân sách nằm ở Nhà cung cấp gửi thư › SMS.',
  changeBudget: 'Đổi ngân sách',
  open: 'Mở nhà cung cấp SMS',
})
