// Modified by Copperholt Ltd on 2026-11-14: every sign-in page shows the AGPL-3.0 source offer with a link to this repository.

"use client";

import { Boundary } from "@/components/boundary";
import { Button } from "@/components/button";
import { CopperholtSourceOffer } from "@/components/copperholt-source-offer";
import { ThemeWrapper } from "@/components/theme-wrapper";
import { Translated } from "@/components/translated";

export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    // global-error must include html and body tags
    <html>
      <body>
        <ThemeWrapper branding={undefined}>
          <Boundary labels={["Login Error"]} color="red">
            <div className="space-y-4">
              <div className="text-sm text-red-500 dark:text-red-500">
                <span className="font-bold">Error:</span> {error?.message}
              </div>
              <div>
                <Button data-i18n-key="error.tryagain" onClick={() => reset()}>
                  <Translated i18nKey="tryagain" namespace="error" />
                </Button>
              </div>
            </div>
          </Boundary>
          <CopperholtSourceOffer />
        </ThemeWrapper>
      </body>
    </html>
  );
}
