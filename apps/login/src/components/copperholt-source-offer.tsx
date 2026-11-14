// Modified by Copperholt Ltd on 2026-11-14: every sign-in page shows the AGPL-3.0 source offer with a link to this repository.

// AGPL-3.0 section 13 source offer (LIC-FIX01). Compliance FINAL wording of 14 Nov 2026, unchanged:
// shown on every page this modified login UI serves, with "Source code" linking to the published fork.
export const COPPERHOLT_SOURCE_OFFER =
  "This sign-in service runs a modified version of ZITADEL, free software under the GNU Affero General Public " +
  "License v3.0. Get the source code of the version running here, free of charge:";
export const COPPERHOLT_SOURCE_LABEL = "Source code";
export const COPPERHOLT_SOURCE_URL = "https://github.com/cipherbuilds/zitadel";

export function CopperholtSourceOffer() {
  return (
    <footer
      className="mx-auto max-w-[440px] px-4 pb-4 text-center text-xs text-gray-600 dark:text-gray-400 md:max-w-full md:px-8"
      data-testid="copperholt-source-offer"
    >
      <p>
        {COPPERHOLT_SOURCE_OFFER}{" "}
        <a
          href={COPPERHOLT_SOURCE_URL}
          className="underline"
          target="_blank"
          rel="noopener noreferrer"
          data-testid="copperholt-source-link"
        >
          {COPPERHOLT_SOURCE_LABEL}
        </a>
      </p>
    </footer>
  );
}
