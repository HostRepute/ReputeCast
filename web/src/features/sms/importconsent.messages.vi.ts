import { translate } from '@/i18n'
import { m } from './importconsent.messages'

export default translate(m, {
  title: 'Đồng ý nhận SMS',
  withEvidence: 'Nhập kèm bằng chứng',
  withEvidenceText:
    'Những dòng ghi có, kèm ngày, nguồn và nội dung đồng ý, sẽ thành Đã đăng ký nhận tin nhắn trên danh sách đích, được ghi là đã nhập cùng bằng chứng đó. Không ai bị nhắn tin.',
  missing:
    'Đồng ý nhận SMS cần bằng chứng: hãy ghép cột {missing} trong Ghép trường. Cho đến lúc đó, số điện thoại vẫn được nhập và đồng ý nhận SMS giữ ở Chưa đăng ký.',
  numbersOnly:
    'Số điện thoại được nhập không kèm đồng ý nhận SMS: chúng giữ ở Chưa đăng ký nhận tin nhắn. Để nhập cả đồng ý nhận SMS, hãy ghép một cột đồng ý cùng ngày, nguồn và nội dung trong Ghép trường.',
  date: 'ngày',
  source: 'nguồn',
  text: 'nội dung',
  mapFields: 'Ghép trường',
})
