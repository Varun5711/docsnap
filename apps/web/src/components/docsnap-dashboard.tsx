"use client";
import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import { ArrowUpRight, Search } from "lucide-react";
import {
  confirmAnchor,
  Evidence,
  fetchScreenshotObjectUrl,
  prepareAnchor,
  searchClaims,
  SearchResult,
  verifyEvidence,
  VerifyResult,
} from "@/lib/api";
import { connectWallet, sendAnchorTx } from "@/lib/wallet";
import { useAuth } from "@/lib/auth";
import { DomainTrustBadge } from "@/components/domain-trust-badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { EvidenceStatusBadge } from "@/components/evidence-status-badge";
import { itemVariants, staggerChildrenVariants } from "@/lib/utils/animations";
const emptyResult: SearchResult = { claims: [], items: [] };
function displayUrl(url: string): string {
  try {
    const parsed = new URL(url);
    const label =
      parsed.hostname.replace(/^www\./, "") +
      parsed.pathname.replace(/\/$/, "");
    return label.length > 44 ? label.slice(0, 44) + "…" : label;
  } catch {
    return url.length > 44 ? url.slice(0, 44) + "…" : url;
  }
}
function StatTile({ label, value }: { label: string; value: number }) {
  return (
    <motion.div
      variants={itemVariants}
      className="liquid-glass relative overflow-hidden p-6 transition-all duration-300 hover:-translate-y-0.5"
    >
      <p className="mb-3 text-[11px] font-semibold uppercase tracking-widest text-muted-foreground/60">
        {label}
      </p>
      <p className="text-[2.25rem] font-bold leading-none tracking-tight text-foreground tabular-nums">
        {value}
      </p>
    </motion.div>
  );
}
export function DocSnapDashboard() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();
  useEffect(() => {
    if (!authLoading && !user) router.push("/login");
  }, [authLoading, user, router]);
  const [query, setQuery] = useState("");
  const [company, setCompany] = useState("");
  const [domain, setDomain] = useState("");
  const [status, setStatus] = useState("");
  const [results, setResults] = useState<SearchResult>(emptyResult);
  const [selected, setSelected] = useState<Evidence | null>(null);
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [screenshotSrc, setScreenshotSrc] = useState<string | null>(null);
  const [anchoring, setAnchoring] = useState(false);
  const [anchorError, setAnchorError] = useState("");
  const [pendingTxHash, setPendingTxHash] = useState<string | null>(null);
  const params = useMemo(() => {
    const values = new URLSearchParams();
    if (query) values.set("q", query);
    if (company) values.set("company", company);
    if (domain) values.set("domain", domain);
    if (status) values.set("status", status);
    return values;
  }, [query, company, domain, status]);
  async function load() {
    setLoading(true);
    try {
      const data = await searchClaims(params);
      setResults(data);
      if (!selected && data.items.length > 0) {
        setSelected(data.items[0]);
      }
    } finally {
      setLoading(false);
    }
  }
  async function runVerify(tamper: boolean) {
    if (!selected) return;
    const result = await verifyEvidence(
      tamper
        ? {
            evidenceId: selected.id,
            scrapedText: `${selected.scrapedText} modified`,
          }
        : { evidenceId: selected.id },
    );
    setVerifyResult(result);
    if (!tamper) {
      await load();
    }
  }
  async function onAnchorWithWallet() {
    if (!selected) return;
    setAnchoring(true);
    setAnchorError("");
    setPendingTxHash(null);
    try {
      const address = await connectWallet();
      const calldata = await prepareAnchor(selected.id);
      const txHash = await sendAnchorTx(address, calldata.to, calldata.data);
      setPendingTxHash(txHash);
      const updated = await confirmAnchor(selected.id, txHash);
      setSelected(updated);
      setPendingTxHash(null);
      await load();
    } catch (err) {
      setAnchorError(err instanceof Error ? err.message : "Anchoring failed");
    } finally {
      setAnchoring(false);
    }
  }
  async function onRecheckAnchor() {
    if (!selected || !pendingTxHash) return;
    setAnchoring(true);
    setAnchorError("");
    try {
      const updated = await confirmAnchor(selected.id, pendingTxHash);
      setSelected(updated);
      setPendingTxHash(null);
      await load();
    } catch (err) {
      setAnchorError(err instanceof Error ? err.message : "Anchoring failed");
    } finally {
      setAnchoring(false);
    }
  }
  useEffect(() => {
    const timer = setTimeout(() => {
      void load();
    }, 300);
    return () => clearTimeout(timer);
  }, [params]);
  useEffect(() => {
    if (!selected) {
      setScreenshotSrc(null);
      return;
    }
    if (selected.screenshotDataUrl) {
      setScreenshotSrc(selected.screenshotDataUrl);
      return;
    }
    let cancelled = false;
    let objectUrl: string | null = null;
    fetchScreenshotObjectUrl(selected.id)
      .then((url) => {
        if (cancelled) {
          URL.revokeObjectURL(url);
          return;
        }
        objectUrl = url;
        setScreenshotSrc(url);
      })
      .catch(() => {
        if (!cancelled) setScreenshotSrc(null);
      });
    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [selected]);
  const verifiedCount = results.items.filter(
    (item) => item.verificationStatus === "verified",
  ).length;
  const certifiedCount = results.items.filter(
    (item) => item.verificationStatus === "certified",
  ).length;
  const tamperedCount = results.items.filter(
    (item) => item.verificationStatus === "tampered",
  ).length;
  if (authLoading || !user) {
    return (
      <main className="mx-auto max-w-7xl space-y-5 px-6 py-10">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-32 w-full" />
      </main>
    );
  }
  return (
    <main className="min-h-screen">
      <div className="border-b border-white/[0.06]">
        <div className="mx-auto max-w-7xl px-6 py-4">
          <div className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground/60">
            My Work
          </div>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight text-foreground">
            Your captures, claims, and verification tools
          </h1>
        </div>
      </div>

      <div className="mx-auto max-w-7xl space-y-5 px-6 py-6">
        <motion.div
          className="grid grid-cols-2 gap-4 lg:grid-cols-4"
          variants={staggerChildrenVariants}
          initial="hidden"
          animate="visible"
        >
          <StatTile label="Evidence" value={results.items.length} />
          <StatTile label="Certified" value={certifiedCount} />
          <StatTile label="Verified" value={verifiedCount} />
          <StatTile label="Tampered" value={tamperedCount} />
        </motion.div>

        <div className="grid grid-cols-12 items-stretch gap-5">
          <aside className="col-span-12 lg:col-span-3">
            <Card className="flex h-full flex-col">
              <CardHeader className="flex-row items-center justify-between space-y-0 p-4">
                <CardTitle className="text-sm font-semibold">
                  Claim Search
                </CardTitle>
                <span className="text-xs text-muted-foreground/60">
                  {results.claims.length} claims
                </span>
              </CardHeader>
              <CardContent className="space-y-3 p-4 pt-0">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground/70">
                    Keyword
                  </label>
                  <Input
                    className="mt-1"
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                    placeholder="refund, approved, price"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground/70">
                    Company
                  </label>
                  <Input
                    className="mt-1"
                    value={company}
                    onChange={(event) => setCompany(event.target.value)}
                    placeholder="ExampleCo"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground/70">
                    Domain
                  </label>
                  <Input
                    className="mt-1"
                    value={domain}
                    onChange={(event) => setDomain(event.target.value)}
                    placeholder="example.com"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground/70">
                    Status
                  </label>
                  <select
                    className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    value={status}
                    onChange={(event) => setStatus(event.target.value)}
                  >
                    <option value="">Any</option>
                    <option value="certified">Certified</option>
                    <option value="verified">Verified</option>
                    <option value="tampered">Tampered</option>
                  </select>
                </div>
              </CardContent>
            </Card>
          </aside>

          <section className="col-span-12 lg:col-span-6">
            <div className="liquid-glass relative flex h-full flex-col overflow-hidden transition-all duration-300 hover:-translate-y-0.5">
              <div className="flex items-center justify-between border-b border-white/[0.06] px-4 py-3">
                <h2 className="text-sm font-semibold text-foreground">
                  Extracted Claims
                </h2>
                <span className="text-xs text-muted-foreground/60">
                  {loading ? "Syncing" : "Live index"}
                </span>
              </div>
              {loading && results.claims.length === 0 ? (
                <div className="space-y-3 p-4">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-16 w-full" />
                  ))}
                </div>
              ) : (
                <motion.div
                  variants={staggerChildrenVariants}
                  initial="hidden"
                  animate="visible"
                  className="min-h-[420px] flex-1 overflow-y-auto"
                >
                  {results.claims.length === 0 && (
                    <div className="px-4 py-12 text-center text-sm text-muted-foreground">
                      No claims found. Capture a webpage or adjust the filters.
                    </div>
                  )}
                  {results.claims.map((claim) => {
                    const evidence = results.items.find(
                      (item) => item.id === claim.evidenceId,
                    );
                    return (
                      <motion.details
                        key={claim.id}
                        variants={itemVariants}
                        className="group border-b border-white/[0.04] transition-colors last:border-0 open:bg-white/[0.03]"
                      >
                        <summary
                          className="flex cursor-pointer list-none items-center gap-3 px-4 py-3 hover:bg-white/[0.03]"
                          onClick={() => evidence && setSelected(evidence)}
                        >
                          <p className="flex-1 truncate text-sm font-medium text-foreground/90">
                            {claim.text}
                          </p>
                          <Badge variant="secondary" className="shrink-0">
                            {claim.type}
                          </Badge>
                        </summary>
                        <div className="px-4 pb-4">
                          <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground/70">
                            <span>
                              {evidence?.company || "Unknown company"}
                            </span>
                            <span>{evidence?.domain || "unknown domain"}</span>
                            <span>
                              {Math.round(claim.confidence * 100)}% confidence
                            </span>
                            {evidence?.domain && (
                              <DomainTrustBadge domain={evidence.domain} />
                            )}
                          </div>
                          <Link
                            href={`/investigations/${claim.id}`}
                            target="_blank"
                            className="mt-2 inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
                          >
                            <Search className="h-3 w-3" /> Investigate this
                            claim
                          </Link>
                        </div>
                      </motion.details>
                    );
                  })}
                </motion.div>
              )}
            </div>
          </section>

          <aside className="col-span-12 lg:col-span-3">
            <Card className="flex h-full flex-col">
              <CardHeader className="p-4">
                <CardTitle className="text-sm font-semibold">
                  Verification
                </CardTitle>
              </CardHeader>
              <CardContent className="flex-1 overflow-y-auto p-4 pt-0">
                {!selected && (
                  <p className="text-sm text-muted-foreground">
                    Select evidence to inspect commitments and run verification.
                  </p>
                )}
                {selected && (
                  <div className="space-y-4">
                    {screenshotSrc ? (
                      <img
                        src={screenshotSrc}
                        alt="Captured screenshot"
                        className="aspect-video w-full rounded-lg border border-white/[0.08] object-cover"
                      />
                    ) : (
                      <div className="flex aspect-video w-full items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.02] text-xs text-muted-foreground/60">
                        No screenshot
                      </div>
                    )}
                    <div>
                      <div className="text-xs font-medium text-muted-foreground/70">
                        Source
                      </div>
                      <a
                        href={selected.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="mt-1 flex items-center gap-1 text-sm text-primary hover:underline"
                      >
                        <span className="truncate">
                          {displayUrl(selected.url)}
                        </span>
                        <ArrowUpRight className="h-3.5 w-3.5 shrink-0" />
                      </a>
                      {selected.domain && (
                        <div className="mt-2">
                          <DomainTrustBadge domain={selected.domain} />
                        </div>
                      )}
                    </div>
                    <HashLine
                      label="Evidence"
                      value={selected.evidenceCommitment}
                    />
                    <HashLine label="Claims root" value={selected.claimsRoot} />
                    <HashLine label="Flare tx" value={selected.flareTxHash} />
                    <HashLine
                      label="TEE cert"
                      value={selected.teeCertificateHash}
                    />
                    {selected.anchorSubmitter && (
                      <div>
                        <div className="text-xs font-medium text-muted-foreground/70">
                          Anchored by
                        </div>
                        <div className="hash mt-1 break-all text-xs text-foreground/80">
                          {selected.anchorSubmitter}
                        </div>
                      </div>
                    )}
                    {selected.verificationStatus === "pending_wallet_anchor" &&
                      selected.publishedBy === user?.id && (
                        <div className="rounded-lg border border-amber-500/40 bg-amber-500/5 p-3">
                          <p className="text-xs text-amber-400">
                            Not on-chain yet — anchor it with your own wallet to
                            certify this evidence.
                          </p>
                          <Button
                            size="sm"
                            className="mt-2 w-full"
                            onClick={onAnchorWithWallet}
                            disabled={anchoring}
                          >
                            {anchoring ? "Anchoring…" : "Anchor with my wallet"}
                          </Button>
                          {anchorError && (
                            <div className="mt-2">
                              <p className="text-xs text-red-400">
                                {anchorError}
                              </p>
                              {pendingTxHash && (
                                <Button
                                  size="sm"
                                  variant="outline"
                                  className="mt-2 w-full"
                                  onClick={onRecheckAnchor}
                                  disabled={anchoring}
                                >
                                  {anchoring ? "Checking…" : "Recheck status"}
                                </Button>
                              )}
                            </div>
                          )}
                        </div>
                      )}
                    <div className="grid grid-cols-2 gap-2">
                      <Button
                        variant="outline"
                        onClick={() => runVerify(false)}
                      >
                        Verify
                      </Button>
                      <Button
                        variant="destructive"
                        onClick={() => runVerify(true)}
                      >
                        Tamper Test
                      </Button>
                    </div>
                    {verifyResult && (
                      <div
                        className={`rounded-lg border px-3 py-3 text-sm ${verifyResult.verified ? "border-green-500/40 text-green-400" : "border-destructive/60 text-red-400"}`}
                      >
                        <div className="font-semibold">
                          {verifyResult.verified
                            ? "Verification passed"
                            : "Verification failed"}
                        </div>
                        <div className="hash mt-2 break-all text-xs text-muted-foreground/70">
                          {verifyResult.actualCommitment}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          </aside>

          <section className="col-span-12">
            <div className="liquid-glass overflow-hidden">
              <div className="border-b border-white/[0.06] px-4 py-3">
                <h2 className="text-sm font-semibold text-foreground">
                  Evidence Stream
                </h2>
              </div>
              <motion.div
                variants={staggerChildrenVariants}
                initial="hidden"
                animate="visible"
              >
                {results.items.map((item) => (
                  <motion.button
                    key={item.id}
                    variants={itemVariants}
                    className="grid w-full grid-cols-[1fr_auto] items-center gap-3 border-b border-white/[0.04] px-4 py-3 text-left transition-colors last:border-0 hover:bg-white/[0.03]"
                    onClick={() => setSelected(item)}
                  >
                    <span>
                      <span className="block text-sm font-medium text-foreground/90">
                        {item.title || item.url}
                      </span>
                      <span className="mt-1 block text-xs text-muted-foreground/60">
                        {item.company} · {item.caseId} ·{" "}
                        {new Date(item.createdAt).toLocaleString()}
                      </span>
                    </span>
                    <EvidenceStatusBadge status={item.verificationStatus} />
                  </motion.button>
                ))}
              </motion.div>
            </div>
          </section>
        </div>
      </div>
    </main>
  );
}
function HashLine({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs font-medium text-muted-foreground/70">
        {label}
      </div>
      <div className="hash mt-1 break-all rounded-md border border-white/[0.06] bg-white/[0.02] px-2 py-2 text-xs text-muted-foreground">
        {value || "pending"}
      </div>
    </div>
  );
}
