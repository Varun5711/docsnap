"use client";
import Link from "next/link";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
export function Nav() {
  const { user, loading, signOut } = useAuth();
  return (
    <div className="border-b border-white/[0.06]">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
        <div className="flex items-center gap-6">
          <Link
            href="/"
            className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground"
          >
            DocSnap
          </Link>
          <nav className="hidden items-center gap-4 text-sm text-muted-foreground sm:flex">
            <Link href="/" className="hover:text-foreground">
              Discover
            </Link>
            <Link href="/my-work" className="hover:text-foreground">
              My Work
            </Link>
          </nav>
        </div>
        <div className="flex items-center gap-3">
          {loading ? null : user ? (
            <>
              <span className="hidden text-xs text-muted-foreground sm:inline">
                {user.displayName}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => void signOut()}
              >
                Log out
              </Button>
            </>
          ) : (
            <>
              <Link href="/login">
                <Button variant="ghost" size="sm">
                  Log in
                </Button>
              </Link>
              <Link href="/signup">
                <Button size="sm">Sign up</Button>
              </Link>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
