import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** Vietnamese for the automation trigger "Answered a poll or survey" (docs/i18n/glossary-vi.md). */
export default translate(m, {
  source: 'Bình chọn hoặc khảo sát',
  pick: 'Chọn một bình chọn hoặc khảo sát',
  polls: 'Bình chọn',
  surveys: 'Khảo sát',
  pollIn: '{question} · {campaign}',
  answer: 'Câu trả lời',
  any: 'Bất kỳ câu trả lời nào',
  is: 'Là một trong',
  isNot: 'Không phải một trong',
  answers: 'Câu trả lời',
  onChange: 'Bắt đầu lại khi câu trả lời thay đổi',
  note: 'Câu trả lời thử và câu trả lời từ người không phải liên hệ không bao giờ bắt đầu luồng.',
  surveyNote: 'Khảo sát bắt đầu luồng khi có người gửi khảo sát.',
  loading: 'Đang tải bình chọn và khảo sát…',
})
