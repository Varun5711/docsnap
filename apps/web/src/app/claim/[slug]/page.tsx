"use client";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { getCanonicalClaim, CanonicalClaim } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { VerdictBadge } from "@/components/verdict-badge";
export default function CanonicalClaimPage() {
  const params = useParams<{
    slug: string;
  }>();
  const [data, setData] = useState<CanonicalClaim | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  useEffect(() => {
    getCanonicalClaim(params.slug)
      .then(setData)
      .catch(() => setError("Claim not found."))
      .finally(() => setLoading(false));
  }, [params.slug]);
  if (loading) {
    return (
      <main className="mx-auto max-w-3xl space-y-4 px-6 py-16">
        <Skeleton className="h-8 w-2/3" />
        <Skeleton className="h-40 w-full" />
      </main>
    );
  }
  if (error || !data) {
    return (
      <main className="mx-auto max-w-3xl px-6 py-16 text-center text-sm text-muted-foreground">
        {error || "Claim not found."}
      </main>
    );
  }
  const supported = data.claims.filter(
    (c) =>
      c.investigationStatus === "SUPPORTED" ||
      c.investigationStatus === "LIKELY_SUPPORTED",
  ).length;
  const contradicted = data.claims.filter(
    (c) =>
      c.investigationStatus === "CONTRADICTED" ||
      c.investigationStatus === "LIKELY_CONTRADICTED",
  ).length;
  const mixed = data.claims.filter(
    (c) => c.investigationStatus === "MIXED",
  ).length;
  const totalSources = data.claims.reduce(
    (sum, c) => sum + (c.sources?.length ?? 0),
    0,
  );
  return (
    <main className="mx-auto max-w-3xl space-y-5 px-6 py-16">
      <div>
        <div className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground/60">
          Claim
        </div>
        <h1 className="mt-2 text-2xl font-semibold leading-relaxed text-foreground">
          "{data.text}"
        </h1>
      </div>

      <Card>
        <CardContent className="grid grid-cols-2 gap-4 p-6 sm:grid-cols-4">
          <Stat label="Investigations" value={data.claims.length} />
          <Stat label="Sources" value={totalSources} />
          <Stat label="Supported" value={supported} />
          <Stat label="Contradicted" value={contradicted} />
        </CardContent>
      </Card>

      <div>
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-widest text-muted-foreground/60">
          Investigations ({data.claims.length})
        </h2>
        {mixed > 0 && (
          <p className="mb-3 text-xs text-amber-300">
            ⚠ Evidence is mixed across independent investigations — read each
            before drawing a conclusion.
          </p>
        )}
        <div className="space-y-3">
          {data.claims.map((c) => (
            <Link
              key={c.id}
              href={`/investigations/${c.id}`}
              className="liquid-glass block p-4 transition-all duration-200 hover:-translate-y-0.5"
            >
              <div className="flex items-center justify-between">
                <VerdictBadge status={c.investigationStatus || "UNVERIFIED"} />
                <span className="text-xs text-muted-foreground">
                  {Math.round((c.investigationConfidence ?? 0) * 100)}%
                  confidence
                </span>
              </div>
              <p className="mt-2 text-xs text-muted-foreground">
                {c.sources?.length ?? 0} sources
                {c.forkedFromClaimId && " · built on an earlier investigation"}
              </p>
            </Link>
          ))}
        </div>
      </div>

      <p className="text-xs text-muted-foreground">
        Note: independent-source counting (detecting when many articles trace
        back to one press release) isn't implemented yet — source counts above
        are raw, not deduplicated by provenance.
      </p>
    </main>
  );
}
function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div>
      <p className="text-[11px] font-semibold uppercase tracking-widest text-muted-foreground/60">
        {label}
      </p>
      <p className="mt-1 text-2xl font-bold text-foreground tabular-nums">
        {value}
      </p>
    </div>
  );
}
