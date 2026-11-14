// Copperholt regression (LIC-FIX01, 14 Nov 2026): every page this modified
// login UI serves shows the AGPL-3.0 section 13 source offer in Compliance's
// FINAL wording, with "Source code" linking to the published fork. Every root
// layout (any file rendering <html>) must render the footer, so a page tree
// added by a later upstream revision fails the image build until it does.
import { render } from "@testing-library/react";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, it } from "vitest";
import { CopperholtSourceOffer } from "./copperholt-source-offer";

// The build runs vitest from apps/login.
const APP = join(process.cwd(), "src", "app");

function roots(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) return roots(path);
    if (!/\.tsx$/.test(name) || /\.test\.tsx$/.test(name)) return [];
    return /<html[\s>]/.test(readFileSync(path, "utf8")) ? [path] : [];
  });
}

describe("Copperholt AGPL source offer", () => {
  it("is Compliance's FINAL sentence with the Source code link and no full stop", () => {
    const { getByTestId } = render(<CopperholtSourceOffer />);
    expect(getByTestId("copperholt-source-offer").textContent).toBe(
      "This sign-in service runs a modified version of ZITADEL, free software under the GNU Affero General " +
        "Public License v3.0. Get the source code of the version running here, free of charge: Source code",
    );
    const link = getByTestId("copperholt-source-link") as HTMLAnchorElement;
    expect(link.textContent).toBe("Source code");
    expect(link.getAttribute("href")).toBe("https://github.com/cipherbuilds/zitadel");
    expect(link.getAttribute("rel")).toBe("noopener noreferrer");
  });

  it("is rendered by every root layout and the global error page", () => {
    const found = roots(APP).map((path) => relative(APP, path)).sort();
    expect(found).toEqual(expect.arrayContaining(["(login)/layout.tsx", "global-error.tsx"]));
    for (const name of found) {
      const source = readFileSync(join(APP, name), "utf8");
      expect(source, name).toContain('from "@/components/copperholt-source-offer"');
      expect(source, name).toContain("<CopperholtSourceOffer />");
    }
    // The layout's loading fallback and its loaded page both carry it.
    const layout = readFileSync(join(APP, "(login)/layout.tsx"), "utf8");
    expect(layout.split("<CopperholtSourceOffer />").length - 1).toBe(2);
  });
});
