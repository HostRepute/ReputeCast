/**
 * @reputecast/adapter-sdk: build ReputeCast community sending adapters.
 * The protocol is specified in docs/adapters.md ("Community adapters").
 */
export * from "./protocol.ts";
export { defineAdapter, isAdapter } from "./define.ts";
export type {
  Adapter,
  AdapterContext,
  AdapterDefinition,
  AdapterEvent,
  HttpSettings,
  SendOutcome,
  SendRefusal,
  SuppressionPage,
} from "./define.ts";
export {
  AdapterError,
  BadSignatureError,
  HttpError,
  MessageRejected,
  NotSupportedError,
  ProviderError,
  isAdapterError,
  isMessageRejected,
  toWireError,
} from "./errors.ts";
export type { ProviderErrorOptions, RejectionOptions } from "./errors.ts";
export { createHttpClient, hostAllowlist } from "./http.ts";
export type { HttpClient, HttpClientOptions, HttpResponse, RequestOptions } from "./http.ts";
export { consoleLogger, redact, silentLogger } from "./log.ts";
export type { LogFields, Logger, LogLevel } from "./log.ts";
export {
  CAPABILITY_KEYS,
  ManifestError,
  capabilityDifferences,
  normalizeCapabilities,
  operationsFor,
  validateManifest,
} from "./manifest.ts";
export type { Capabilities, ConfigField, Manifest, NormalizedCapabilities, Terms } from "./manifest.ts";
export { createHandler, serve } from "./server.ts";
export type { AdapterServer, ServeOptions } from "./server.ts";
export { MAX_SKEW_SECONDS, newSecret, secretsFromEnv, sign, signatureHeader, timingSafeEqual, verifySignature } from "./signing.ts";
export type { SignatureCheck } from "./signing.ts";
export { classifyBounce, splitSmtpReply } from "./bounce.ts";
export {
  WebhookRequest,
  parseSignatureHeader,
  splitJson,
  verifyBasicAuth,
  verifyHmac,
  verifyTimestampedHmac,
} from "./webhooks.ts";
export type { HmacOptions, TimestampedHmacOptions } from "./webhooks.ts";
export { wireEvent } from "./dispatch.ts";
export { SDK_VERSION } from "./version.ts";
