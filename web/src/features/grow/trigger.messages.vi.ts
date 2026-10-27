import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** Vietnamese for the automation trigger "Reached a milestone" (docs/i18n/glossary-vi.md). */
export default translate(m, {
  kind: 'Đạt mốc thưởng',
  anyMilestone: 'mọi mốc',
  aMilestone: 'một mốc',
  program: 'Chương trình',
  anyProgram: 'Mọi chương trình',
  milestone: 'Mốc',
  anyMilestoneOption: 'Mọi mốc',
  programMilestone: '{list} · {milestone}',
  once: 'Một lần cho mỗi liên hệ ở mỗi mốc',
  onceDetail: 'một lần cho mỗi liên hệ ở mỗi mốc',
  explain:
    'Bắt đầu khi người giới thiệu đạt mốc, dù mốc đó có tự gửi email hay không. Các biến của mốc thưởng và phần thưởng dùng được trong email của quy trình này.',
  noMilestones: 'Chưa có chương trình giới thiệu nào có mốc thưởng, nên chưa có gì kích hoạt quy trình này.',
  setUp: 'Thiết lập giới thiệu',
  needs: 'Chỉ người có thể sửa tự động hóa mới chọn được trình kích hoạt này.',
})
