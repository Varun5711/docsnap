import { getProof } from "@/lib/api";
const API_BASE =
  process.env.NEXT_PUBLIC_DOCSNAP_API_URL ?? "http://localhost:8080";
function HashLine({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs font-medium text-muted-foreground/70">
        {label}
      </div>
      <div className="hash mt-1 break-all rounded-md border border-white/[0.06] bg-white/[0.02] px-3 py-2 text-xs text-muted-foreground">
        {value || "pending"}
      </div>
    </div>
  );
}
export default async function ProofPage({
  params,
}: {
  params: Promise<{
    id: string;
  }>;
}) {
  const { id } = await params;
  let proof;
  try {
    proof = await getProof(API_BASE, id);
  } catch {
    return (
      <main className="mx-auto max-w-2xl px-6 py-16 text-center text-sm text-muted-foreground">
        No evidence found for this proof id.
      </main>
    );
  }
  return (
    <main className="mx-auto max-w-6xl space-y-5 px-6 py-16">
      <div>
        <div className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground/60">
          DocSnap Proof
        </div>
        <h1 className="mt-2 text-2xl font-semibold text-foreground">
          Independently verifiable evidence record
        </h1>
        <p className="mt-2 text-base text-muted-foreground">
          This page needs no login and no trust in DocSnap&apos;s own UI —
          everything below can be independently re-derived and checked
          against what&apos;s anchored on Coston2.
        </p>
      </div>

      <div className="grid grid-cols-12 items-stretch gap-5">
        <div className="col-span-12 lg:col-span-7">
          <div className="liquid-glass flex h-full flex-col p-6">
            <div className="text-sm font-semibold text-foreground">
              Screenshot
            </div>
            <div className="mt-3 flex-1">
              {proof.screenshotDataUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={proof.screenshotDataUrl}
                  alt="Captured screenshot"
                  className="w-full rounded-lg border border-white/[0.08] object-cover"
                />
              ) : (
                <div className="flex h-full min-h-[240px] items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.02] text-sm text-muted-foreground/60">
                  No screenshot for this capture
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="col-span-12 lg:col-span-5">
          <div className="liquid-glass flex h-full flex-col space-y-4 p-6">
            <div>
              <div className="text-xs font-medium text-muted-foreground/70">
                Source
              </div>
              <a
                href={proof.url}
                target="_blank"
                rel="noopener noreferrer"
                className="mt-1 block break-all text-sm text-primary hover:underline"
              >
                {proof.url}
              </a>
            </div>
            <div>
              <div className="text-xs font-medium text-muted-foreground/70">
                Captured at
              </div>
              <div className="mt-1 text-base text-foreground/90">
                {new Date(proof.capturedAt).toUTCString()}
              </div>
            </div>
            <div>
              <div className="text-xs font-medium text-muted-foreground/70">
                Status
              </div>
              <div className="mt-1 text-base capitalize text-foreground/90">
                {proof.verificationStatus}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="liquid-glass p-6">
        <div className="mb-4 text-sm font-semibold text-foreground">
          Cryptographic Commitments
        </div>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <HashLine label="Screenshot hash" value={proof.screenshotHash} />
          <HashLine
            label="Scraped text hash"
            value={proof.scrapedTextHash}
          />
          <HashLine
            label="Metadata commitment"
            value={proof.metadataCommitment}
          />
          <HashLine label="Claims root" value={proof.claimsRoot} />
          <HashLine
            label="Evidence commitment"
            value={proof.evidenceCommitment}
          />
          <HashLine label="Flare transaction" value={proof.flareTxHash} />
          <HashLine
            label="TEE certificate hash"
            value={proof.teeCertificateHash}
          />
        </div>
      </div>

      <div className="liquid-glass space-y-2 p-6 text-base text-muted-foreground">
        <p className="font-semibold text-foreground">Verify independently</p>
        <p>
          The evidence commitment is{" "}
          <code className="hash">
            sha256(screenshotHash | scrapedTextHash | metadataCommitment |
            claimsRoot)
          </code>
          . Recompute it from the values above — if it matches{" "}
          <code className="hash">evidenceCommitment</code>, nothing has been
          altered since capture. Then check the Flare transaction hash on the{" "}
          <a
            href="https://coston2-explorer.flare.network"
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline"
          >
            Coston2 explorer
          </a>{" "}
          to confirm this exact commitment was anchored on-chain.
        </p>
        <p>
          DocSnap proves what was captured, when, and whether it&apos;s been
          modified since — it does not claim everything on the original page
          is true. See the linked investigation for the evidence-based
          verdict on the claim itself.
        </p>
      </div>
    </main>
  );
}
