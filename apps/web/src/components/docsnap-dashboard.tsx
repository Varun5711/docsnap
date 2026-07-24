"use client";

import { useEffect, useMemo, useState } from "react";
import { createCapture, Evidence, searchClaims, SearchResult, verifyEvidence, VerifyResult } from "@/lib/api";

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
    const result = await verifyEvidence({
      evidenceId: selected.id,
      scrapedText: tamper ? `${selected.scrapedText} modified` : selected.scrapedText,
      screenshotDataUrl: selected.screenshotDataUrl,
      claims: selected.claims
    });
    setVerifyResult(result);
  }

  useEffect(() => {
    void load();
  }, [params]);

  return (
    <main className="min-h-screen">
      <div className="border-b border-[var(--line)] bg-[var(--surface)]">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
          <div>
            <div className="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--seal)]">DocSnap</div>
            <h1 className="mt-1 text-2xl font-semibold tracking-normal">Searchable web claims certified through Flare FCC</h1>
          </div>
          <div className="flex items-center gap-3 text-sm text-[var(--ledger)]">
            <span className="rounded-full border border-[var(--line)] px-3 py-1">Coston2 ready</span>
            <span className="rounded-full border border-[var(--line)] px-3 py-1">TEE certificate</span>
          </div>
        </div>
      </div>

      <div className="mx-auto grid max-w-7xl grid-cols-12 gap-5 px-6 py-6">
        <aside className="col-span-12 lg:col-span-3">
          <section className="border border-[var(--line)] bg-[var(--surface)] p-4 shadow-audit">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-sm font-semibold">Claim Search</h2>
              <span className="text-xs text-[var(--ledger)]">{results.claims.length} claims</span>
            </div>
            <label className="block text-xs font-medium text-[var(--ledger)]">Keyword</label>
            <input className="mt-1 w-full border border-[var(--line)] bg-[var(--paper)] px-3 py-2 text-sm outline-none focus:border-[var(--seal)]" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="refund, approved, price" />
            <label className="mt-3 block text-xs font-medium text-[var(--ledger)]">Company</label>
            <input className="mt-1 w-full border border-[var(--line)] bg-[var(--paper)] px-3 py-2 text-sm outline-none focus:border-[var(--seal)]" value={company} onChange={(event) => setCompany(event.target.value)} placeholder="ExampleCo" />
            <label className="mt-3 block text-xs font-medium text-[var(--ledger)]">Domain</label>
            <input className="mt-1 w-full border border-[var(--line)] bg-[var(--paper)] px-3 py-2 text-sm outline-none focus:border-[var(--seal)]" value={domain} onChange={(event) => setDomain(event.target.value)} placeholder="example.com" />
            <label className="mt-3 block text-xs font-medium text-[var(--ledger)]">Status</label>
            <select className="mt-1 w-full border border-[var(--line)] bg-[var(--paper)] px-3 py-2 text-sm outline-none focus:border-[var(--seal)]" value={status} onChange={(event) => setStatus(event.target.value)}>
              <option value="">Any</option>
              <option value="certified">Certified</option>
              <option value="verified">Verified</option>
              <option value="tampered">Tampered</option>
            </select>
          </section>

          <section className="mt-5 border border-[var(--line)] bg-[var(--surface)] p-4">
            <h2 className="text-sm font-semibold">Manual Capture</h2>
            <div className="mt-3 space-y-3">
              <input className="w-full border border-[var(--line)] bg-[var(--paper)] px-3 py-2 text-sm" value={capture.url} onChange={(event) => setCapture({ ...capture, url: event.target.value })} />
              <input className="w-full border border-[var(--line)] bg-[var(--paper)] px-3 py-2 text-sm" value={capture.company} onChange={(event) => setCapture({ ...capture, company: event.target.value })} />
              <input className="w-full border border-[var(--line)] bg-[var(--paper)] px-3 py-2 text-sm" value={capture.caseId} onChange={(event) => setCapture({ ...capture, caseId: event.target.value })} />
              <textarea className="h-32 w-full resize-none border border-[var(--line)] bg-[var(--paper)] px-3 py-2 text-sm" value={capture.scrapedText} onChange={(event) => setCapture({ ...capture, scrapedText: event.target.value })} />
              <button className="w-full bg-[var(--seal)] px-3 py-2 text-sm font-semibold text-white" onClick={submitCapture} disabled={loading}>{loading ? "Working" : "Capture Evidence"}</button>
            </div>
          </section>
        </aside>

        <section className="col-span-12 lg:col-span-6">
          <div className="border border-[var(--line)] bg-[var(--surface)]">
            <div className="flex items-center justify-between border-b border-[var(--line)] px-4 py-3">
              <h2 className="text-sm font-semibold">Extracted Claims</h2>
              <span className="text-xs text-[var(--ledger)]">{loading ? "Syncing" : "Live index"}</span>
            </div>
            <div className="divide-y divide-[var(--soft-line)]">
              {results.claims.length === 0 && <div className="px-4 py-12 text-center text-sm text-[var(--ledger)]">No claims found. Capture a webpage or adjust the filters.</div>}
              {results.claims.map((claim) => {
                const evidence = results.items.find((item) => item.id === claim.evidenceId);
                return (
                  <button key={claim.id} className="block w-full px-4 py-4 text-left hover:bg-[rgba(36,93,99,0.06)]" onClick={() => evidence && setSelected(evidence)}>
                    <div className="flex items-start justify-between gap-3">
                      <p className="text-sm font-medium leading-6">{claim.text}</p>
                      <span className="shrink-0 border border-[var(--line)] px-2 py-1 text-xs text-[var(--seal)]">{claim.type}</span>
                    </div>
                    <div className="mt-3 flex flex-wrap gap-3 text-xs text-[var(--ledger)]">
                      <span>{evidence?.company || "Unknown company"}</span>
                      <span>{evidence?.domain || "unknown domain"}</span>
                      <span>{Math.round(claim.confidence * 100)}% confidence</span>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          <div className="mt-5 border border-[var(--line)] bg-[var(--surface)]">
            <div className="border-b border-[var(--line)] px-4 py-3">
              <h2 className="text-sm font-semibold">Evidence Stream</h2>
            </div>
            <div className="divide-y divide-[var(--soft-line)]">
              {results.items.map((item) => (
                <button key={item.id} className="grid w-full grid-cols-[1fr_auto] gap-3 px-4 py-3 text-left hover:bg-[rgba(36,93,99,0.06)]" onClick={() => setSelected(item)}>
                  <span>
                    <span className="block text-sm font-medium">{item.title || item.url}</span>
                    <span className="mt-1 block text-xs text-[var(--ledger)]">{item.company} · {item.caseId} · {new Date(item.createdAt).toLocaleString()}</span>
                  </span>
                  <span className="self-start border border-[var(--line)] px-2 py-1 text-xs text-[var(--seal)]">{item.verificationStatus}</span>
                </button>
              ))}
            </div>
          </div>
        </section>

        <aside className="col-span-12 lg:col-span-3">
          <section className="border border-[var(--line)] bg-[var(--surface)] p-4">
            <h2 className="text-sm font-semibold">Verification</h2>
            {!selected && <p className="mt-3 text-sm text-[var(--ledger)]">Select evidence to inspect commitments and run verification.</p>}
            {selected && (
              <div className="mt-4 space-y-4">
                {selected.screenshotDataUrl && <img src={selected.screenshotDataUrl} alt="Captured screenshot" className="aspect-video w-full border border-[var(--line)] object-cover" />}
                <div>
                  <div className="text-xs font-medium text-[var(--ledger)]">Source</div>
                  <div className="mt-1 break-words text-sm">{selected.url}</div>
                </div>
                <HashLine label="Evidence" value={selected.evidenceCommitment} />
                <HashLine label="Claims root" value={selected.claimsRoot} />
                <HashLine label="Flare tx" value={selected.flareTxHash} />
                <HashLine label="TEE cert" value={selected.teeCertificateHash} />
                <div className="grid grid-cols-2 gap-2">
                  <button className="border border-[var(--seal)] px-3 py-2 text-sm font-semibold text-[var(--seal)]" onClick={() => runVerify(false)}>Verify</button>
                  <button className="border border-[var(--wax)] px-3 py-2 text-sm font-semibold text-[var(--wax)]" onClick={() => runVerify(true)}>Tamper Test</button>
                </div>
                {verifyResult && (
                  <div className={`border px-3 py-3 text-sm ${verifyResult.verified ? "border-[var(--seal)] text-[var(--seal)]" : "border-[var(--wax)] text-[var(--wax)]"}`}>
                    <div className="font-semibold">{verifyResult.verified ? "Verification passed" : "Verification failed"}</div>
                    <div className="hash mt-2 break-all text-xs">{verifyResult.actualCommitment}</div>
                  </div>
                )}
              </div>
            )}
          </section>
        </aside>
      </div>
    </main>
  );
}

function HashLine({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs font-medium text-[var(--ledger)]">{label}</div>
      <div className="hash mt-1 break-all border border-[var(--soft-line)] bg-[var(--paper)] px-2 py-2 text-xs">{value || "pending"}</div>
    </div>
  );
}

