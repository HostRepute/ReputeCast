import type { BounceClass } from "./protocol.ts";

// The bounce taxonomy ReputeCast uses (internal/providers/bounce), so a
// community adapter classifies a diagnostic exactly like a built-in one.

const ENHANCED = /\b([245])\.(\d{1,3})\.(\d{1,3})\b/;
const ENHANCED_ONLY = /^[245]\.\d{1,3}\.\d{1,3}$/;
const REPLY_ONLY = /^[245]\d\d$/;
const REPLY = /\b([245]\d\d)\b/;

function nextToken(s: string, pos: number): [string, number] {
  let i = pos;
  while (i < s.length && /\s/.test(s[i] as string)) i++;
  const start = i;
  while (i < s.length && !/\s/.test(s[i] as string)) i++;
  return [s.slice(start, i), i];
}

/**
 * Splits an SMTP reply such as "smtp; 550 5.1.1 <a@b>: User unknown" into
 * its code ("550 5.1.1") and the rest ("<a@b>: User unknown").
 */
export function splitSmtpReply(reply: string): { code: string; rest: string } {
  let s = reply.trim();
  if (s.slice(0, 5).toLowerCase() === "smtp;") s = s.slice(5).trim();
  const codes: string[] = [];
  let pos = 0;
  while (codes.length < 2) {
    const [tok, next] = nextToken(s, pos);
    if (tok === "") break;
    const t = tok.replace(/-+$/, "");
    if ((REPLY_ONLY.test(t) && codes.length === 0) || ENHANCED_ONLY.test(t)) {
      codes.push(t);
      pos = next;
      continue;
    }
    // Multi-line replies glue the code to the next part: "550-5.7.1".
    if (codes.length === 0 && t.length > 4 && t[3] === "-" && REPLY_ONLY.test(t.slice(0, 3))) {
      codes.push(t.slice(0, 3));
      if (ENHANCED_ONLY.test(t.slice(4))) {
        codes.push(t.slice(4));
        pos = next;
        continue;
      }
      pos = next - tok.length + 4;
      continue;
    }
    break;
  }
  if (codes.length === 0) return { code: "", rest: s };
  return { code: codes.join(" "), rest: s.slice(pos).trim() };
}

function enhanced(s: string): string {
  return ENHANCED.exec(s)?.[0] ?? "";
}

function containsAny(s: string, ...subs: string[]): boolean {
  return subs.some((x) => s.includes(x));
}

/** The taxonomy class for a status code and diagnostic text. */
export function classifyBounce(status: string, diagnostic = ""): BounceClass {
  const text = (status + " " + diagnostic).toLowerCase();
  const e = enhanced(status) || enhanced(diagnostic);
  if (containsAny(text, "reputation", "tss04", "sender score", "ip rate limited", "rate limited due to")) return "reputation";
  if (containsAny(text, "spamhaus", "blocklist", "blacklist", "listed on", "rbl", "spam", "content rejected", "message content", "virus", "malware", "phish")) {
    return "spam_block";
  }
  switch (e) {
    case "5.1.1":
    case "5.1.0":
    case "5.1.6":
    case "5.2.1":
      return "mailbox_unknown";
    case "4.2.2":
    case "5.2.2":
      return "mailbox_full";
    case "5.1.2":
    case "5.1.10":
    case "4.1.2":
    case "5.4.4":
    case "4.4.4":
      return "domain_invalid";
    case "5.7.1":
    case "5.7.0":
    case "5.7.23":
    case "5.7.24":
    case "5.7.25":
    case "5.7.26":
    case "5.7.27":
    case "5.4.1":
    case "5.7.509":
    case "5.7.606":
      return "policy_block";
  }
  if (containsAny(text, "user unknown", "no such user", "does not exist", "mailbox not found", "unknown recipient", "invalid recipient", "no mailbox")) return "mailbox_unknown";
  if (containsAny(text, "mailbox full", "over quota", "quota exceeded", "insufficient storage")) return "mailbox_full";
  if (containsAny(text, "domain not found", "host not found", "no mx", "nxdomain", "unrouteable", "unroutable")) return "domain_invalid";
  if (containsAny(text, "dmarc", "spf", "dkim", "not authorized", "access denied", "relay", "policy")) return "policy_block";
  if (e !== "" && e[0] === "4") return "temporary";
  const reply = REPLY.exec(status)?.[0];
  if (reply && reply[0] === "4") return "temporary";
  return "other";
}
