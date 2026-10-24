import { describe, expect, it } from "vitest";

import { trimTrailingSlashes } from "../src/url.ts";

describe("trimTrailingSlashes", () => {
  it("drops only the trailing slashes", () => {
    expect(trimTrailingSlashes("https://api.example.com///")).toBe("https://api.example.com");
    expect(trimTrailingSlashes("https://api.example.com/v3")).toBe("https://api.example.com/v3");
    expect(trimTrailingSlashes("/")).toBe("");
    expect(trimTrailingSlashes("")).toBe("");
  });

  it("stays linear on a long run of slashes that doesn't end the string", () => {
    const s = "/".repeat(100_000) + "x";
    const start = performance.now();
    expect(trimTrailingSlashes(s)).toBe(s);
    expect(performance.now() - start).toBeLessThan(1000);
  });
});
