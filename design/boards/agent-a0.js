/* A0 design prototype only. No API calls, storage or real model execution. */
(() => {
  const root = document.getElementById('agent-app');
  const initialPage = document.body.dataset.board || 'run';
  const defaults = () => ({page: initialPage, phase: 'plan', error: 'budget', dark: false, bundle: 'pending', post: 'pending', diff: 'pending', conflict: false, dialog: '', returnFocus: '', message: '', enabled: true, manager: true, voiceLearned: false, voiceRevision: 3, selected: [true, true, false], voice: 'Clear, warm and practical. Address the reader as “you”. Keep sentences short. Prefer “try” and “build”; avoid “revolutionary” and “game-changing”.', brief: 'Re-engage Team-plan readers who have not opened in 60 days. Draft three emails over ten days and use an existing discount pool.', reply: '', calls: 40, tokens: 300000, dollars: 2, runLimitRaised: false, voiceEnabled: true});
  let state = defaults();
  const escape = value => String(value).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const paths = {
    check: '<path d="m5 12 4 4L19 6"/>', shield: '<path d="M12 3 3 6v6c0 5 9 9 9 9s9-4 9-9V6Z"/><path d="m8 12 3 3 5-6"/>',
    mail: '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3 6 9 7 9-7"/>', flow: '<circle cx="6" cy="5" r="2"/><circle cx="6" cy="19" r="2"/><circle cx="18" cy="12" r="2"/><path d="M6 7v10m0-7h6a6 6 0 0 1 4 2"/>',
    users: '<circle cx="9" cy="8" r="3"/><path d="M3 20v-2a6 6 0 0 1 12 0v2m1-15a3 3 0 0 1 0 6m2 3a5 5 0 0 1 3 4v2"/>', grid: '<path d="M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z"/>',
    settings: '<path d="M4 6h16M4 12h16M4 18h16"/><circle cx="9" cy="6" r="2"/><circle cx="16" cy="12" r="2"/><circle cx="8" cy="18" r="2"/>', star: '<path d="m12 2 2.5 7.5L22 12l-7.5 2.5L12 22l-2.5-7.5L2 12l7.5-2.5Z"/>', info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v6m0-11v1"/>', clock: '<circle cx="12" cy="12" r="9"/><path d="M12 6v6l4 2"/>', doc: '<rect x="5" y="3" width="14" height="18" rx="2"/><path d="M8 8h8M8 12h8M8 16h5"/>'
  };
  const icon = name => `<svg class="icon" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">${paths[name] || paths.doc}</svg>`;
  const button = (id, label, cls = '', extra = '') => `<button type="button" id="${id}" data-action="${id}" class="btn ${cls}" ${extra}>${label}</button>`;
  const badge = (label, tone = '') => `<span class="badge ${tone}">${label}</span>`;
  const notice = (title, text, cls = '') => `<div class="notice ${cls}">${icon(cls ? 'info' : 'shield')}<div><strong>${title}</strong><p>${text}</p></div></div>`;
  const item = (n, title, text, mark = '') => `<div class="item"><span class="step-number">${n}</span><div class="grow"><div class="row between"><h3>${title}</h3>${mark}</div><p>${text}</p></div></div>`;
  const check = text => `<li>${icon('check')}<span>${text}</span></li>`;
  const header = (title, sub, status = '') => `<div class="pagehead row between"><div><h1 tabindex="-1" id="page-title">${title}</h1><p class="sub">${sub}</p></div>${status}</div>`;
  const budget = () => `<section class="card pad"><h2>Run budget</h2><p class="sub">Illustrative usage · not a provider quote</p><div style="margin-top:16px" class="facts"><div class="fact"><span>Tool calls</span><strong>${state.runLimitRaised ? '40 / 60' : '18 / 40'}</strong></div><div class="fact"><span>Tokens</span><strong>84,200 / 300,000</strong></div><div class="fact"><span>Estimated cost</span><strong>$0.42 / $2.00</strong></div></div><div class="meter"><span style="width:21%"></span></div><small>All planning, retries and repairs count. The monthly cap still applies.</small></section>`;
  const privacy = () => `<section class="card pad stack"><h2>What the model sees</h2><ul class="checklist">${check('Your reviewed brief, permitted schema and aggregate counts')}${check('Selected unrendered draft content and Acme voice v3')}${check('Coupon tokens, never actual codes')}</ul><p class="sub">No contact-record, recipient-list, secret or export tools. Filters can miss personal information in ordinary text: review and remove private details before sharing.</p><p class="sub">Rounded totals do not prevent inference across overlapping campaigns. Your model provider controls retention and training under your account terms.</p><hr class="divider"><small>Run content expires after 30 days. Only you and authorized integration managers can open this run, within resource scope.</small></section>`;
  const nav = [['Overview','grid'],['Campaigns','mail'],['Automations','flow'],['Transactional','mail'],['Audience','users'],['Grow','flow'],['Earn','grid'],['Forms & pages','doc'],['Deliverability','shield'],['Sending providers','grid'],['Settings','settings']];
  const screens = [['run','Run'],['review','Review'],['states','States'],['settings','Settings'],['voice','Voice'],['entry','Entry points']];
  const tabs = () => `<nav class="tabs" aria-label="Run sections">${[['plan','Brief & plan'],['building','Steps'],['review','Review'],['list','All runs']].map(([key,label]) => `<button id="tab-${key}" data-phase="${key}" aria-current="${state.phase === key ? 'page' : 'false'}">${label}</button>`).join('')}</nav>`;

  function plan() {
    return `${header('Team re-engagement', 'Linh Nguyen · Claude Sonnet 5.5 · Acme Newsletter', badge('Ready to build','warn'))}${tabs()}<div class="columns"><div class="stack">
      <section class="card"><div class="cardhead row between"><h2>Your brief</h2><small>Plan v2 · answers included</small></div><div class="pad stack"><label class="field">What would you like to draft?<textarea id="brief" data-field="brief">${escape(state.brief)}</textarea></label><div class="fields"><label class="field">Language<select aria-label="Language"><option>English</option></select></label><label class="field">Style source<select aria-label="Style source"><option>Acme brand kit · Voice v3</option></select></label></div>${notice('Clarified with you', 'Weekly Digest readers with consent; no recorded human opens or clicks in 60 days. Use Welcome 15% for Acme Merch purchases, not a subscription discount.')}</div></section>
      <section class="card"><div class="cardhead row between"><h2>The plan</h2>${badge('Nothing sent or scheduled')}</div>
      ${item('1','Stage a segment', 'Team · quiet 60 days. 214 matching contacts of 48,213, counted just now in this sample. Consent and suppression checks remain required.',badge('New draft'))}
      ${item('2','Draft a three-email automation', 'A warm invitation on day 0, an Acme Merch offer on day 5 and a final check-in on day 10. Trigger: enters the segment. No enrollment or backfill on Keep.',badge('New draft'))}
      ${item('3','Reuse Welcome 15%', '2,540 codes available in Acme Merch. Reference the existing pool; do not create, reserve or reveal codes.',badge('Read only'))}
      ${item('4','Optional post and a subject proposal', 'A bonus post draft, plus a shorter subject for Weekly Digest #143. The existing campaign changes only if you accept its diff.',badge('Separate review'))}
      <div class="pad">${notice('Open tracking is incomplete', 'The count is an audience estimate, not evidence that every reader stopped reading. Review conditions before publishing the flow.','warning')}</div>
      <div class="footer stack"><div class="row between"><div><strong>Estimated total: $0.30–$0.70</strong><p class="sub">Example only · planning counts toward the $2 run ceiling.</p></div>${button('build','Build drafts','primary')}</div><div class="row">${button('edit-brief','Edit brief')}${button('cancel-run','Cancel')}</div><small>Build authorizes staging drafts in this plan. Sending and publishing always require a person.</small></div></section></div><aside class="aside stack">${privacy()}<section class="card pad stack"><h2>Limits for this run</h2><div class="facts"><div class="fact"><span>Tool calls</span><strong>40</strong></div><div class="fact"><span>Tokens</span><strong>300,000</strong></div><div class="fact"><span>Estimated cost</span><strong>$2.00</strong></div></div><small>Configured by Linh Nguyen. Reaching any limit pauses the run.</small></section></aside></div>`;
  }

  function steps() {
    return `${header('Team re-engagement', 'Drafting within plan v2. You can leave this page; the run keeps its progress.',badge('Building','warn'))}${tabs()}<div class="columns"><div class="stack"><section class="card"><div class="cardhead row between"><h2>Steps</h2>${button('cancel-run','Stop run')}</div>${item('✓','Read allowed context','Acme voice v3, the condition schema and pool metadata. No contact rows.',badge('Complete','ok'))}${item('✓','Stage the audience and first email','Segment conditions validated; email 1 compiled.',badge('Complete','ok'))}${item('!','Repair email 2','The coupon URL token was missing. Restored {{ coupons.welcome15.url }} and checked again. Repair 1 of 2.',badge('Repaired','ok'))}${item('4','Validate the remaining drafts','Compile email 3; check Liquid, subjects and the unpublished graph.',badge('In progress','warn'))}<div class="footer">${button('finish-sample','Show completed sample','primary')}<p class="sub">Prototype control: advances to the recorded example, without a model request.</p></div></section><section class="card pad stack"><h2>Conversation in this run</h2><p class="sub">Keep requests tied to this task. A changed audience or new resource needs a revised plan.</p><label class="field">Add an instruction<textarea id="reply" data-field="reply" placeholder="Make email two shorter">${escape(state.reply)}</textarea></label>${button('reply','Add to this run')}<div class="notice"><div><strong>Visible record</strong><p>Actions, validation results and your decisions. No hidden model reasoning.</p></div></div></section></div><aside class="aside stack">${budget()}${privacy()}</aside></div>`;
  }

  function review() {
    const left = [state.bundle,state.post,state.diff].filter(x => x === 'pending').length;
    const pending = state.bundle === 'pending';
    return `${header('Review the drafts', 'Team re-engagement · Plan v2 · Nothing sent, scheduled or published',badge(left ? `${left} decisions remaining` : 'Review completed',left ? 'warn' : 'ok'))}<div class="columns"><div class="stack">
      ${notice('Keep is not approval to send', 'New drafts stay separate until you keep them. Changes to existing drafts require an accepted diff. Publish and send checks run later.')}
      <div class="row between"><h2>New drafts</h2><a href="#diff-card">Review the existing-draft change</a></div>
      <section class="card" id="bundle-card"><div class="cardhead row between"><div><h2>Team re-engagement</h2><small>New bundle · 1 segment, 1 flow, 3 embedded emails</small></div>${badge(state.bundle === 'pending' ? 'Not reviewed' : state.bundle === 'kept' ? 'Kept as drafts' : 'Discarded',state.bundle === 'kept' ? 'ok' : '')}</div>
      ${state.bundle !== 'discarded' ? `<div class="pad stack"><div class="row between"><strong>Team · quiet 60 days</strong><span class="muted">214 matches · count is advisory</span></div><div class="email-grid">${[['0','A good time to catch up','A short invitation to see what is new with Acme.'],['5','A small thank-you from Acme','15% off Acme Merch, using your existing pool.'],['10','Still want to hear from us?','A final invitation, with preference and unsubscribe links.']].map(([day,title,copy]) => `<article class="email"><small>Day ${day} · Email draft</small><h3>${title}</h3><p>${copy}</p>${day === '5' ? '<div class="token"><code>{{ coupons.welcome15.code }}</code></div>' : ''}</article>`).join('')}</div><ul class="checklist">${check('Sample checks passed: MJML, Liquid, subject and graph')}${check('Email 2 repaired: coupon URL restored, then revalidated')}${check('Trigger: enters segment. Keep does not enroll existing members.')}</ul><small>Dependency bundle: these drafts are kept together. Removing an email requires a revised flow.</small></div>` : '<div class="pad"><p>The unkept bundle and its dependent drafts were removed. No existing resource was deleted.</p></div>'}
      <div class="footer row">${pending ? button('keep-bundle','Keep bundle','primary') + button('discard-bundle','Discard bundle','danger') : state.bundle === 'kept' ? button('open-flow','Open in flow editor') : ''}</div></section>
      <section class="card"><div class="cardhead row between"><div><h2>A little more from Acme</h2><small>Optional post · independent of the automation</small></div>${badge(state.post === 'pending' ? 'New draft' : state.post === 'kept' ? 'Kept as draft' : 'Discarded', state.post === 'kept' ? 'ok' : '')}</div><div class="pad"><p>${state.post === 'discarded' ? 'Post removed from this run. The automation is unchanged.' : 'A brief companion post introducing the Acme Merch collection. Unpublished and not included in any feed.'}</p></div>${state.post === 'pending' ? `<div class="footer row">${button('keep-post','Keep post')}${button('discard-post','Discard post','danger')}</div>` : ''}</section>
      <section class="card" id="diff-card"><div class="cardhead row between"><div><h2>Changes to an existing draft</h2><small>Weekly Digest #143 · subject · base revision 9</small></div>${badge(state.diff === 'pending' ? 'Your acceptance required' : state.diff === 'accepted' ? 'Accepted' : 'Rejected',state.diff === 'accepted' ? 'ok' : '')}</div><div class="pad stack"><div class="diff"><p class="removed"><small>Removed · current subject</small><del>Weekly Digest #143: what we have been building for you</del></p><p class="added"><small>Added · proposed subject</small><ins>What’s new at Acme this week</ins></p></div>${state.conflict ? notice('Changed since this proposal', 'Sofia edited revision 10. Refresh the diff and review it again. Nothing was overwritten.','warning') : '<small>Only the subject changes. Audience, schedule and content stay unchanged. Existing approval may become stale.</small>'}</div>${state.diff === 'pending' ? `<div class="footer row">${button('accept-diff','Accept change','primary',state.conflict ? 'disabled' : '')}${button('reject-diff','Reject')}${button('conflict',state.conflict ? 'Refresh sample diff' : 'Simulate another edit')}</div>` : ''}</section>
      </div><aside class="aside stack"><section class="card pad stack"><h2>Review decisions</h2><div class="facts"><div class="fact"><span>Automation bundle</span><strong>${escape(state.bundle)}</strong></div><div class="fact"><span>Optional post</span><strong>${escape(state.post)}</strong></div><div class="fact"><span>Existing subject</span><strong>${escape(state.diff)}</strong></div></div><small>Decisions are attributed to Linh Nguyen via AI agent. No send permission is delegated.</small>${button('discard-all','Discard remaining new drafts','danger',!pending && state.post !== 'pending' ? 'disabled' : '')}</section>${budget()}<section class="card pad stack"><h2>Continue with a person</h2><p class="sub">Keep the bundle to open its flow editor. Publishing remains a separate action.</p>${button('open-flow-side','Open in flow editor','',state.bundle !== 'kept' ? 'disabled' : '')}<small>Kept drafts survive cancellation and run expiry.</small></section></aside></div>`;
  }

  const states = {
    budget: ['Run budget reached','Paused','40 / 40 tool calls · 84,200 / 300,000 tokens · $0.42 / $2.00 estimated. The next step needs another tool call. Your completed drafts are safe.','A manager can raise the cumulative call limit. Continue never resets usage or bypasses the monthly cap.','Review completed drafts','review'],
    monthly: ['Monthly cap reached','Paused','This workspace has used its $25 monthly cap. The remaining run budget cannot override it. Completed drafts remain available.','Ask an integration manager to change the workspace cap, or wait for its reset.','Review completed drafts','review'],
    provider: ['The model provider is unavailable','Paused','Anthropic returned 529 (overloaded). Saved drafts and completed step results are unchanged.','Retry resumes the failed step only. Unknown billing is reconciled before retrying.','Retry failed step','retry'],
    rate: ['The provider is limiting requests','Paused','No new request will be sent until the provider’s retry interval has passed.','Keep working on the drafts. The server enforces the retry interval.','Review completed drafts','review'],
    key: ['The model key was rejected','Paused','Ask an integration manager to replace the key in AI settings. No credentials appear in this run.','Existing drafts are safe. Resume only after the connection works.','Open AI settings','settings'],
    missing: ['Connect a model to use the agent','Not configured','Use your workspace’s own Anthropic, OpenAI, Google, Azure or compatible model.','Nothing is sent before you ask. An integration manager connects and enables the feature.','Open AI settings','settings'],
    tools: ['This model does not support tools','Unavailable','Agent runs need verified tool-calling support. One-shot subject ideas and rewriting can still work.','Choose a model with a successful Tools capability check.','Open AI settings','settings'],
    off: ['The agent is switched off','Unavailable','An integration manager disabled Agent for this workspace. No further model steps or draft writes will start.','You can still review drafts you have permission to edit.','Review completed drafts','review'],
    permission: ['Your permissions changed','Paused','You no longer have permission to edit automations. No later tool call can use your earlier access.','Authorized recovery drafts remain accessible. Restoring a permission still requires an explicit continuation.','Review permitted drafts','review'],
    scope: ['A resource is outside your scope','Needs a revised plan','The requested audience belongs to a resource scope you cannot access. Its name, count and content are not disclosed.','Choose a permitted resource; a manager cannot use integration access to bypass content scope.','Edit the plan','plan'],
    refusal: ['The model declined this request','Paused','“I can help write a consent-based newsletter, but not impersonate a payment service.”','This is a sample provider refusal. Valid completed drafts are unchanged.','Edit the brief','plan'],
    validation: ['Email 2 still needs a correction','Needs attention','The coupon URL is invalid after two repair attempts. No invalid runnable draft was saved.','Edit the failing content manually or revise the task. The other drafts remain available.','Review valid drafts','review'],
    cancelled: ['Run cancelled','Cancelled','No further steps will start. An in-flight model request may still be billed.','Successful drafts remain available until run expiry. Discard is a separate choice.','Review saved drafts','review'],
    offline: ['Connection lost','Reconnecting','The worker may still be running. This screen is showing the last saved checkpoint, not claiming the run failed.','Reconnect fetches persisted progress; it does not restart completed steps.','Show reconnected sample','retry'],
    conflict: ['Changed since this proposal','Conflict','Weekly Digest #143 is now revision 10. The proposal was made against revision 9.','Nothing was applied. Refresh the diff before accepting again.','Review the conflict','conflict-review'],
    expired: ['This run has expired','Expired','Its brief, plan, messages and unkept drafts were removed after 30 days. Unapplied proposals expired too.','Kept drafts and content-free activity records follow their normal retention.','Start a new brief','plan'],
    reputation: ['Reputation questions belong in HostRepute','Outside this task','This agent drafts content. It cannot diagnose blocklists, inbox placement or sender reputation.','Open Deliverability to access HostRepute if connected. No run content is forwarded.','View handoff','handoff'],
    empty: ['Start with a brief','No runs yet','Describe the content and audience you want to draft. Leave out contact details and credentials.','You review a plan before any product drafts are created.','Draft a plan','plan'],
    loading: ['Preparing your plan','Planning','Reading only the allowed context and recording usage. No product drafts exist yet.','Planning consumes the same run budget. Cancellation may not prevent billing for an in-flight request.','Cancel run','cancel'],
    queued: ['Waiting for an execution slot','Queued','Two runs are active in this workspace. This run is next in the queue.','No model request is billed while this run is only queued.','Cancel queued run','cancel'],
    ready: ['Your drafts are ready to review','Notification','Team re-engagement has new drafts and a proposed change. Nothing was sent or published.','Only the initiator receives this notification, once for this status revision. Email contains a link, no draft content.','Open the run','review']
  };

  function errorStates() {
    const [title,status,body,detail,action] = state.error === 'budget' && state.runLimitRaised
      ? ['Run limit raised', 'Ready to continue', '40 / 60 tool calls used. Token and cost ceilings are unchanged. Completed drafts are safe.', 'Continue rechecks permissions and reserves the next request against the remaining run and monthly budgets.', 'Continue sample run']
      : states[state.error];
    const options = Object.entries(states).map(([key,v]) => `<option value="${key}" ${state.error === key ? 'selected' : ''}>${v[0]}</option>`).join('');
    return `${header('Agent states','Design fixtures · choose a state to inspect its recovery action')}<div class="state-grid"><nav class="state-list" aria-label="State examples">${Object.entries(states).map(([key,v]) => `<button data-error="${key}" id="state-${key}" aria-current="${state.error === key}">${v[0]}</button>`).join('')}</nav><div class="stack"><label class="field state-select">State example<select id="state-picker">${options}</select></label><section class="card state-body">${badge(status,status === 'Expired' || status === 'Not configured' ? '' : 'warn')}<h2>${title}</h2><p>${body}</p><hr class="divider"><p>${detail}</p><div class="row">${button('state-action',action,'primary')}${state.error === 'budget' && !state.runLimitRaised ? button('raise-limit',state.manager ? 'Review higher limit' : 'Ask an integration manager') : ''}</div>${state.error === 'budget' ? `<div class="row">${button('role',state.manager ? 'Preview as non-manager' : 'Preview as manager')}<small>Current call ceiling: ${state.runLimitRaised ? 60 : 40}. Counters are not reset.</small></div>` : ''}</section>${privacy()}</div></div>`;
  }

  function settings() {
    return `${header('AI assistant','Settings › Integrations · extension to the existing connection',badge('Tools verified','ok'))}<div class="columns"><div class="stack"><section class="card"><div class="cardhead row between"><h2>Features and what each sends</h2>${button('role',state.manager ? 'Preview as member' : 'Preview as manager')}</div><div class="pad row between"><div class="grow"><h3>Agent</h3><p class="sub">Plan and build staging drafts using your workspace’s model. Existing resources need your acceptance.</p></div><button id="agent-toggle" data-action="agent-toggle" class="switch" role="switch" aria-label="Enable Agent" aria-checked="${state.enabled}" ${!state.manager ? 'disabled' : ''}></button></div><div class="pad row between"><div class="grow"><h3>Brand voice</h3><p class="sub">Learn a suggested voice from selected, unrendered campaign text. Manual voice editing remains available when this is off.</p></div><button id="voice-toggle" data-action="voice-toggle" class="switch" role="switch" aria-label="Enable Brand voice learning" aria-checked="${state.voiceEnabled}" ${!state.manager ? 'disabled' : ''}></button></div><div class="pad">${notice(state.enabled ? 'Enabled for permitted members' : 'Agent is off',state.enabled ? 'Requires AI access and the permissions for each resource. The original seven one-shot feature switches stay unchanged.' : 'New runs cannot start. Active runs pause before their next step. Review of permitted saved drafts remains available.')}</div></section><section class="card"><div class="cardhead"><h2>Default limits per run</h2><p class="sub">The first limit reached pauses the run. Only integration managers can change these.</p></div><form id="budget-form" class="pad stack"><div class="fields"><label class="field">Tool calls<input id="calls" name="calls" type="number" min="1" step="1" value="${state.calls}" required ${!state.manager ? 'disabled' : ''}></label><label class="field">Total tokens<input id="tokens" name="tokens" type="number" min="1" step="1" value="${state.tokens}" required ${!state.manager ? 'disabled' : ''}></label><label class="field">Estimated US dollars<input id="dollars" name="dollars" type="number" min="0.01" step="0.01" value="${state.dollars}" required ${!state.manager ? 'disabled' : ''}></label></div><small>Planning, repairs and retries count. Changing defaults does not reset existing runs.</small><div class="row"><button type="submit" class="btn primary" ${!state.manager ? 'disabled' : ''}>Save run defaults</button><span class="muted">Monthly cap stays $25.00</span></div></form></section><section class="card pad stack"><h2>Content, access and retention</h2><p>Brief, plan and step summaries expire 30 days after creation. Messages and sanitized tool results expire seven days after closure, or at run expiry if earlier.</p><p class="sub">Only the initiator and integration managers can read a run, within resource scope. Activity records remain content-free for 90 days. Kept drafts are not deleted.</p><small>Backup copies follow the instance’s backup policy. The model provider’s own retention is separate.</small></section></div><aside class="aside stack">${privacy()}<section class="card pad stack"><h2>Your connection</h2><p>Anthropic · Claude Sonnet 5.5</p><p class="sub">Text · Structured output · Tools</p><small>The capability check uses a fixed prompt without workspace content. API keys remain write-only.</small></section></aside></div>`;
  }

  function voice() {
    return `${header('Brand voice','Settings › Brand kit · Acme Newsletter',badge(`Saved revision ${state.voiceRevision}`,'ok'))}<div class="columns"><div class="stack"><section class="card pad stack"><h2>How Acme sounds</h2><p class="sub">Used by subject ideas, rewrites, draft-from-brief and agent runs. Existing drafts do not change when you save a voice.</p><label class="field">Tone, address and vocabulary<textarea id="voice" data-field="voice" style="min-height:145px">${escape(state.voice)}</textarea></label><div>${button('save-voice','Save voice','primary')}</div><small>Manual editing needs permission to edit templates. No AI request is required.</small></section><section class="card pad stack"><h2>Learn from sent campaigns</h2><p class="sub">Choose up to ten. Only selected, unrendered text is sent; tokens remain unexpanded. This example selects two campaigns.</p><div>${['October Product Update','Weekly Digest #142','Partner webinar'].map((name,i) => `<label class="checkfield"><input type="checkbox" data-sample="${i}" ${state.selected[i] ? 'checked' : ''}>${name}</label>`).join('')}</div><div>${button('learn-voice','Learn a suggested voice','',state.selected.some(Boolean) ? '' : 'disabled')}</div><small>Requires the Brand voice feature, AI access and read access to each selected campaign. Never sends recipients or their field values.</small></section>${state.voiceLearned ? `<section class="card pad stack"><h2>Suggested voice · not saved</h2><div class="diff"><p class="removed"><small>Before</small>${escape(state.voice)}</p><p class="added"><small>Proposed</small>Warm, direct and specific. Address the reader as “you”. Lead with what changed, use short sentences, and close with one clear invitation. Prefer “try” and “build”; avoid hype.</p></div><div class="row">${button('accept-voice','Save suggested revision','primary')}${button('reject-voice','Reject')}</div></section>` : ''}</div><aside class="aside stack">${privacy()}<section class="card pad stack"><h2>Preview</h2><p>“A few useful things shipped this week. Here is what changed, and how to try them.”</p><small>Illustrative voice example. Not generated from live campaign content.</small></section></aside></div>`;
  }

  function entry() {
    return `${header('Start where you work','A0 additions to existing screens · each action opens a scoped task')}<div class="entry-grid">${[
      ['Campaigns','New campaign',['Regular email','Recurring digest','With the agent'],'Choose a task instead of a blank canvas. Existing creation paths stay available.'],
      ['Composer · Weekly Digest #143','Assist',['Subject ideas','Rewrite selection','Draft from a brief'],'A whole replacement for an existing draft is a proposal. It does not overwrite the draft.'],
      ['Automation editor','Create',['Add a step','Describe the automation'],'Only registered triggers and steps. Review the graph diff before applying or publishing.'],
      ['Command palette','Find an action',['Search campaigns','Ask the agent…'],'Opens a named task, never an unbounded workspace chat.']
    ].map(([title,menu,items,copy],n) => `<section class="card pad stack"><div class="row between"><h2>${title}</h2>${badge('A0 addition')}</div><h3>${menu}</h3><div class="entry-menu">${items.map((label,i) => `<button data-entry="${n}" data-target="${i === items.length - 1 ? 'agent' : 'existing'}" class="${i === items.length - 1 ? 'feature' : ''}">${i === items.length - 1 ? icon('star') : ''} ${label}</button>`).join('')}</div><p class="sub">${copy}</p></section>`).join('')}</div><div style="margin-top:20px">${notice('No new send or publish shortcut','After reviewing drafts, open the normal editor or Review & send. Existing approvals and readiness checks remain mandatory.')}</div>`;
  }

  function runList() {
    return `${header('Agent runs','Only runs you are allowed to open in Acme Newsletter',button('new-run','New brief','primary'))}${tabs()}<section class="card"><div class="cardhead"><h2>Recent runs</h2></div>${item(icon('star'),'Team re-engagement','Linh Nguyen · Claude Sonnet 5.5 · $0.42 estimated · expires 30 days after creation',button('list-open','Ready to review'))}${item(icon('clock'),'Shorter onboarding copy','Linh Nguyen · sample history · $0.18 estimated',badge('Completed','ok'))}<div class="footer"><small>Illustrative history. No production records are created by this prototype.</small></div></section>`;
  }

  function modal() {
    if (!state.dialog) return '';
    const messages = {
      'discard-bundle': ['Discard this bundle?', 'Remove the staged segment, unpublished flow and its three embedded emails together. Existing resources and kept drafts are not affected.','Discard bundle'],
      'discard-all': ['Discard remaining new drafts?', 'Remove only unkept drafts and their unkept dependencies. Kept objects and the existing campaign proposal are not deleted.','Discard remaining'],
      'raise-limit': ['Raise this run’s call limit?', `40 → 60 cumulative tool calls. Used calls stay 40. The token ceiling stays 300,000; the cost ceiling stays $2.00 and the monthly cap still applies. This sample manager action is audited.`, 'Raise to 60 calls'],
      handoff: ['Continue in the existing editor', 'This is a design-only handoff. In the product, the editor opens with ordinary drafts. Publishing, sending, requesting approval and test sends require a separate human action.','Close']
    };
    const [title,copy,action] = messages[state.dialog];
    return `<div class="modal-scrim"><section role="dialog" aria-modal="true" aria-labelledby="dialog-title" class="modal"><h2 id="dialog-title">${title}</h2><p>${copy}</p><div class="row">${button('confirm-dialog',action,state.dialog.includes('discard') ? 'danger' : 'primary')}${state.dialog !== 'handoff' ? button('close-dialog','Cancel') : ''}</div></section></div>`;
  }

  function render(focusId) {
    document.documentElement.dataset.theme = state.dark ? 'dark' : 'light';
    const content = state.page === 'run' ? (state.phase === 'building' ? steps() : state.phase === 'review' ? review() : state.phase === 'list' ? runList() : plan()) : state.page === 'review' ? review() : state.page === 'states' ? errorStates() : state.page === 'settings' ? settings() : state.page === 'voice' ? voice() : entry();
    root.innerHTML = `<div class="prototype"><div class="row between"><strong>A0 · Interactive design, not the app</strong><span>Sample data and costs · no model calls or real writes</span></div><div class="row between" style="margin-top:8px"><nav class="row" aria-label="Design boards">${screens.map(([key,label]) => `<button class="btn" id="page-${key}" data-page="${key}" aria-pressed="${state.page === key}">${label}</button>`).join('')}</nav><div class="row">${button('theme',state.dark ? 'Light' : 'Dark')}${button('reset','Reset')}</div></div></div><div class="shell"><aside class="sidebar"><div class="brand">${icon('shield')}<span class="label">ReputeCast</span></div><div class="workspace"><strong>Acme Newsletter</strong><small>Owner</small></div><nav aria-label="Workspace">${nav.map(([label,glyph]) => `<span class="side-link ${label === (['settings','voice'].includes(state.page) ? 'Settings' : 'Campaigns') ? 'active' : ''}" title="${label}">${icon(glyph)}<span class="label">${label}</span></span>`).join('')}</nav></aside><div><header class="topbar row between"><div class="row"><span class="muted">${['settings','voice'].includes(state.page) ? 'Settings' : 'Campaigns'}</span><span class="muted">/</span><strong>AI agent</strong></div><span class="row"><small>Acme Newsletter</small><span class="avatar">LN</span><strong>Linh</strong></span></header><main class="content">${state.message ? `<div class="alert-status" role="status" style="margin-bottom:16px">${escape(state.message)}</div>` : '<div role="status" class="sr"></div>'}${content}</main><nav class="bottomnav" aria-label="Mobile navigation">${[['Overview','grid'],['Campaigns','mail'],['Agent','star'],['More','grid']].map(([label,glyph]) => `<button data-action="${label === 'Agent' ? 'new-run' : 'handoff'}" class="${label === 'Agent' ? 'active' : ''}">${icon(glyph)}${label}</button>`).join('')}</nav></div></div>${modal()}`;
    if (!state.manager && state.page === 'settings') {
      const role = root.querySelector('.workspace small');
      if (role) role.textContent = 'Editor · member preview';
      root.querySelector('.avatar').textContent = 'SR';
      root.querySelector('.topbar > span strong').textContent = 'Sofia';
      root.querySelector('.content').insertAdjacentHTML('afterbegin', notice('Member preview · settings are read-only', 'Sofia can use enabled AI features, but only an integration manager can change these settings. The preview controls are outside the product permissions.'));
    }
    const target = state.dialog ? document.getElementById('confirm-dialog') : document.getElementById(focusId) || (focusId ? document.getElementById('page-title') : null);
    target?.focus({preventScroll:true});
  }

  root.addEventListener('input', e => { if (e.target.dataset.field) state[e.target.dataset.field] = e.target.value; });
  root.addEventListener('change', e => {
    if (e.target.id === 'state-picker') { state.error = e.target.value; state.message = ''; render('state-picker'); }
    if (e.target.dataset.sample !== undefined) { state.selected[Number(e.target.dataset.sample)] = e.target.checked; render(); }
  });
  root.addEventListener('submit', e => {
    e.preventDefault();
    if (e.target.id !== 'budget-form' || !state.manager || !e.target.reportValidity()) return;
    const data = new FormData(e.target);
    for (const key of ['calls','tokens','dollars']) state[key] = Number(data.get(key));
    state.message = 'Run defaults saved in this prototype only. Existing runs and monthly caps are unchanged.';
    render('calls');
  });
  root.addEventListener('click', e => {
    const el = e.target.closest('button');
    if (!el || el.disabled) return;
    const id = el.dataset.action;
    if (!id && !el.dataset.page && !el.dataset.phase && !el.dataset.error && el.dataset.entry === undefined) return;
    state.message = '';
    if (el.dataset.page) { state.page = el.dataset.page; render(el.id); return; }
    if (el.dataset.phase) { state.phase = el.dataset.phase; render(el.id); return; }
    if (el.dataset.error) { state.error = el.dataset.error; render(el.id); return; }
    if (el.dataset.entry) { if (el.dataset.target === 'agent') {state.page='run';state.phase='plan';} else {state.dialog='handoff';} render('page-title'); return; }
    if (id === 'reset') state = defaults();
    else if (id === 'theme') state.dark = !state.dark;
    else if (id === 'build') { if (!state.brief.trim()) {state.message='Enter a brief before building.';render('brief');return;} state.phase='building'; }
    else if (id === 'edit-brief') { document.getElementById('brief').focus(); return; }
    else if (id === 'finish-sample') {state.page='review';state.message='Sample run finished. Email 2 was repaired; every draft still needs your review.';}
    else if (id === 'cancel-run') {state.page='states';state.error='cancelled';}
    else if (id === 'reply') {state.message=state.reply.trim() ? 'Instruction added to this sample run. In the product, a changed scope requires a new plan before more drafts are built.' : 'Write an instruction for this run first.';state.reply='';}
    else if (id === 'keep-bundle') {state.bundle='kept';state.message='Bundle kept as ordinary drafts. No contacts enrolled and no emails sent.';}
    else if (id === 'keep-post') {state.post='kept';state.message='Post kept as an unpublished draft.';}
    else if (id === 'discard-post') {state.post='discarded';state.message='Optional post discarded. The automation bundle is unchanged.';}
    else if (id === 'accept-diff' && !state.conflict) {state.diff='accepted';state.message='Subject change accepted in this example. Sending still requires readiness and approval checks.';}
    else if (id === 'reject-diff') {state.diff='rejected';state.message='Proposal rejected. Weekly Digest #143 is unchanged.';}
    else if (id === 'conflict') {state.conflict=!state.conflict;state.message=state.conflict ? 'Simulated a concurrent edit. Accept is blocked.' : 'Sample diff refreshed. Review again before accepting.';}
    else if (['discard-bundle','discard-all','raise-limit','handoff','open-flow','open-flow-side'].includes(id)) {
      if (id === 'raise-limit' && !state.manager) state.message='Ask Linh Nguyen or Marcus Chen to review the cumulative run limits. No limit was changed.';
      else {state.dialog=id.startsWith('open-') ? 'handoff' : id;state.returnFocus=el.id;}
    }
    else if (id === 'close-dialog') {state.dialog='';render(state.returnFocus);return;}
    else if (id === 'confirm-dialog') {
      if (state.dialog === 'discard-bundle') state.bundle='discarded';
      if (state.dialog === 'discard-all') {if(state.bundle==='pending') state.bundle='discarded';if(state.post==='pending') state.post='discarded';}
      if (state.dialog === 'raise-limit') {state.runLimitRaised=true;state.message='Sample limit raised to 60. Used calls stay 40; continuation requires a new authorized request.';}
      state.dialog='';render(state.returnFocus || 'page-title');return;
    }
    else if (id === 'state-action') {
      const next = state.error === 'budget' && state.runLimitRaised ? 'retry' : states[state.error][5];
      if (next === 'review') state.page='review';
      else if(next==='settings') state.page='settings';
      else if(next==='plan'){state.page='run';state.phase='plan';}
      else if(next==='retry'){state.page='run';state.phase='building';}
      else if(next==='cancel') state.error='cancelled';
      else if(next==='conflict-review'){state.page='review';state.conflict=true;}
      else state.dialog='handoff';
    }
    else if (id === 'role') state.manager=!state.manager;
    else if (id === 'agent-toggle' && state.manager) state.enabled=!state.enabled;
    else if (id === 'voice-toggle' && state.manager) state.voiceEnabled=!state.voiceEnabled;
    else if (id === 'save-voice') {state.voiceRevision++;state.message='Manual voice revision saved in this prototype. No existing draft was rewritten.';}
    else if (id === 'learn-voice') {
      if (!state.voiceEnabled) state.message='Brand voice learning is off. Ask an integration manager to enable it, or edit the voice manually.';
      else {state.voiceLearned=true;state.message=`Sample suggestion from ${state.selected.filter(Boolean).length} selected campaigns. Review before saving.`;}
    }
    else if (id === 'accept-voice') {state.voice='Warm, direct and specific. Address the reader as “you”. Lead with what changed, use short sentences, and close with one clear invitation. Prefer “try” and “build”; avoid hype.';state.voiceRevision++;state.voiceLearned=false;state.message='Suggested voice saved as a new revision.';}
    else if (id === 'reject-voice') {state.voiceLearned=false;state.message='Suggestion rejected. Saved voice unchanged.';}
    else if (id === 'new-run') {state.page='run';state.phase='plan';}
    else if (id === 'list-open') state.page='review';
    render(document.getElementById(id) ? id : 'page-title');
    if (['build','finish-sample','state-action','new-run'].includes(id)) window.scrollTo(0,0);
  });
  document.addEventListener('keydown', e => {
    if (!state.dialog) return;
    if (e.key === 'Escape') {state.dialog='';render(state.returnFocus);}
    if (e.key === 'Tab') {
      const controls=[...root.querySelectorAll('[role=dialog] button')];
      if (e.shiftKey && document.activeElement===controls[0]) {e.preventDefault();controls.at(-1).focus();}
      else if (!e.shiftKey && document.activeElement===controls.at(-1)) {e.preventDefault();controls[0].focus();}
    }
  });
  render();
})();
