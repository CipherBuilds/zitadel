// Copperholt regression (QA-AUTH-STORAGE, 7 Nov 2026): every cookie the login
// app sets on the sign-in host is Secure, HttpOnly and SameSite=Lax. The live
// Test scan found fingerprintId and NEXT_LOCALE without Secure. This scans the
// whole login source so a cookie setter added by a later upstream revision
// fails the image build until it carries the same flags.
// ZITADEL-UP01 (v4.20.4): the setters moved to lib/server/; setBrowserCookie
// (fingerprintId, bindingId, verificationCheck, deviceCodeCheck) and the
// sessions cookie use SameSite=None only when iframe embedding is enabled,
// which Copperholt never enables, and Lax otherwise.
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, it } from "vitest";

// The build runs vitest from apps/login.
const ROOT = join(process.cwd(), "src");

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) return sources(path);
    if (!/\.(ts|tsx)$/.test(name) || /\.test\.(ts|tsx)$/.test(name)) return [];
    return [path];
  });
}

// Each cookies().set / response.cookies.set call with its first argument.
function cookieSetters(source: string): string[] {
  const calls: string[] = [];
  const pattern = /(?:cookiesList|cookies\(\)|\.cookies)\s*\.set\(/g;
  for (let match = pattern.exec(source); match; match = pattern.exec(source)) {
    let depth = 1;
    let end = pattern.lastIndex;
    while (end < source.length && depth > 0) {
      if (source[end] === "(") depth += 1;
      if (source[end] === ")") depth -= 1;
      end += 1;
    }
    calls.push(source.slice(pattern.lastIndex, end - 1));
  }
  return calls;
}

describe("Copperholt sign-in host cookie flags", () => {
  const setters = sources(ROOT).flatMap((path) =>
    cookieSetters(readFileSync(path, "utf8")).map((call) => ({ path: relative(ROOT, path), call })),
  );

  it("finds the known cookie setters", () => {
    expect(setters.map((setter) => setter.path).sort((a, b) => a.localeCompare(b))).toEqual([
      "lib/server/cookies.ts",
      "lib/server/cookies.ts",
      "lib/server/fingerprint.ts",
    ]);
  });

  it("marks every cookie Secure and HttpOnly with an explicit SameSite", () => {
    for (const { path, call } of setters) {
      expect(call, `${path} sets Secure`).toMatch(/secure:\s*(true|process\.env\.NODE_ENV === "production")/);
      expect(call, `${path} sets HttpOnly`).toMatch(/httpOnly:\s*true/);
      expect(call, `${path} sets SameSite`).toMatch(/sameSite:\s*("lax"|resolvedSameSite|iFrameEnabled \? "none" : "lax")/);
    }
  });

  // Compliance (8 Nov 2026, cookie policy v14): fingerprintId lasts 30 days. It (and bindingId) is set through
  // setBrowserCookie without a lifetime, so that setter's default is the cookie's lifetime.
  it("keeps fingerprintId for 30 days", () => {
    const fingerprint = setters.filter((setter) => setter.path === "lib/server/fingerprint.ts");
    expect(fingerprint).toHaveLength(1);
    expect(fingerprint[0].call).toMatch(/\bmaxAge,/);
    const source = readFileSync(join(ROOT, "lib/server/fingerprint.ts"), "utf8");
    expect(source).toMatch(/export async function setBrowserCookie\(name: string, value: string, maxAge: number = 2592000\)/);
    expect(source).toMatch(/return getOrSetBrowserCookie\("fingerprintId"\);/);
    expect(source).not.toMatch(/31536000/);
  });
});
