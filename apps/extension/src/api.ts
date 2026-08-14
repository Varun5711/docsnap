export const DEFAULT_API_URL = "http://localhost:8080";
export const WEB_APP_ORIGIN = "http://localhost:3000";
export const DEFAULT_COMPANY_PLACEHOLDER = "ExampleCo";
export type StoredSettings = {
  apiUrl?: string;
  apiKey?: string;
  userId?: string;
};
export function randomHex(length: number): string {
  const bytes = new Uint8Array(Math.ceil(length / 2));
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0"))
    .join("")
    .slice(0, length);
}
export function freshCaseId(): string {
  const date = new Date().toISOString().slice(0, 10).replace(/-/g, "");
  return `CASE-${date}-${randomHex(4)}`;
}
export function guessCompany(url: string): string | null {
  try {
    const host = new URL(url).hostname.replace(/^www\./, "");
    const label = host.split(".")[0];
    return label ? label[0].toUpperCase() + label.slice(1) : null;
  } catch {
    return null;
  }
}
export async function getOrCreateUserId(): Promise<string> {
  const stored = (await chrome.storage.sync.get(["userId"])) as StoredSettings;
  if (stored.userId) return stored.userId;
  const userId = `auditor-${randomHex(8)}@docsnap.local`;
  await chrome.storage.sync.set({ userId });
  return userId;
}
export async function getConnectionSettings(): Promise<{
  apiUrl: string;
  apiKey: string;
}> {
  const stored = (await chrome.storage.sync.get([
    "apiUrl",
    "apiKey",
  ])) as StoredSettings;
  return {
    apiUrl: stored.apiUrl || DEFAULT_API_URL,
    apiKey: stored.apiKey || "",
  };
}
function authHeaders(apiKey: string): Record<string, string> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (apiKey) headers.Authorization = `Bearer ${apiKey}`;
  return headers;
}
export type CapturePayload = {
  url: string;
  title: string;
  company: string;
  caseId: string;
  userId: string;
  screenshotDataUrl?: string;
  scrapedText: string;
};
export async function capture(
  apiUrl: string,
  apiKey: string,
  payload: CapturePayload,
): Promise<any> {
  const response = await fetch(`${apiUrl}/api/captures`, {
    method: "POST",
    headers: authHeaders(apiKey),
    body: JSON.stringify({ ...payload, capturedAt: new Date().toISOString() }),
  });
  if (!response.ok) throw new Error("Capture failed");
  return response.json();
}
export async function investigate(
  apiUrl: string,
  apiKey: string,
  claimId: string,
): Promise<any> {
  const response = await fetch(`${apiUrl}/api/claims/${claimId}/investigate`, {
    method: "POST",
    headers: authHeaders(apiKey),
  });
  if (!response.ok) throw new Error("Investigation failed");
  return response.json();
}
export type RecentInvestigation = {
  id: string;
  text: string;
  status: string;
  at: string;
};
export async function addRecentInvestigation(
  entry: RecentInvestigation,
): Promise<void> {
  const { recentInvestigations = [] } = await chrome.storage.local.get([
    "recentInvestigations",
  ]);
  const next = [
    entry,
    ...(recentInvestigations as RecentInvestigation[]),
  ].slice(0, 5);
  await chrome.storage.local.set({ recentInvestigations: next });
}
export async function getRecentInvestigations(): Promise<
  RecentInvestigation[]
> {
  const { recentInvestigations = [] } = await chrome.storage.local.get([
    "recentInvestigations",
  ]);
  return recentInvestigations as RecentInvestigation[];
}
