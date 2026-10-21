import { defineMessages } from '@/i18n'

/** The contact profile's consent per channel (SmsStates › Contact profile). */
export const m = defineMessages('smsContact', {
  title: 'Consent per list · {name}',
  lead: 'Email and SMS are separate. Each change keeps its evidence.',
  list: 'List',
  subscribed: 'Subscribed',
  confirmed: 'Confirmed',
  pending: 'Pending',
  unsubscribed: 'Unsubscribed',
  optedOut: 'Opted out',
  notSubscribed: 'Not subscribed',
  noPhone: 'No phone number yet.',
  addPhone: 'Add phone',
  changePhone: 'Change phone',
  phone: 'Phone number',
  phoneHint: 'In international format, like +1 312 555 0123.',
  phoneText: 'The country comes from the number; its line type and time zone are read from it too. SMS consent stays per list.',
  phoneSaved: 'Phone number saved',
  save: 'Save',
  cancel: 'Cancel',
  lineMobile: 'mobile',
  lineLandline: 'landline',
  lineTollFree: 'toll-free',
  lineVoip: 'VoIP',
  lineUnknown: 'line type unknown',
  // reason: the suppression's code; reasonText: an unknown code as words.
  suppressed:
    'This number is on the SMS suppression list ({reason, select, opted_out {opted out} landline {landline} not_in_service {not in service} invalid {invalid} filtered {filtered} provider_list {provider list} manual {manual} other {{reasonText}}}, {date}).',
  byKeyword: 'by texting {keyword}',
  consentVersion: 'consent text v{version}',
  // status: the opt-in confirmation text's status in the message log.
  confirmation:
    'confirmation {status, select, queued {queued} held {held} skipped {skipped} sending {sending} sent {sent} delivered {delivered} undelivered {undelivered} failed {failed} cancelled {cancelled} other {{status}}} {time}',
  withdraw: 'Withdraw SMS consent',
  record: 'Record SMS consent',
  consentSaved: 'SMS consent on {list} saved',
  foot: 'STOP withdraws SMS consent on every list and never touches email; an email unsubscribe never touches texts.',
})
