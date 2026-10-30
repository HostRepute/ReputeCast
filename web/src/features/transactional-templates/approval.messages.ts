import { defineMessages } from '@/i18n'

/** The approval rule for publishing transactional templates (Settings › Approvals, the review screen). */
export const m = defineMessages('transactional.approval', {
  ruleTitle: 'Publishing a transactional template also requires approval',
  ruleDetail: 'Applies to everyone who can publish transactional templates. Off: publishing is logged only, and the API sends the new version at once.',
  // The approval review of a template publish request.
  cardTitle: 'Transactional template',
  cardCaption: 'The draft the approval lets its author publish',
  version: 'Version',
  versionValue: 'v{version} of {key}',
  versionReplaces: 'v{version} of {key}, replacing live v{live}',
  category: 'Category',
  subject: 'Subject',
  variables: 'Variables',
  content: 'Content',
  noContentChange: 'No content changes',
  breaking: 'Breaking change',
  testRenders: 'Test renders',
  renders: 'Renders',
  fails: 'Fails',
  checksLine: '{passed, plural, one {# check passed} other {# checks passed}}{warnings, plural, =0 {} one {, # warning} other {, # warnings}}',
  published: 'Published as v{version}',
  decisionNote: 'Approving covers this exact draft. If {name} edits it before publishing, publishing needs a new approval.',
  rejectDetail: "{name} is told and the draft isn't published. The API keeps sending the live version. Reject when it shouldn't go out at all; for fixes, request changes instead.",
  changesDetail: 'The request goes back to {name} with your comment. The draft stays unpublished until they ask again.',
})
