// End-to-end check of the published client against a running ReputeCast
// instance with the Acme demo (internal/sdktest runs it with a fresh one):
// REPUTECAST_URL, REPUTECAST_API_KEY (subscribers:view, subscribers:create,
// subscribers:edit and transactional:send, without team:view) and
// REPUTECAST_WORKSPACE.
import assert from "node:assert/strict";

import { createClient, paginate, ReputeCastError, unwrap } from "@reputecast/client";

const baseUrl = process.env.REPUTECAST_URL;
const apiKey = process.env.REPUTECAST_API_KEY;
const workspace = process.env.REPUTECAST_WORKSPACE ?? "acme-newsletter";
assert.ok(baseUrl && apiKey, "set REPUTECAST_URL and REPUTECAST_API_KEY");

const rc = createClient({ baseUrl, apiKey });
const path = { workspace };

// Lists.
const lists = unwrap(await rc.GET("/api/v1/workspaces/{workspace}/lists", { params: { path } }));
assert.ok(lists.items.length > 0, "the demo has lists");
const digest = lists.items.find((l) => l.name === "Weekly Digest") ?? lists.items[0];

// Upsert a contact, twice: created, then updated.
const email = "sdk-e2e-ts@example.com";
const body = { email, firstName: "Tess", lists: [digest.id], tags: ["sdk-e2e"], consent: "single" };
const first = unwrap(await rc.POST("/api/v1/workspaces/{workspace}/contacts/upsert", { params: { path }, body }));
assert.equal(first.created, true);
const again = unwrap(await rc.POST("/api/v1/workspaces/{workspace}/contacts/upsert", { params: { path }, body: { ...body, lastName: "Script" } }));
assert.equal(again.created, false);
assert.equal(again.contactId, first.contactId);

// Page through contacts with the cursor: three pages of 25.
const seen = new Set();
for await (const contact of paginate(rc, "/api/v1/workspaces/{workspace}/contacts", { params: { path, query: { pageSize: 25 } } })) {
  assert.ok(!seen.has(contact.id), `contact ${contact.id} came twice`);
  seen.add(contact.id);
  if (seen.size === 75) break;
}
assert.equal(seen.size, 75, "three pages of contacts");
const found = unwrap(await rc.GET("/api/v1/workspaces/{workspace}/contacts", { params: { path, query: { q: email } } }));
assert.equal(found.items[0]?.id, first.contactId);

// A transactional message, with an Idempotency-Key: a repeat sends nothing new.
const user = { id: "u_42", email: "sam.rivera@outlook.com", name: "Sam Rivera", firstName: "Sam" };
const token = { id: "t_7" };
const send = () =>
  rc.POST("/api/v1/workspaces/{workspace}/transactional/messages", {
    params: { path: { workspace }, header: { "Idempotency-Key": `pwreset-${user.id}-${token.id}` } },
    body: {
      template: "password-reset",
      to: { email: user.email, name: user.name },
      data: { first_name: user.firstName, reset_url: `https://app.acme.io/reset/${token.id}`, expires_in: "30 minutes" },
    },
  });
const msg = unwrap(await send());
assert.ok(msg.id && msg.status, "the message is accepted");
assert.equal(unwrap(await send()).id, msg.id);

// A missing permission is a ReputeCastError with the problem's fields.
try {
  unwrap(await rc.GET("/api/v1/workspaces/{workspace}/members", { params: { path } }));
  assert.fail("listing members needs team:view");
} catch (err) {
  assert.ok(err instanceof ReputeCastError, String(err));
  assert.equal(err.status, 403);
  assert.equal(err.code, "permission_denied");
  assert.equal(err.permission, "team:view");
  assert.ok(err.detail.length > 0);
}

console.log(`ok ts: ${lists.items.length} lists, ${seen.size} contacts paged, message ${msg.status}, members answered 403 permission_denied`);
