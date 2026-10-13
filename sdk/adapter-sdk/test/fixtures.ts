/** A valid manifest with every capability declared (the Go tests' fake adapter). */
export function manifest(): Record<string, any> {
  return {
    protocol: 1,
    kind: "remote",
    id: "fake-esp",
    name: "Fake ESP",
    version: "1.2.3",
    maintainer: "reputecast-tests",
    capabilities: { batch: 50, webhooks: true, registerWebhooks: true, domainSetup: "full", suppressionSync: "two-way", quotaDiscovery: "rate" },
    terms: { bulkMarketing: "allowed" },
    pricing: "$0.50 per 1,000",
    config: [
      { key: "api_key", label: "API key", secret: true, required: true },
      { key: "region", label: "Region", options: ["eu", "us"] },
    ],
    permissions: { network: ["api.fake-esp.test", "*.fake-esp.test:443", "settings:region"] },
  };
}
