import { translate } from '@/i18n'
import { m } from './formfields.messages'

export default translate(m, {
  phone: 'Số di động',
  phoneOptional: 'Số di động · không bắt buộc',
  phoneRequired: 'Số di động · bắt buộc',
  textsTitle: 'Tin nhắn',
  smsConsent: 'Ô đồng ý nhận SMS · mặc định không đánh dấu',
  needsPhone: 'Cần trường Số di động.',
  consentText: 'Nội dung đồng ý nhận SMS',
  evidence: 'Số điện thoại, các danh sách, đúng nội dung này cùng phiên bản của nó, trang, địa chỉ IP và thời điểm được lưu làm bằng chứng đồng ý nhận SMS.',
  newVersion: 'Lưu sẽ bắt đầu phiên bản {version}; những người đăng ký trước giữ nội dung họ đã đồng ý.',
  separate: 'Đồng ý nhận email không bao giờ đăng ký ai nhận tin nhắn: ô này là ô riêng.',
  confirmation: 'Sau khi đánh dấu ô này, người đó nhận tin xác nhận đăng ký từ Định tuyến và quy tắc.',
  // The words the server's Vietnamese consent text links (texting signup.termsWords).
  termsWords: 'điều khoản tin nhắn',
})
