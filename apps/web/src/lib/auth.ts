"use client";
import { useCallback, useEffect, useState } from "react";
export type User = {
  id: string;
  email: string;
  displayName: string;
  createdAt: string;
};
const API_BASE =
  process.env.NEXT_PUBLIC_DOCSNAP_API_URL ?? "http://localhost:8080";
const TOKEN_KEY = "docsnap_session_token";
export function getToken(): string {
  if (typeof window === "undefined") return "";
  return localStorage.getItem(TOKEN_KEY) ?? "";
}
function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}
export function authHeaders(): Record<string, string> {
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
export async function signup(
  email: string,
  password: string,
  displayName: string,
): Promise<User> {
  const response = await fetch(`${API_BASE}/api/auth/signup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, displayName }),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || "Signup failed");
  }
  const data = await response.json();
  setToken(data.token);
  return data.user;
}
export async function login(email: string, password: string): Promise<User> {
  const response = await fetch(`${API_BASE}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || "Login failed");
  }
  const data = await response.json();
  setToken(data.token);
  return data.user;
}
export async function logout(): Promise<void> {
  await fetch(`${API_BASE}/api/auth/logout`, {
    method: "POST",
    headers: authHeaders(),
  }).catch(() => {});
  clearToken();
}
export async function fetchMe(): Promise<User | null> {
  const token = getToken();
  if (!token) return null;
  const response = await fetch(`${API_BASE}/api/auth/me`, {
    headers: authHeaders(),
    cache: "no-store",
  });
  if (!response.ok) {
    clearToken();
    return null;
  }
  return response.json();
}
export function useAuth() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      setUser(await fetchMe());
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    void refresh();
  }, [refresh]);
  const signOut = useCallback(async () => {
    await logout();
    setUser(null);
  }, []);
  return { user, loading, refresh, signOut };
}
