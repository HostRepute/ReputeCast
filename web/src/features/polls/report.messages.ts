import { defineMessages } from '@/i18n'

/**
 * Poll results in the campaign report (design expansion, X11; brief §5.8,
 * PollResults): the tiles, "How answers are counted", the click map's poll
 * region and the Poll section.
 */
export const m = defineMessages('pollReport', {
  // Headline tiles.
  tileAnswers: 'Poll answers',
  tileAnswersSub: '{percent} of delivered · {nonContacts, plural, =0 {none from non-contacts} other {# more from non-contacts}}',
  tileAutomated: 'Automated visits ignored',
  tileAutomatedSub: 'gateways and prefetchers · never answers',

  // How answers are counted.
  howTitle: 'How answers are counted',
  howRecorded: 'An answer is recorded on the answer page, after it’s visible for a moment, or when someone presses Confirm. Opening a link records nothing.',
  howAutomated:
    '<b>{count, plural, one {# automated visit} other {# automated visits}}</b> {count, plural, one {was} other {were}} ignored: link scanners and prefetchers caught by the tracking module’s classifier.',
  howChanged: '<b>{count, plural, one {# person changed their answer.} other {# people changed their answer.}}</b> The latest answer counts; the history is kept.',
  howNonContacts:
    '<b>{count, plural, one {# answer from someone who isn’t a contact} other {# answers from people who aren’t contacts}}</b> came through the web archive. They’re counted apart.',
  howTests: 'Test answers from test sends are never counted.',
  howPage: 'Answer page: {url}',
  openPage: 'Open the answer page',

  // Click map.
  mapRegion: 'Poll · {count, plural, one {# answer} other {# answers}}',
  mapNote: 'The poll is one region with its answer count, not a share of clicks: answers come from the answer page, so a link scanner opening the buttons adds nothing here.',

  // The section's header.
  kicker: 'Poll',
  kickerOf: 'Poll {n} of {total}',
  statusOpenCloses: 'Open · closes {date}',
  statusOpen: 'Open · until someone closes it',
  statusClosedScheduled: 'Closed · {date} · as scheduled',
  statusClosedBy: 'Closed · {date} · by {name}',
  statusClosed: 'Closed · {date}',
  live: 'Live · updates every minute',
  answersOf: '<b>{count, plural, one {# answer} other {# answers}}</b> · {percent} of delivered',
  answersOfFinal: '<b>{count, plural, one {# answer} other {# answers}}</b> · {percent} of delivered · final',
  answersSoFar: '<b>{count, plural, one {# answer so far} other {# answers so far}}</b> · {elapsed} after sending',
  automatedChip: '{count, plural, one {# automated visit ignored} other {# automated visits ignored}}',
  automatedWhy:
    '{gateways, plural, one {# from a security gateway that opens links before the reader does} other {# from security gateways that open links before the reader does}}, {prefetch, plural, one {# from a prefetching proxy} other {# from prefetching proxies}}. None of them counts as an answer.',
  closeNow: 'Close poll now',
  reopen: 'Reopen poll',
  exportResults: 'Export results',
  exportAnswers: 'Export answers with contacts',
  exportAnswersAction: 'Exporting answers with contacts',
  closeAction: 'Closing the poll',
  closeBadge: 'Close poll',
  closeTitle: 'Close this poll now?',
  closeBody: 'Its answer page will say it closed and record nothing. Closing is audited, and you can reopen it.',
  closeConfirm: 'Close poll',
  closed: 'Poll closed',
  reopened: 'Poll reopened',
  failed: 'Couldn’t change the poll',
  exportFailed: 'Couldn’t export',
  closedNote: 'The answer page now says “This poll closed on {date}.” and records nothing. Reopening is audited; answers made after reopening count as usual.',

  // 0–10 scale.
  nps: 'Net Promoter Score',
  npsSoFar: 'NPS so far',
  npsLine: 'promoters {promoters} minus detractors {detractors}',
  npsEarly: 'early answers lean positive; it settles over the first day',
  npsBar: 'Promoters {promoters} ({pp}), Passives {passives} ({pap}), Detractors {detractors} ({dp})',
  promotersItem: '<b>Promoters</b> 9–10 · {count} · {percent}',
  passivesItem: '<b>Passives</b> 7–8 · {count} · {percent}',
  detractorsItem: '<b>Detractors</b> 0–6 · {count} · {percent}',
  promotersTitle: 'Promoters · {count}',
  passivesTitle: 'Passives · {count}',
  detractorsTitle: 'Detractors · {count}',
  byScore: 'Answers by score',
  byScoreAria: 'Answers by score, 0 to 10: {list}',
  scoreTitle: '{score} · {count, plural, one {# answer} other {# answers}}',
  scoreItem: '{score}: {count}',

  // Choice and emoji polls.
  answers: 'Answers',
  savedAsTags: 'Saved as tags',
  tagsNote: 'A changed answer swaps the tag. Segments can use the tags right away, for example “has tag {tag}”.',
  savedToField: 'Saved to a field',
  fieldNote: 'Each answer sets {field} on the contact; a changed answer changes it.',

  // Breakdown.
  breakdown: 'Breakdown',
  breakdownBy: 'Break down by',
  byTier: 'Engagement tier',
  byList: 'List',
  byLanguage: 'Language',
  breakdownTable: 'NPS by {by}',
  answersTable: 'Answers by {by}',
  colTier: 'Tier',
  colList: 'List',
  colLanguage: 'Language',
  colAnswers: 'Answers',
  colShare: 'Share',
  colNps: 'NPS',
  colTop: 'Top answer',
  tooFew: 'Too few to compare',
  tooFewNote: 'Rows with fewer than 30 answers say “Too few to compare”.',
  langSummary: 'By language: {list}.',
  langSummaryFew: 'By language: {list}; {few}.',
  langNps: '{name} NPS {nps}',
  langNpsShort: '{name} {nps}',
  langFew: '{name} has {count, plural, one {# answer} other {# answers}}',
  noBreakdown: 'No answers to break down yet.',

  // Over time.
  overTime: 'Answers over the first 72 hours',
  soFarSix: 'Answers so far · the first 6 hours',
  cumulative: 'cumulative',
  overTimeNote: '{percent} came in the first 6 hours. {after, plural, =0 {None arrived after 72 hours.} other {# more arrived after 72 hours.}}',
  overTimeAria: 'Cumulative answers by hour: {count} by {hours} hours',
  bySix: '{percent} by +6 h',
  sentAt: 'Sent {time}',
  // An hour tick on the chart: +24 h.
  plusHours: '+{n} h',
  now: 'now',

  // Comments.
  comments: 'Comments · {count}',
  filterComments: 'Filter comments',
  all: 'All',
  promoters: 'Promoters',
  passives: 'Passives',
  detractors: 'Detractors',
  searchComments: 'Search comments',
  commentsTable: 'Comments',
  colScore: 'Score',
  colAnswer: 'Answer',
  colComment: 'Comment',
  colContact: 'Contact',
  colTime: 'Time',
  promoter: 'promoter',
  passive: 'passive',
  detractor: 'detractor',
  notAContact: 'Not a contact',
  aContact: 'A contact',
  quote: '“{text}”',
  /** A comment's line on phones, under the quote. */
  commentBy: '{who} · {when}',
  commentByAnswer: '{answer} · {who} · {when}',
  showing: 'Showing {shown} of {total} · names are shown to people with subscribers:view',
  previous: 'Previous',
  next: 'Next',
  noComments: 'No comments yet.',
  noMatch: 'No comment matches “{query}”.',

  // Footer.
  footChanged: '{count, plural, one {# person changed their answer} other {# people changed their answer}}',
  footNonContacts: '{count, plural, one {# answer from someone who isn’t a contact, counted apart} other {# answers from people who aren’t contacts, counted apart}}',
  footTests: 'test answers are never counted',
  footExports: 'Results exports are aggregate only, and audited',

  // Empty, loading, error.
  emptyTitle: 'No answers yet',
  emptyBody: '{campaign} went out at {time}, {ago}. Answers appear here as people pick them on the answer page. Opening an email or its links doesn’t count.',
  emptyPoll: 'Poll: “{question}” · {closes}',
  closesOn: 'closes {date}',
  loading: 'Loading the poll results',
  errorTitle: 'Couldn’t load the poll results',
  errorBody: 'The rest of the report loaded. {detail} Nothing is lost, and answers keep being recorded.',
  retry: 'Retry',
  request: 'request {id}',
})
