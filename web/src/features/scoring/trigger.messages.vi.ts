import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** Vietnamese for the automation trigger "Score crosses…" (docs/i18n/glossary-vi.md). */
export default translate(m, {
  kind: 'Điểm vượt ngưỡng…',
  risesTo: 'Tăng lên {threshold} trở lên',
  fallsBelow: 'Giảm xuống dưới {threshold}',
  oncePer: 'Tối đa một lần cho mỗi liên hệ trong {days, plural, other {# ngày}}',
  startsWhen: 'Bắt đầu khi điểm',
  rises: 'Tăng lên',
  falls: 'Giảm xuống dưới',
  direction: 'Chiều',
  threshold: 'Điểm',
  orMore: 'trở lên · {band} bắt đầu từ {from}',
  below: 'trở xuống · {band} bắt đầu từ {from}',
  orMorePlain: 'trở lên',
  belowPlain: 'trở xuống',
  bandHot: 'Nóng',
  bandWarm: 'Ấm',
  oncePerLabel: 'Tối đa một lần cho mỗi liên hệ trong',
  days: 'ngày',
  explain:
    'Bắt đầu khi điểm thay đổi do hoạt động và giảm dần. Với những liên hệ vượt ngưỡng chỉ vì một quy tắc chấm điểm thay đổi, bạn sẽ được hỏi khi lưu quy tắc; mặc định là không bắt đầu.',
  needs: 'Để chọn trình kích hoạt này, bạn cần quyền automations:edit.',
  noScoring: 'Chấm điểm chưa có quy tắc nào, nên chưa có điểm nào vượt ngưỡng.',
  setUp: 'Thiết lập chấm điểm',
  isOneOf: 'là một trong',
  isOneOfPhrase: '{field} là {values}',
})
