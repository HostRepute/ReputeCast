package demo

// The web-only posts of the website sample (publish-sell brief §4): their
// MJML as the composer's post mode saves it. The runbook has its free
// preview marker after the third paragraph.

const websiteRunbookMJML = `<mjml><mj-body>
<mj-section><mj-column>
<mj-text><p>On 28 September we moved Acme's main database from a self-managed Postgres 14 cluster to Postgres 17, with logical replication and a 41-second pause for writes at the cutover. Nobody outside the on-call channel noticed, which was the point.</p></mj-text>
<mj-text><p>This is the runbook we followed, step by step: how we sized the new cluster, seeded it without hammering production, kept 2.1 TB of tables in sync for nine days, and rehearsed the cutover until it bored us.</p></mj-text>
<mj-text><p>We're sharing it because every migration guide we found stopped at “then switch over”. The switching over is where the risk lives, so that's where most of this post goes.</p></mj-text>
<mj-raw rc-preview="end"><!-- rc:preview-end --></mj-raw>
<mj-text><h2>1. Size the target for the week after, not the day of</h2><p>A freshly seeded replica has cold caches and no bloat, so it flatters every benchmark. We sized the new primary for the load we expected a week after the cutover, with vacuum running and caches warm: 64 vCPUs and 512 GB of memory, one size up from the old one.</p></mj-text>
<mj-text><h2>2. Seed from a snapshot, then let replication catch up</h2><p>Copying 2.1 TB through a publication would have held a replication slot open for most of a day. Instead we restored last night's snapshot, created the subscription with <code>copy_data = false</code>, and let it replay only what changed since.</p></mj-text>
<mj-text><h2>3. Nine days of replication, watched like a release</h2><p>We graphed replication lag next to the API's p99 for nine days and set an alert at 30 seconds of lag. It fired twice, both times during the nightly batch export, and both times the replica caught up within four minutes.</p></mj-text>
<mj-text><h2>4. Rehearse the cutover until it's boring</h2><p>We rehearsed the cutover six times against a copy of production: freeze writes, wait for lag to reach zero, move the sequences, flip the connection string, unfreeze. The last three rehearsals took under 45 seconds each.</p></mj-text>
<mj-text><h2>5. The 41 seconds</h2><p>On the night, writes paused for 41 seconds. Reads never stopped. The only alert was the one we set ourselves, telling us the old primary had gone quiet.</p></mj-text>
</mj-column></mj-section>
</mj-body></mjml>`

const websiteAskAcmeMJML = `<mjml><mj-body>
<mj-section><mj-column>
<mj-text><p>Every month we collect the deliverability questions you send us and answer the ones that come up most. This month: the Promotions tab, warming up a new IP, and reading a DMARC report without a decoder ring.</p></mj-text>
<mj-text><h2>Why does Gmail put some of my mail in Promotions?</h2><p>Gmail sorts by what a message looks like and how people treat mail like it. Heavy images, several calls to action and a price in the subject all lean towards Promotions. It isn't a penalty: most readers check the tab. If a message is personal, write it like one.</p></mj-text>
<mj-text><h2>Should I warm up a new IP?</h2><p>Yes, if you send more than a few thousand messages a day. Start with your most engaged readers and double the volume every day or two while bounces and complaints stay low.</p></mj-text>
<mj-text><h2>What is a DMARC report really telling me?</h2><p>Who sends mail as your domain, and whether it passed SPF and DKIM. Read it for sources you don't recognise before you move your policy to quarantine.</p></mj-text>
</mj-column></mj-section>
</mj-body></mjml>`

const websiteAMAMJML = `<mjml><mj-body>
<mj-section><mj-column>
<mj-text><p>The full recording of the October AMA, with notes and timestamps for every question.</p></mj-text>
<mj-text><h2>The questions</h2><ul><li>00:02 Why the v1 API retires on 30 November, and what moves with it</li><li>00:14 How feature flags price when you have thousands of them</li><li>00:27 What the audit log export to S3 includes</li><li>00:41 The Q4 roadmap, and what we cut</li></ul></mj-text>
<mj-text><p>Founding members get the recording a week before anyone else, and pick next month's questions.</p></mj-text>
</mj-column></mj-section>
</mj-body></mjml>`

const websiteRoadmapMJML = `<mjml><mj-body>
<mj-section><mj-column>
<mj-text><p>What we're building this quarter, what we're cutting, and why the v1 API retirement comes first.</p></mj-text>
<mj-text><h2>First: the v1 API retires</h2><p>Every integration still on v1 gets an email from us in November with what to change. The v2 endpoints have been stable for a year; the retirement frees the team to build what's below.</p></mj-text>
<mj-text><h2>Building</h2><ul><li>Scheduled rollouts for feature flags</li><li>Faster exports, streamed instead of batched</li><li>Audit log exports to your SIEM</li></ul></mj-text>
<mj-text><h2>Cutting</h2><p>The legacy dashboard widgets. Fewer than one in a hundred accounts uses them, and keeping them costs us a sprint every quarter.</p></mj-text>
</mj-column></mj-section>
</mj-body></mjml>`

const websiteRateLimitsMJML = `<mjml><mj-body>
<mj-section><mj-column>
<mj-text><p>Last spring our API rate limiter throttled our three largest customers during their own product launches. Each of them was well within their contract. The limiter was counting the wrong thing.</p></mj-text>
<mj-text><p>This post walks through the redesign: buckets per customer instead of per IP, burst credits that refill overnight, and the dashboard that tells a customer how close they are before it matters.</p></mj-text>
<mj-raw rc-preview="end"><!-- rc:preview-end --></mj-raw>
<mj-text><h2>Counting customers, not addresses</h2><p>A large customer sends from hundreds of addresses behind a load balancer; a small one from two. Per-IP buckets punished the first and barely noticed the second. A token bucket per customer, sized from their plan, fixed both.</p></mj-text>
<mj-text><h2>Burst credits</h2><p>Launch days are spiky by nature. Each customer now earns burst credits overnight that a spike can spend, so a launch borrows tomorrow's quiet hours instead of failing today.</p></mj-text>
</mj-column></mj-section>
</mj-body></mjml>`
