"use client";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowUpRight, Flag } from "lucide-react";
import {
  addEvidence,
  forkClaim,
  getInvestigation,
  investigateClaim,
  publishClaim,
  reportContribution,
  Investigation,
} from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { VerdictBadge } from "@/components/verdict-badge";
const CONTRIBUTION_LABELS: Record<string, string> = {
  support: "Supporting evidence",
  contradict: "Contradicting evidence",
  context: "Context",
  correction: "Correction",
};
const STAGES = [
  "Analyzing claim…",
  "Searching evidence…",
  "Checking primary sources…",
  "Generating proof…",
];
function displayUrl(url: string): string {
  try {
    const parsed = new URL(url);
    return (
      parsed.hostname.replace(/^www\./, "") + parsed.pathname.replace(/\/$/, "")
    );
  } catch {
    return url;
  }
}
export default function InvestigationPage() {
  const params = useParams<{
    id: string;
  }>();
  const router = useRouter();
  const { user } = useAuth();
  const [data, setData] = useState<Investigation | null>(null);
  const [loading, setLoading] = useState(true);
  const [investigating, setInvestigating] = useState(false);
  const [stage, setStage] = useState(0);
  const [error, setError] = useState("");
  const [forking, setForking] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [showVisibilityOptions, setShowVisibilityOptions] = useState(false);
  const [contributing, setContributing] = useState(false);
  const [contributionType, setContributionType] = useState("support");
  const [contributionUrl, setContributionUrl] = useState("");
  const [contributionNote, setContributionNote] = useState("");
  const [reportedIds, setReportedIds] = useState<Set<string>>(new Set());
  async function onReport(contributionId: string) {
    // Optimistic — flip the button immediately so a double-click can't fire
    // two requests, and there's no reason to make the reporter wait on a
    // round-trip for a one-way "noted" action.
    setReportedIds((prev) => new Set(prev).add(contributionId));
    try {
      await reportContribution(contributionId);
    } catch {
      setReportedIds((prev) => {
        const next = new Set(prev);
        next.delete(contributionId);
        return next;
      });
    }
  }
  async function load() {
    setLoading(true);
    setError("");
    try {
      const result = await getInvestigation(params.id);
      setData(result);
    } catch {
      setError("Investigation not found.");
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    void load();
  }, [params.id]);
  async function runInvestigation() {
    setInvestigating(true);
    setStage(0);
    const timers = STAGES.map((_, i) => setTimeout(() => setStage(i), i * 900));
    try {
      await investigateClaim(params.id);
      await load();
    } catch {
      setError("Investigation failed — try again.");
    } finally {
      timers.forEach(clearTimeout);
      setInvestigating(false);
    }
  }
  async function onFork() {
    if (!user) {
      router.push("/login");
      return;
    }
    setForking(true);
    try {
      const forked = await forkClaim(params.id);
      router.push(`/investigations/${forked.id}`);
    } catch {
      setError("Couldn't build on this investigation — try again.");
      setForking(false);
    }
  }
  async function onPublish(visibility: "private" | "unlisted" | "public") {
    setPublishing(true);
    try {
      await publishClaim(params.id, visibility);
      await load();
    } catch {
      setError("Publish failed — try again.");
    } finally {
      setPublishing(false);
    }
  }
  async function onContribute(e: React.FormEvent) {
    e.preventDefault();
    if (!user) {
      router.push("/login");
      return;
    }
    if (!contributionUrl.trim()) {
      setError("Add a source URL — that's the one required field.");
      return;
    }
    setContributing(true);
    try {
      await addEvidence(params.id, {
        type: contributionType,
        url: contributionUrl.trim(),
        note: contributionNote.trim(),
      });
      setContributionUrl("");
      setContributionNote("");
      await load();
    } catch {
      setError("Adding evidence failed — try again.");
    } finally {
      setContributing(false);
    }
  }
  if (loading) {
    return (
      <main className="mx-auto max-w-4xl space-y-5 px-6 py-10">
        <Skeleton className="h-8 w-2/3" />
        <Skeleton className="h-40 w-full" />
        <Skeleton className="h-60 w-full" />
      </main>
    );
  }
  if (error || !data) {
    return (
      <main className="mx-auto max-w-4xl px-6 py-10 text-center text-sm text-muted-foreground">
        {error || "Investigation not found."}
      </main>
    );
  }
  const { claim, evidence } = data;
  const hasVerdict = Boolean(claim.investigationStatus);
  const supports = (claim.sources ?? []).filter(
    (s) => s.relationship === "supports",
  );
  const contradicts = (claim.sources ?? []).filter(
    (s) => s.relationship === "contradicts",
  );
  const unrelated = (claim.sources ?? []).filter(
    (s) => s.relationship === "unrelated",
  );
  const timeline = [
    { label: "Page captured", at: evidence.capturedAt },
    claim.investigatedAt
      ? { label: "Investigation completed", at: claim.investigatedAt }
      : null,
  ].filter(
    (
      t,
    ): t is {
      label: string;
      at: string;
    } => Boolean(t),
  );
  const hasWhy = hasVerdict && Boolean(claim.reasoning);
  const hasEvidenceSources = hasVerdict && (claim.sources?.length ?? 0) > 0;
  return (
    <main className="mx-auto max-w-6xl space-y-5 px-6 py-10">
      <div>
        <div className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground/60">
          DocSnap Investigation
        </div>
        <h1 className="mt-2 text-xl font-semibold leading-relaxed text-foreground">
          "{claim.text}"
        </h1>
        {claim.forkedFromClaimId && (
          <p className="mt-2 text-xs text-muted-foreground">
            <span className="rounded-full border border-white/[0.1] px-2 py-0.5 text-[10px] uppercase tracking-wide">
              Fork
            </span>{" "}
            Built on{" "}
            <Link
              href={`/investigations/${claim.forkedFromClaimId}`}
              className="text-primary hover:underline"
            >
              an earlier investigation
            </Link>
            {claim.forkedFromOwnerName && (
              <> published by {claim.forkedFromOwnerName}</>
            )}
          </p>
        )}
        {claim.canonicalClaimSlug && (
          <p className="mt-1 text-xs text-muted-foreground">
            Part of a shared claim —{" "}
            <Link
              href={`/claim/${claim.canonicalClaimSlug}`}
              className="text-primary hover:underline"
            >
              see all investigations
            </Link>
          </p>
        )}
      </div>

      <div className="grid grid-cols-12 items-stretch gap-5">
        <div className={hasWhy ? "col-span-12 lg:col-span-5" : "col-span-12"}>
          <Card className="flex h-full flex-col">
            <CardContent className="space-y-4 p-6">
              {hasVerdict ? (
                <>
                  <VerdictBadge status={claim.investigationStatus!} />
                  <p className="text-sm text-muted-foreground">
                    Confidence:{" "}
                    {Math.round((claim.investigationConfidence ?? 0) * 100)}%
                  </p>
                  <div className="flex flex-wrap items-center gap-2 border-t border-white/[0.06] pt-4">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={onFork}
                      disabled={forking}
                    >
                      {forking ? "Building…" : "Build on this investigation"}
                    </Button>
                    {(!claim.publishedBy || claim.publishedBy === user?.id) && (
                      <div className="flex items-center gap-2">
                        <Badge
                          variant="outline"
                          className="capitalize text-muted-foreground"
                        >
                          {claim.visibility || "private"}
                        </Badge>
                        {claim.visibility !== "public" &&
                          !showVisibilityOptions && (
                            <Button
                              size="sm"
                              onClick={() => onPublish("public")}
                              disabled={publishing}
                            >
                              {publishing ? "Publishing…" : "Publish publicly"}
                            </Button>
                          )}
                        {!showVisibilityOptions ? (
                          <button
                            onClick={() => setShowVisibilityOptions(true)}
                            className="text-xs text-muted-foreground hover:text-foreground hover:underline"
                          >
                            Change visibility
                          </button>
                        ) : (
                          <div className="flex items-center gap-1.5">
                            {(["private", "unlisted", "public"] as const).map(
                              (v) => (
                                <button
                                  key={v}
                                  onClick={() => {
                                    onPublish(v);
                                    setShowVisibilityOptions(false);
                                  }}
                                  disabled={publishing}
                                  className={`rounded-full border px-2.5 py-1 text-xs capitalize transition-colors ${
                                    (claim.visibility || "private") === v
                                      ? "border-primary bg-primary text-primary-foreground"
                                      : "border-white/[0.1] text-muted-foreground hover:text-foreground"
                                  }`}
                                >
                                  {v}
                                </button>
                              ),
                            )}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </>
              ) : (
                <div className="space-y-3">
                  <p className="text-sm text-muted-foreground">
                    This claim hasn't been investigated yet.
                  </p>
                  {investigating ? (
                    <div className="space-y-1.5">
                      {STAGES.map((label, i) => (
                        <p
                          key={label}
                          className={
                            i <= stage
                              ? "text-sm text-foreground"
                              : "text-sm text-muted-foreground/40"
                          }
                        >
                          {i < stage ? "✓ " : i === stage ? "· " : "  "}
                          {label}
                        </p>
                      ))}
                    </div>
                  ) : (
                    <Button onClick={runInvestigation}>
                      Run investigation
                    </Button>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {hasWhy && (
          <div className="col-span-12 lg:col-span-7">
            <Card className="flex h-full flex-col">
              <CardHeader className="p-4">
                <CardTitle className="text-sm font-semibold">Why?</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4 p-4 pt-0 text-sm">
                {claim.reasoning!.knowns.length > 0 && (
                  <div>
                    <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground/60">
                      What we know
                    </p>
                    <ul className="space-y-1">
                      {claim.reasoning!.knowns.map((k, i) => (
                        <li key={i} className="text-foreground/90">
                          ✓ {k}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                {claim.reasoning!.unknowns.length > 0 && (
                  <div>
                    <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground/60">
                      What we couldn't verify
                    </p>
                    <ul className="space-y-1">
                      {claim.reasoning!.unknowns.map((u, i) => (
                        <li key={i} className="text-muted-foreground">
                          ? {u}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                {claim.reasoning!.conflicts.length > 0 && (
                  <div>
                    <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground/60">
                      What conflicts
                    </p>
                    <ul className="space-y-1">
                      {claim.reasoning!.conflicts.map((c, i) => (
                        <li key={i} className="text-amber-300">
                          ⚠ {c}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        )}
      </div>

      <div className="grid grid-cols-12 items-stretch gap-5">
        {hasEvidenceSources && (
          <div className="col-span-12 lg:col-span-6">
            <Card className="flex h-full flex-col">
              <CardHeader className="p-4">
                <CardTitle className="text-sm font-semibold">
                  Evidence
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4 p-4 pt-0">
                <SourceGroup
                  title="Supports"
                  sources={supports}
                  tone="text-green-400"
                />
                <SourceGroup
                  title="Contradicts"
                  sources={contradicts}
                  tone="text-red-400"
                />
                <SourceGroup
                  title="Unrelated / weak"
                  sources={unrelated}
                  tone="text-muted-foreground"
                />
              </CardContent>
            </Card>
          </div>
        )}

        <div
          className={
            hasEvidenceSources ? "col-span-12 lg:col-span-6" : "col-span-12"
          }
        >
          <Card className="flex h-full flex-col">
            <CardHeader className="flex-row items-center justify-between space-y-0 p-4">
              <div>
                <CardTitle className="text-sm font-semibold">
                  Community Evidence
                </CardTitle>
                <p className="mt-1 text-xs text-muted-foreground/70">
                  Independent of the verdict above — added by users, not
                  verified by DocSnap.
                </p>
              </div>
              <span className="shrink-0 text-xs text-muted-foreground/60">
                {claim.contributions?.length ?? 0} contribution
                {(claim.contributions?.length ?? 0) === 1 ? "" : "s"}
              </span>
            </CardHeader>
            <CardContent className="space-y-4 p-4 pt-0">
              {(claim.contributions?.length ?? 0) === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No community evidence yet — be the first to add some.
                </p>
              ) : (
                <div className="space-y-2">
                  {claim.contributions!.map((c) => {
                    const reportedByMe = reportedIds.has(c.id);
                    return (
                      <div
                        key={c.id}
                        className={`rounded-md border px-3 py-2 text-sm ${
                          c.flagged
                            ? "border-red-500/30 bg-red-500/[0.03]"
                            : "border-white/[0.06]"
                        }`}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-1.5">
                            <Badge variant="outline" className="text-[10px]">
                              {CONTRIBUTION_LABELS[c.type] ?? c.type}
                            </Badge>
                            {c.flagged && (
                              <Badge
                                variant="outline"
                                className="border-red-500/40 text-[10px] text-red-400"
                              >
                                Reported
                              </Badge>
                            )}
                          </div>
                          <span className="text-xs text-muted-foreground">
                            {new Date(c.createdAt).toLocaleDateString()}
                          </span>
                        </div>
                        {c.note && (
                          <p className="mt-1 text-foreground/90">{c.note}</p>
                        )}
                        <div className="mt-1.5 flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
                          <a
                            href={c.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="truncate text-xs text-primary hover:underline"
                          >
                            {displayUrl(c.url)}
                          </a>
                          <div className="flex items-center gap-3 text-xs text-muted-foreground">
                            <span>
                              by {c.contributorName || "a DocSnap user"}
                            </span>
                            <button
                              onClick={() => onReport(c.id)}
                              disabled={reportedByMe}
                              className="flex items-center gap-1 hover:text-red-400 disabled:cursor-default disabled:hover:text-muted-foreground"
                            >
                              <Flag className="h-3 w-3" />
                              {reportedByMe ? "Reported" : "Report"}
                            </button>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}

              <form
                onSubmit={onContribute}
                className="space-y-2 border-t border-white/[0.06] pt-4"
              >
                <div className="flex gap-2">
                  <select
                    value={contributionType}
                    onChange={(e) => setContributionType(e.target.value)}
                    className="h-10 rounded-md border border-input bg-background px-2 text-sm text-foreground outline-none"
                  >
                    <option value="support">Support</option>
                    <option value="contradict">Contradict</option>
                    <option value="context">Context</option>
                    <option value="correction">Correction</option>
                  </select>
                  <Input
                    value={contributionUrl}
                    onChange={(e) => setContributionUrl(e.target.value)}
                    placeholder="Source URL (required)"
                    className="flex-1"
                  />
                </div>
                <Input
                  value={contributionNote}
                  onChange={(e) => setContributionNote(e.target.value)}
                  placeholder="Why does this matter? (optional)"
                />
                <Button type="submit" size="sm" disabled={contributing}>
                  {contributing
                    ? "Adding…"
                    : user
                      ? "Add evidence"
                      : "Log in to add evidence"}
                </Button>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>

      <div className="grid grid-cols-12 items-stretch gap-5">
        <div className="col-span-12 lg:col-span-4">
          <Card className="flex h-full flex-col">
            <CardHeader className="p-4">
              <CardTitle className="text-sm font-semibold">Timeline</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 p-4 pt-0 text-sm">
              {timeline.map((t) => (
                <div
                  key={t.label}
                  className="flex items-center justify-between border-b border-white/[0.04] pb-2 last:border-0"
                >
                  <span className="text-foreground/90">{t.label}</span>
                  <span className="text-xs text-muted-foreground">
                    {new Date(t.at).toLocaleString()}
                  </span>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>

        <div className="col-span-12 lg:col-span-8">
          <Card className="flex h-full flex-col">
            <details>
              <summary className="cursor-pointer list-none p-4 text-sm font-semibold text-foreground">
                Technical Proof
              </summary>
              <CardContent className="space-y-3 p-4 pt-0">
                <HashLine
                  label="Evidence commitment"
                  value={evidence.evidenceCommitment}
                />
                <HashLine label="Claims root" value={evidence.claimsRoot} />
                <HashLine label="Flare tx" value={evidence.flareTxHash} />
                <HashLine
                  label="TEE certificate"
                  value={evidence.teeCertificateHash}
                />
                <a
                  href={`/proof/${evidence.id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-sm text-primary hover:underline"
                >
                  Verify independently <ArrowUpRight className="h-3.5 w-3.5" />
                </a>
              </CardContent>
            </details>
          </Card>
        </div>
      </div>
    </main>
  );
}
function SourceGroup({
  title,
  sources,
  tone,
}: {
  title: string;
  sources: NonNullable<Investigation["claim"]["sources"]>;
  tone: string;
}) {
  if (sources.length === 0) return null;
  return (
    <div>
      <p
        className={`mb-2 text-xs font-semibold uppercase tracking-wide ${tone}`}
      >
        {title} ({sources.length})
      </p>
      <div className="space-y-2">
        {sources.map((s) => (
          <a
            key={s.id}
            href={s.url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center justify-between gap-3 rounded-md border border-white/[0.06] px-3 py-2 text-sm hover:bg-white/[0.03]"
          >
            <span className="flex items-center gap-2 truncate">
              <span className="truncate text-foreground/90">
                {s.name || displayUrl(s.url)}
              </span>
              <Badge variant="outline" className="shrink-0 text-[10px]">
                {s.sourceType}
              </Badge>
            </span>
            <span className="shrink-0 text-xs text-muted-foreground">
              {"★".repeat(s.starRating)}
              {"☆".repeat(5 - s.starRating)}
            </span>
          </a>
        ))}
      </div>
    </div>
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
