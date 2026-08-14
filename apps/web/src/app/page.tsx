"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import {
  discover,
  similarClaims,
  startInvestigation,
  Claim,
  CanonicalClaim,
} from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { VerdictBadge } from "@/components/verdict-badge";
import { staggerChildrenVariants, itemVariants } from "@/lib/utils/animations";
function ClaimRow({ claim }: { claim: Claim }) {
  return (
    <Link
      href={`/investigations/${claim.id}`}
      className="block border-b border-white/[0.04] px-4 py-4 transition-colors last:border-0 hover:bg-white/[0.03]"
    >
      <p className="text-sm font-medium leading-6 text-foreground/90">
        {claim.text}
      </p>
      <div className="mt-2 flex items-center justify-between">
        <VerdictBadge
          status={claim.investigationStatus || "UNVERIFIED"}
          size="sm"
        />
        <span className="text-xs text-muted-foreground">
          {claim.sources?.length ?? 0} sources
        </span>
      </div>
    </Link>
  );
}
function RollupResult({ rollup }: { rollup: CanonicalClaim }) {
  return (
    <Link
      href={`/claim/${rollup.slug}`}
      className="block rounded-lg border border-white/[0.06] p-4 hover:bg-white/[0.03]"
    >
      <p className="text-sm font-medium text-foreground/90">{rollup.text}</p>
      <p className="mt-2 text-xs text-muted-foreground">
        {rollup.claims.length} investigation
        {rollup.claims.length === 1 ? "" : "s"}
      </p>
    </Link>
  );
}
export default function DiscoverPage() {
  const router = useRouter();
  const { user } = useAuth();
  const [query, setQuery] = useState("");
  const [matches, setMatches] = useState<CanonicalClaim[] | null>(null);
  const [searching, setSearching] = useState(false);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState("");
  const [recent, setRecent] = useState<Claim[]>([]);
  const [trending, setTrending] = useState<Claim[]>([]);
  const [loadingFeed, setLoadingFeed] = useState(true);
  useEffect(() => {
    discover()
      .then((data) => {
        setRecent(data.recent);
        setTrending(data.trending);
      })
      .finally(() => setLoadingFeed(false));
  }, []);
  async function runSearch(e: React.FormEvent) {
    e.preventDefault();
    const text = query.trim();
    if (!text) return;
    setSearching(true);
    setError("");
    try {
      setMatches(await similarClaims(text));
    } catch {
      setError("Search failed — try again.");
    } finally {
      setSearching(false);
    }
  }
  async function startFresh() {
    if (!user) {
      router.push("/login");
      return;
    }
    const text = query.trim();
    if (!text) return;
    setStarting(true);
    setError("");
    try {
      const claim = await startInvestigation(text);
      router.push(`/investigations/${claim.id}`);
    } catch {
      setError("Couldn't start an investigation — try again.");
      setStarting(false);
    }
  }
  return (
    <main className="min-h-screen">
      <div className="mx-auto max-w-3xl px-6 pt-16 pb-10 text-center">
        <h1 className="text-3xl font-semibold tracking-tight text-foreground">
          Don't just trust what you see online.
        </h1>
        <p className="mt-3 text-muted-foreground">
          Investigate claims, discover existing evidence, and verify what was
          actually published.
        </p>
        <form onSubmit={runSearch} className="mx-auto mt-8 flex max-w-xl gap-2">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search a claim, company, website..."
            className="h-12 text-base"
          />
          <Button type="submit" className="h-12 px-6" disabled={searching}>
            {searching ? "Searching…" : "Search"}
          </Button>
        </form>
        {error && <p className="mt-3 text-sm text-red-400">{error}</p>}
      </div>

      {matches !== null && (
        <div className="mx-auto max-w-3xl space-y-3 px-6 pb-10">
          {matches.length > 0 ? (
            <>
              <p className="text-sm font-semibold text-foreground">
                We already know about this
              </p>
              {matches.map((m) => (
                <RollupResult key={m.id} rollup={m} />
              ))}
              <Button
                variant="outline"
                className="w-full"
                onClick={startFresh}
                disabled={starting}
              >
                {starting ? "Starting…" : "Start a new investigation anyway"}
              </Button>
            </>
          ) : (
            <div className="rounded-lg border border-white/[0.06] p-6 text-center">
              <p className="text-sm text-muted-foreground">
                No existing investigations found for this.
              </p>
              <Button className="mt-4" onClick={startFresh} disabled={starting}>
                {starting
                  ? "Investigating…"
                  : user
                    ? "Investigate this claim"
                    : "Log in to investigate"}
              </Button>
            </div>
          )}
        </div>
      )}

      <div className="mx-auto max-w-5xl space-y-8 px-6 pb-16">
        <section>
          <h2 className="mb-3 text-sm font-semibold uppercase tracking-widest text-muted-foreground/60">
            Trending Investigations
          </h2>
          <div className="liquid-glass overflow-hidden">
            {loadingFeed ? (
              <div className="space-y-3 p-4">
                <Skeleton className="h-16 w-full" />
                <Skeleton className="h-16 w-full" />
              </div>
            ) : trending.length === 0 ? (
              <div className="px-4 py-12 text-center text-sm text-muted-foreground">
                Nothing published yet — be the first to publish an
                investigation.
              </div>
            ) : (
              <motion.div
                variants={staggerChildrenVariants}
                initial="hidden"
                animate="visible"
              >
                {trending.map((c) => (
                  <motion.div key={c.id} variants={itemVariants}>
                    <ClaimRow claim={c} />
                  </motion.div>
                ))}
              </motion.div>
            )}
          </div>
        </section>

        <section>
          <h2 className="mb-3 text-sm font-semibold uppercase tracking-widest text-muted-foreground/60">
            Recently Investigated
          </h2>
          <Card>
            {loadingFeed ? (
              <CardContent className="space-y-3 p-4">
                <Skeleton className="h-16 w-full" />
              </CardContent>
            ) : recent.length === 0 ? (
              <CardContent className="p-4 text-center text-sm text-muted-foreground">
                Nothing here yet.
              </CardContent>
            ) : (
              <motion.div
                variants={staggerChildrenVariants}
                initial="hidden"
                animate="visible"
              >
                {recent.map((c) => (
                  <motion.div key={c.id} variants={itemVariants}>
                    <ClaimRow claim={c} />
                  </motion.div>
                ))}
              </motion.div>
            )}
          </Card>
        </section>
      </div>
    </main>
  );
}
