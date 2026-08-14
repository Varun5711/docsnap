"use client";
import { useEffect, useState } from "react";
import { DomainTrust, getDomainTrust } from "@/lib/api";
const trustCache = new Map<string, Promise<DomainTrust>>();
function fetchDomainTrust(domain: string): Promise<DomainTrust> {
  let cached = trustCache.get(domain);
  if (!cached) {
    cached = getDomainTrust(domain);
    trustCache.set(domain, cached);
    cached.catch(() => trustCache.delete(domain));
  }
  return cached;
}
export function DomainTrustBadge({ domain }: { domain: string }) {
  const [trust, setTrust] = useState<DomainTrust | null>(null);
  useEffect(() => {
    let cancelled = false;
    if (!domain || domain === "unknown domain") return;
    fetchDomainTrust(domain)
      .then((result) => !cancelled && setTrust(result))
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [domain]);
  if (!trust || trust.label === "none") return null;
  const isLowTrust = trust.label === "low_trust";
  return (
    <span
      title={`${trust.contradicted}/${trust.totalInvestigated} investigated claims from this domain came back contradicted`}
      className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-medium ${isLowTrust ? "border-red-500/40 bg-red-500/10 text-red-400" : "border-amber-500/40 bg-amber-500/10 text-amber-400"}`}
    >
      {isLowTrust ? "⚠ Repeatedly contradicted" : "◐ Inconsistent reporting"}
    </span>
  );
}
