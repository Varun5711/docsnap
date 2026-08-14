"use client";
import { useState } from "react";
import Link from "next/link";
import { signup } from "@/lib/auth";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
export default function SignupPage() {
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      await signup(email, password, displayName);
      window.location.href = "/";
    } catch (err) {
      setError(err instanceof Error ? err.message : "Signup failed");
      setLoading(false);
    }
  }
  return (
    <main className="mx-auto max-w-sm px-6 py-20">
      <Card>
        <CardHeader className="p-6">
          <CardTitle className="text-lg font-semibold">Sign up</CardTitle>
        </CardHeader>
        <CardContent className="p-6 pt-0">
          <form onSubmit={onSubmit} className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-muted-foreground/70">
                Display name
              </label>
              <Input
                className="mt-1"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="Alex"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-muted-foreground/70">
                Email
              </label>
              <Input
                className="mt-1"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-muted-foreground/70">
                Password
              </label>
              <Input
                className="mt-1"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                minLength={8}
                required
              />
              <p className="mt-1 text-xs text-muted-foreground/60">
                At least 8 characters.
              </p>
            </div>
            {error && <p className="text-sm text-red-400">{error}</p>}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Creating account…" : "Sign up"}
            </Button>
          </form>
          <p className="mt-4 text-center text-xs text-muted-foreground">
            Already have an account?{" "}
            <Link href="/login" className="text-primary hover:underline">
              Log in
            </Link>
          </p>
        </CardContent>
      </Card>
    </main>
  );
}
