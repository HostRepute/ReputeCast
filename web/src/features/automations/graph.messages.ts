import { defineMessages } from '@/i18n'

/**
 * How a flow names its steps (graph.ts): step types, trigger kinds,
 * conditions and the titles on the canvas, the stats list and the step
 * panel. They mirror internal/automations/describe.go.
 *
 * A {condition} is one of the `cond…` phrases, in lower case. A {step} is
 * another step as running text names it ("step 1", from the email's name,
 * which is workspace content), or `anyEmail`. A {duration} is `duration`.
 */
export const mg = defineMessages('automations.graph', {
  // Step types: the palette, the node header, the stats list.
  typeTrigger: 'Trigger',
  typeDelay: 'Delay',
  typeEmail: 'Send email',
  typeBranch: 'Branch',
  typeFilter: 'Filter',
  typeSplit: 'A/B split',
  typeWebhook: 'Webhook',
  typeList: 'List action',
  typeGoal: 'Goal',
  typeExit: 'Exit',
  typeSms: 'Send SMS',
  typePush: 'Send push',
  typeWhatsapp: 'Send WhatsApp',

  // Trigger kinds: the palette and "Starts when".
  kindJoinedList: 'Joined list',
  kindFormSubmitted: 'Form submitted',
  kindTagAdded: 'Tag added',
  kindApiEvent: 'API event',
  kindDateField: 'Date field',
  kindPollAnswered: 'Answered a poll or survey',
  kindOrderPlaced: 'Order placed',
  kindFirstOrder: 'First order',
  kindAbandonedCheckout: 'Abandoned checkout',
  /** After the type in a trigger's header, shown in capitals: "Trigger · joined list". */
  kindWord:
    '{kind, select, joined_list {joined list} form_submitted {form submitted} tag_added {tag added} api_event {api event} date_field {date field} score_crosses {score crosses} poll_answered {poll answered} bought_product {bought product} milestone_reached {milestone reached} order_placed {order placed} first_order {first order} abandoned_checkout {abandoned checkout} other {{kind}}}',

  /** A wait or a window: "10 minutes", "2 days". */
  duration:
    '{unit, select, minutes {{n, plural, one {# minute} other {# minutes}}} hours {{n, plural, one {# hour} other {# hours}}} weeks {{n, plural, one {# week} other {# weeks}}} other {{n, plural, one {# day} other {# days}}}}',

  // What a step points at when it isn't set or no longer exists.
  aRemovedStep: 'a removed step',
  aList: 'a list',
  aForm: 'a form',
  aPollOrSurvey: 'a poll or survey',
  aStore: 'a store',
  anyEmail: 'any email',

  // Conditions, in lower case: "Opened step 1 within 2 days?", "Only contacts tagged vip".
  condOpened: 'opened {step}',
  condOpenedWithin: 'opened {step} within {duration}',
  condClickedUrl: 'clicked a {url} link',
  condClickedUrlWithin: 'clicked a {url} link within {duration}',
  condClickedIn: 'clicked a link in {step}',
  condClickedInWithin: 'clicked a link in {step} within {duration}',
  condTagged: 'tagged {tag}',
  condInList: 'in {list}',
  condSentEvent: 'sent {event}',
  condPlacedOrder: 'placed an order',
  condFieldSet: '{field} is set',
  condFieldEmpty: '{field} is empty',
  condFieldNot: "{field} isn't {value}",
  condFieldContains: '{field} contains {value}',
  condFieldIs: '{field} is {value}',

  // A goal's title.
  goalEnd: 'Reached the end of the flow',
  goalClickedUrl: 'Clicked a {url} link',
  goalClickedAny: 'Clicked any link',
  goalOpened: 'Opened an email',
  goalTagged: 'Tagged {tag}',
  goalJoined: 'Joined {list}',
  goalSent: 'Sent {event}',
  goalPlacedOrder: 'Placed an order',
  goalFallback: 'Goal',

  // The bold line on a step.
  pickTrigger: 'Pick a trigger',
  checkoutAfter: '{store} · after {hours, plural, one {# hour} other {# hours}}',
  waitFor: 'Wait {duration}',
  wait: 'Wait',
  pickEmail: 'Pick an email',
  /** A branch asks its condition: "Opened step 1 within 2 days?" */
  branchTitle: '{condition}?',
  setCondition: 'Set a condition',
  filterTitle: 'Only contacts {condition}',
  /** An A/B split: its weights ("50 / 50") and where its first path leads. */
  splitOn: '{weights} on {step}',
  smsTitle: 'Send SMS',
  writeText: 'Write the text',
  writeNotification: 'Write the notification',
  freeText: 'Free text',
  writeMessage: 'Write the message',
  pickTemplate: 'Pick a template',
  setUrl: 'Set a URL',
  pickAction: 'Pick an action',
  addToList: 'Add to {list}',
  removeFromList: 'Remove from {list}',
  addTag: 'Add tag {tag}',
  removeTag: 'Remove tag {tag}',
  unsubscribeAll: 'Unsubscribe from all lists',
  exitDefault: 'Leave the flow · stays on the list',

  // The line under the title.
  confirmedOnly: 'double opt-in confirmed only',
  sentThroughApi: 'sent through the API',
  /** A date-field trigger: "3 days before, every year at 09:00". */
  dateDetail:
    '{when, select, before {{days, plural, one {# day} other {# days}} before} after {{days, plural, one {# day} other {# days}} after} other {on the date}}{yearly, select, yes {, every year} other {}} at {time}',
  humanOnly: 'human opens only',

  noTrigger: 'No trigger yet',
  pathYes: 'Yes',
  pathNo: 'No',
})
