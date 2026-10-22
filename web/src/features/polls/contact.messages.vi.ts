import { translate } from '@/i18n'
import { m } from './contact.messages'

/** Vietnamese for poll answers and survey responses on the contact profile (docs/i18n/glossary-vi.md). */
export default translate(m, {
  answered: 'Đã trả lời {answer} cho “{question}” trong {campaign}',
  answeredComment: 'Đã trả lời {answer} cho “{question}” trong {campaign} · kèm nhận xét',
  changed: 'đổi từ {previous}',
  quote: '“{text}”',
  responded: 'Đã trả lời khảo sát “{name}” · đã trả lời {answered}/{questions} câu hỏi',
  started: 'Đã bắt đầu khảo sát “{name}” · đã trả lời {answered}/{questions} câu hỏi',
  editable: 'có thể đổi câu trả lời đến {date}',
  fromPoll: 'từ một cuộc bình chọn, {date}',
  fromSurvey: 'từ một khảo sát, {date}',
})
