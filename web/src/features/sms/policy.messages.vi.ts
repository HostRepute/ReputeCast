import { translate } from '@/i18n'
import { m } from './policy.messages'

export default translate(m, {
  costTitle: 'Chiến dịch SMS hoặc WhatsApp có chi phí ước tính ít nhất',
  costDetail: 'Tính từ giá của các nhà cung cấp cho người nhận của chiến dịch. Với tin nhắn, “quyền gửi” là sms:send, thêm vào campaigns:send.',
  costAmount: 'Chi phí ước tính tối thiểu, tính bằng đô la',
  costInvalid: 'Hãy nhập một số tiền lớn hơn $0.',
})
