"use client";

import { useEffect, useMemo, useState } from "react";
import { motion } from "framer-motion";
import { createCapture, Evidence, screenshotUrl, searchClaims, SearchResult, verifyEvidence, VerifyResult } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { EvidenceStatusBadge } from "@/components/evidence-status-badge";
import { itemVariants, staggerChildrenVariants } from "@/lib/utils/animations";

const emptyResult: SearchResult = { claims: [], items: [] };

export function DocSnapDashboard() {
  const [query, setQuery] = useState("");
  const [company, setCompany] = useState("");
  const [domain, setDomain] = useState("");
  const [status, setStatus] = useState("");
  const [results, setResults] = useState<SearchResult>(emptyResult);
  const [selected, setSelected] = useState<Evidence | null>(null);
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [capture, setCapture] = useState({
    url: "https://example.com/pricing",
    title: "Pricing and compliance claims",
    company: "ExampleCo",
    caseId: "CASE-2026-001",
    userId: "auditor@docsnap.local",
    scrapedText: "ExampleCo states that its enterprise plan is SOC 2 compliant, includes encrypted evidence storage, and offers a 30 day refund policy at $99 per month.",
    screenshotDataUrl: ""
  });

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

  async function submitCapture() {
    setLoading(true);
    setVerifyResult(null);
    try {
      const item = await createCapture({
        ...capture,
        capturedAt: new Date().toISOString()
      });
      setSelected(item);
      await load();
    } finally {
      setLoading(false);
    }
  }

  async function runVerify(tamper: boolean) {
    if (!selected) return;
    // A real Verify sends nothing but the id — the server checks its own
    // stored data, and only that "pure" check persists a status back to the
    // row. Tamper Test deliberately submits modified text so the server
    // recomputes a mismatch; that's a synthetic drill, not a real state
    // change, so it stays ephemeral (server won't persist it either way).
    const result = await verifyEvidence(
      tamper
        ? { evidenceId: selected.id, scrapedText: `${selected.scrapedText} modified` }
        : { evidenceId: selected.id }
    );
    setVerifyResult(result);
    if (!tamper) {
      await load();
    }
  }

  useEffect(() => {
    void load();
  }, [params]);

  return (
    <main className="min-h-screen">
      <div className="border-b border-white/[0.06]">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
          <div>
            <div className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground/60">DocSnap</div>
            <h1 className="mt-1 text-2xl font-semibold tracking-tight text-foreground">Searchable web claims certified through Flare FCC</h1>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant="outline">Coston2 ready</Badge>
            <Badge variant="outline">TEE certificate</Badge>
          </div>
        </div>
      </div>

      <div className="mx-auto grid max-w-7xl grid-cols-12 gap-5 px-6 py-6">
        <aside className="col-span-12 space-y-5 lg:col-span-3">
          <Card>
            <CardHeader className="flex-row items-center justify-between space-y-0 p-4">
              <CardTitle className="text-sm font-semibold">Claim Search</CardTitle>
              <span className="text-xs text-muted-foreground/60">{results.claims.length} claims</span>
            </CardHeader>
            <CardContent className="space-y-3 p-4 pt-0">
              <div>
                <label className="block text-xs font-medium text-muted-foreground/70">Keyword</label>
                <Input className="mt-1" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="refund, approved, price" />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground/70">Company</label>
                <Input className="mt-1" value={company} onChange={(event) => setCompany(event.target.value)} placeholder="ExampleCo" />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground/70">Domain</label>
                <Input className="mt-1" value={domain} onChange={(event) => setDomain(event.target.value)} placeholder="example.com" />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground/70">Status</label>
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

          {/* Collapsed by default — this is a manual testing tool for pasting
              fake evidence, not the primary flow (real captures come from the
              extension). Keeping it out of the way of the actual product. */}
          <Card>
            <details>
              <summary className="cursor-pointer list-none p-4 text-sm font-semibold text-foreground">
                Manual Capture <span className="ml-1 text-xs font-normal text-muted-foreground/60">(testing tool)</span>
              </summary>
              <CardContent className="space-y-3 p-4 pt-0">
                <Input value={capture.url} onChange={(event) => setCapture({ ...capture, url: event.target.value })} />
                <Input value={capture.company} onChange={(event) => setCapture({ ...capture, company: event.target.value })} />
                <Input value={capture.caseId} onChange={(event) => setCapture({ ...capture, caseId: event.target.value })} />
                <textarea
                  className="h-32 w-full resize-none rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  value={capture.scrapedText}
                  onChange={(event) => setCapture({ ...capture, scrapedText: event.target.value })}
                />
                <Button className="w-full" onClick={submitCapture} disabled={loading}>
                  {loading ? "Working" : "Capture Evidence"}
                </Button>
              </CardContent>
            </details>
          </Card>
        </aside>

        <section className="col-span-12 space-y-5 lg:col-span-6">
          <div className="liquid-glass relative overflow-hidden transition-all duration-300 hover:-translate-y-0.5">
            <div className="flex items-center justify-between border-b border-white/[0.06] px-4 py-3">
              <h2 className="text-sm font-semibold text-foreground">Extracted Claims</h2>
              <span className="text-xs text-muted-foreground/60">{loading ? "Syncing" : "Live index"}</span>
            </div>
            {loading && results.claims.length === 0 ? (
              <div className="space-y-3 p-4">
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className="h-16 w-full" />
                ))}
              </div>
            ) : (
              <motion.div variants={staggerChildrenVariants} initial="hidden" animate="visible">
                {results.claims.length === 0 && (
                  <div className="px-4 py-12 text-center text-sm text-muted-foreground">No claims found. Capture a webpage or adjust the filters.</div>
                )}
                {results.claims.map((claim) => {
                  const evidence = results.items.find((item) => item.id === claim.evidenceId);
                  return (
                    <motion.button
                      key={claim.id}
                      variants={itemVariants}
                      className="block w-full border-b border-white/[0.04] px-4 py-4 text-left transition-colors last:border-0 hover:bg-white/[0.03]"
                      onClick={() => evidence && setSelected(evidence)}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <p className="text-sm font-medium leading-6 text-foreground/90">{claim.text}</p>
                        <Badge variant="secondary" className="shrink-0">{claim.type}</Badge>
                      </div>
                      <div className="mt-3 flex flex-wrap gap-3 text-xs text-muted-foreground/70">
                        <span>{evidence?.company || "Unknown company"}</span>
                        <span>{evidence?.domain || "unknown domain"}</span>
                        <span>{Math.round(claim.confidence * 100)}% confidence</span>
                      </div>
                    </motion.button>
                  );
                })}
              </motion.div>
            )}
          </div>

          <div className="liquid-glass overflow-hidden">
            <div className="border-b border-white/[0.06] px-4 py-3">
              <h2 className="text-sm font-semibold text-foreground">Evidence Stream</h2>
            </div>
            <motion.div variants={staggerChildrenVariants} initial="hidden" animate="visible">
              {results.items.map((item) => (
                <motion.button
                  key={item.id}
                  variants={itemVariants}
                  className="grid w-full grid-cols-[1fr_auto] items-center gap-3 border-b border-white/[0.04] px-4 py-3 text-left transition-colors last:border-0 hover:bg-white/[0.03]"
                  onClick={() => setSelected(item)}
                >
                  <span>
                    <span className="block text-sm font-medium text-foreground/90">{item.title || item.url}</span>
                    <span className="mt-1 block text-xs text-muted-foreground/60">{item.company} · {item.caseId} · {new Date(item.createdAt).toLocaleString()}</span>
                  </span>
                  <EvidenceStatusBadge status={item.verificationStatus} />
                </motion.button>
              ))}
            </motion.div>
          </div>
        </section>

        <aside className="col-span-12 lg:col-span-3">
          <Card>
            <CardHeader className="p-4">
              <CardTitle className="text-sm font-semibold">Verification</CardTitle>
            </CardHeader>
            <CardContent className="p-4 pt-0">
              {!selected && <p className="text-sm text-muted-foreground">Select evidence to inspect commitments and run verification.</p>}
              {selected && (
                <div className="space-y-4">
                  <img src={selected.screenshotDataUrl || screenshotUrl(selected.id)} alt="Captured screenshot" className="aspect-video w-full rounded-lg border border-white/[0.08] object-cover" />
                  <div>
                    <div className="text-xs font-medium text-muted-foreground/70">Source</div>
                    <div className="mt-1 break-words text-sm text-foreground/90">{selected.url}</div>
                  </div>
                  <HashLine label="Evidence" value={selected.evidenceCommitment} />
                  <HashLine label="Claims root" value={selected.claimsRoot} />
                  <HashLine label="Flare tx" value={selected.flareTxHash} />
                  <HashLine label="TEE cert" value={selected.teeCertificateHash} />
                  <div className="grid grid-cols-2 gap-2">
                    <Button variant="outline" onClick={() => runVerify(false)}>Verify</Button>
                    <Button variant="destructive" onClick={() => runVerify(true)}>Tamper Test</Button>
                  </div>
                  {verifyResult && (
                    <div className={`rounded-lg border px-3 py-3 text-sm ${verifyResult.verified ? "border-green-500/40 text-green-400" : "border-destructive/60 text-red-400"}`}>
                      <div className="font-semibold">{verifyResult.verified ? "Verification passed" : "Verification failed"}</div>
                      <div className="hash mt-2 break-all text-xs text-muted-foreground/70">{verifyResult.actualCommitment}</div>
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </aside>
      </div>
    </main>
  );
}

function HashLine({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs font-medium text-muted-foreground/70">{label}</div>
      <div className="hash mt-1 break-all rounded-md border border-white/[0.06] bg-white/[0.02] px-2 py-2 text-xs text-muted-foreground">{value || "pending"}</div>
    </div>
  );
}
