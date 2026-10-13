import { readFileSync } from "node:fs";

import { defineContract } from "@reputecast/adapter-sdk/testing";
import type { ExpectedEvent } from "@reputecast/adapter-sdk/testing";

import adapter from "./src/adapter.ts";
import { API_KEY, WEBHOOK_KEY, fakeExampleEsp, signWebhook } from "./test/fake-api.ts";

// The contract suite (npx reputecast-adapter test) runs the adapter over the
// protocol against a fake of Example ESP. The recorded webhooks are shared
// with ReputeCast's Go end-to-end test.
const manifest: unknown = JSON.parse(readFileSync(new URL("./manifest.json", import.meta.url), "utf8"));
const fixtures = JSON.parse(readFileSync(new URL("./test/fixtures/webhooks.json", import.meta.url), "utf8")) as {
  name: string;
  payload: unknown;
  want: ExpectedEvent[];
}[];

export default defineContract({
  adapter,
  manifest,
  api: fakeExampleEsp,
  connection: () => ({ credentials: { api_key: API_KEY, webhook_signing_key: WEBHOOK_KEY } }),
  environment: (env) => ({ EXAMPLE_ESP_API_URL: env.url }),
  delivered(env, msgs) {
    const posts = env.requests.filter((r) => r.method === "POST" && r.path === "/v1/messages");
    if (posts.length !== msgs.length) throw new Error(`want one POST /v1/messages per message, got ${posts.length}`);
    for (const m of msgs) {
      const req = posts.find((r) => r.json<{ metadata?: Record<string, string> }>().metadata?.["rc_message"] === m.id);
      if (!req) throw new Error(`message ${m.id} wasn't sent with its id as metadata`);
      const body = req.json<{ tracking: { opens: boolean; clicks: boolean }; headers: Record<string, string> }>();
      if (body.tracking.opens || body.tracking.clicks) throw new Error("Example ESP's own tracking must be off: ReputeCast tracks opens and clicks");
      if (body.headers["List-Unsubscribe-Post"] !== "List-Unsubscribe=One-Click") throw new Error("the one-click unsubscribe header was lost");
    }
  },
  webhooks: fixtures.map((f) => ({ name: f.name, request: () => signWebhook(JSON.stringify(f.payload)), want: f.want })),
});
