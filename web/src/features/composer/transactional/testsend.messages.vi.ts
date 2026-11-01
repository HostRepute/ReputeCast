import { translate } from '@/i18n'
import { mts } from './testsend.messages'

export default translate(mts, {
  menuItem: 'Gửi email thử…',
  title: 'Gửi thử {key}',
  sendTo: 'Gửi đến',
  ownOnly: 'Không có quyền transactional:send, bạn chỉ gửi thử được đến địa chỉ của mình.',
  fillWith: 'Điền bằng',
  samples: 'Giá trị mẫu',
  testData: 'Dữ liệu thử · {name}',
  note: 'Bản nháp hiện tại, gửi qua tuyến System, có [Test] trong tiêu đề. Không được tính vào số liệu của nhật ký.',
  send: 'Gửi thử',
  sending: 'Đang gửi…',
  sent: 'Đã gửi thử đến {email}',
  sentWith: 'Đã gửi thử đến {email} với "{name}"',
})
