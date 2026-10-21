import type { MessageDescriptor } from '@/i18n'
import type { TemplateKey } from './format'
import { m } from './messages'

/**
 * One block of a template's thumbnail (PagesStates · the template picker):
 * i the publisher's accent, w a white box, g a grey line, '' lavender.
 * Left and width in percent of the thumbnail, top and height in pixels.
 */
export type Wire = readonly ['i' | 'w' | 'g' | '', number, number, number, number]

export interface TemplateInfo {
  key: TemplateKey
  name: MessageDescriptor
  /** One line on what it's for (the picker's card). */
  line: MessageDescriptor
  /** "What “Sales page” adds". */
  adds: MessageDescriptor
  /** The template's own step between "opens in the builder" and "publish". */
  step: MessageDescriptor
  stepSub: MessageDescriptor
  /** The thumbnail; none for Blank ("Empty page"). */
  wires: readonly Wire[]
}

export const TEMPLATES: readonly TemplateInfo[] = [
  {
    key: 'about',
    name: m.tplAbout,
    line: m.tplAboutLine,
    adds: m.tplAboutAdds,
    step: m.tplAboutStep,
    stepSub: m.tplAboutStepSub,
    wires: [
      ['w', 8, 10, 84, 30],
      ['i', 14, 18, 40, 6],
      ['g', 14, 28, 56, 4],
      ['g', 8, 48, 84, 4],
      ['g', 8, 56, 70, 4],
      ['w', 8, 70, 26, 26],
      ['w', 38, 70, 26, 26],
      ['i', 8, 100, 84, 8],
    ],
  },
  {
    key: 'lead_magnet',
    name: m.tplLeadMagnet,
    line: m.tplLeadMagnetLine,
    adds: m.tplLeadMagnetAdds,
    step: m.tplLeadMagnetStep,
    stepSub: m.tplLeadMagnetStepSub,
    wires: [
      ['i', 8, 10, 54, 6],
      ['g', 8, 21, 46, 4],
      ['w', 70, 8, 22, 30],
      ['w', 8, 46, 84, 30],
      ['g', 14, 54, 48, 5],
      ['i', 14, 64, 30, 6],
      ['w', 8, 84, 26, 20],
      ['w', 37, 84, 26, 20],
      ['w', 66, 84, 26, 20],
    ],
  },
  {
    key: 'launch',
    name: m.tplLaunch,
    line: m.tplLaunchLine,
    adds: m.tplLaunchAdds,
    step: m.tplLaunchStep,
    stepSub: m.tplLaunchStepSub,
    wires: [
      ['i', 8, 10, 50, 6],
      ['g', 8, 21, 40, 4],
      ['w', 8, 34, 16, 18],
      ['w', 27, 34, 16, 18],
      ['w', 46, 34, 16, 18],
      ['w', 70, 8, 22, 30],
      ['w', 8, 62, 84, 22],
      ['i', 14, 70, 34, 6],
      ['g', 8, 92, 84, 4],
      ['g', 8, 100, 60, 4],
    ],
  },
  {
    key: 'webinar',
    name: m.tplWebinar,
    line: m.tplWebinarLine,
    adds: m.tplWebinarAdds,
    step: m.tplWebinarStep,
    stepSub: m.tplWebinarStepSub,
    wires: [
      ['i', 8, 10, 60, 6],
      ['g', 8, 21, 44, 4],
      ['w', 8, 34, 84, 28],
      ['g', 14, 41, 30, 4],
      ['g', 14, 50, 40, 4],
      ['g', 8, 70, 40, 4],
      ['g', 8, 78, 52, 4],
      ['w', 8, 90, 84, 16],
      ['i', 14, 95, 24, 6],
    ],
  },
  {
    key: 'sales',
    name: m.tplSales,
    line: m.tplSalesLine,
    adds: m.tplSalesAdds,
    step: m.tplSalesStep,
    stepSub: m.tplSalesStepSub,
    wires: [
      ['i', 8, 12, 46, 6],
      ['g', 8, 23, 40, 4],
      ['g', 8, 31, 36, 4],
      ['i', 8, 42, 30, 8],
      ['', 64, 8, 28, 46],
      ['g', 8, 62, 40, 4],
      ['g', 52, 62, 40, 4],
      ['g', 8, 70, 40, 4],
      ['g', 52, 70, 40, 4],
      ['w', 8, 84, 26, 20],
      ['w', 37, 84, 26, 20],
      ['w', 66, 84, 26, 20],
    ],
  },
  {
    key: 'blank',
    name: m.tplBlank,
    line: m.tplBlankLine,
    adds: m.tplBlankAdds,
    step: m.tplBlankStep,
    stepSub: m.tplBlankStepSub,
    wires: [],
  },
]

export function templateInfo(key: TemplateKey): TemplateInfo {
  return TEMPLATES.find((x) => x.key === key) ?? TEMPLATES[TEMPLATES.length - 1]!
}
