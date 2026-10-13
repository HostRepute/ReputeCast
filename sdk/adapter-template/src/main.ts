import { readFileSync } from "node:fs";

import { serve } from "@reputecast/adapter-sdk";

import adapter from "./adapter.ts";

// manifest.json is what ReputeCast installed: serve() refuses to start when
// the code declares something else, and takes the network permissions from it.
const manifest: unknown = JSON.parse(readFileSync(new URL("../manifest.json", import.meta.url), "utf8"));

// PORT (4600), HOST (127.0.0.1; 0.0.0.0 in containers) and
// REPUTECAST_ADAPTER_SECRET (the signing secret ReputeCast printed) come from
// the environment.
const server = await serve(adapter, { manifest });

for (const signal of ["SIGINT", "SIGTERM"] as const) {
  process.once(signal, () => {
    void server.close().then(() => process.exit(0));
  });
}
