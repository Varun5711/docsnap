export type ClaimReasoning = {
  knowns: string[];
  unknowns: string[];
  conflicts: string[];
};
export type Source = {
  id: string;
  claimId: string;
  url: string;
  name: string;
  sourceType: string;
  starRating: number;
  relationship: "supports" | "contradicts" | "unrelated";
  relevance: number;
  capturedAt: string;
};
export type VerdictStatus =
  | "SUPPORTED"
  | "LIKELY_SUPPORTED"
  | "MIXED"
  | "UNVERIFIED"
  | "LIKELY_CONTRADICTED"
  | "CONTRADICTED";
export type Claim = {
  id: string;
  evidenceId: string;
  text: string;
  type: string;
  confidence: number;
  sourceExcerpt: string;
  hash: string;
  status: string;
  subject?: string;
  predicate?: string;
  object?: string;
  claimDate?: string;
  location?: string;
  entities?: string[];
  investigationStatus?: VerdictStatus | "";
  investigationConfidence?: number;
  reasoning?: ClaimReasoning;
  investigatedAt?: string;
  sources?: Source[];
  canonicalClaimId?: string;
  canonicalClaimSlug?: string;
  visibility?: "private" | "unlisted" | "public";
  publishedBy?: string;
  forkedFromClaimId?: string;
  forkedFromOwnerName?: string;
  contributions?: EvidenceContribution[];
};
export type Evidence = {
  id: string;
  url: string;
  domain: string;
  title: string;
  company: string;
  caseId: string;
  userId: string;
  screenshotDataUrl: string;
  scrapedText: string;
  screenshotHash: string;
  scrapedTextHash: string;
  metadataCommitment: string;
  claimsRoot: string;
  evidenceCommitment: string;
  flareTxHash: string;
  teeCertificateHash: string;
  teeSignature: string;
  verificationStatus: string;
  publishedBy?: string;
  anchorSubmitter?: string;
  capturedAt: string;
  createdAt: string;
  claims: Claim[];
};
export type SubmitCalldata = {
  to: string;
  data: string;
  chainId: number;
};
export type DomainTrust = {
  domain: string;
  totalInvestigated: number;
  contradicted: number;
  supported: number;
  falseRatio: number;
  label: "low_trust" | "inconsistent" | "none";
};
export type SearchResult = {
  claims: Claim[];
  items: Evidence[];
};
export type VerifyResult = {
  evidenceId: string;
  verified: boolean;
  expectedCommitment: string;
  actualCommitment: string;
  status: string;
};
export type Investigation = {
  claim: Claim;
  evidence: Evidence;
};
export type Proof = {
  evidenceId: string;
  url: string;
  screenshotHash: string;
  scrapedTextHash: string;
  metadataCommitment: string;
  claimsRoot: string;
  evidenceCommitment: string;
  flareTxHash: string;
  teeCertificateHash: string;
  verificationStatus: string;
  capturedAt: string;
};
import { getToken } from "@/lib/auth";
const API_BASE =
  process.env.NEXT_PUBLIC_DOCSNAP_API_URL ?? "http://localhost:8080";
function personHeaders(): Record<string, string> {
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
export async function fetchScreenshotObjectUrl(
  evidenceId: string,
): Promise<string> {
  const response = await fetch(
    `${API_BASE}/api/evidence/${evidenceId}/screenshot`,
    {
      headers: personHeaders(),
    },
  );
  if (!response.ok) {
    throw new Error("Screenshot fetch failed");
  }
  const blob = await response.blob();
  return URL.createObjectURL(blob);
}
export async function searchClaims(
  params: URLSearchParams,
): Promise<SearchResult> {
  const response = await fetch(`${API_BASE}/api/claims?${params.toString()}`, {
    cache: "no-store",
    headers: personHeaders(),
  });
  if (!response.ok) {
    throw new Error("Search failed");
  }
  return response.json();
}
export async function verifyEvidence(
  payload: Record<string, unknown>,
): Promise<VerifyResult> {
  const response = await fetch(`${API_BASE}/api/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...personHeaders() },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    throw new Error("Verification failed");
  }
  return response.json();
}
export async function getInvestigation(
  claimId: string,
): Promise<Investigation> {
  const response = await fetch(`${API_BASE}/api/investigations/${claimId}`, {
    cache: "no-store",
    headers: personHeaders(),
  });
  if (!response.ok) {
    throw new Error("Investigation not found");
  }
  return response.json();
}
export async function investigateClaim(claimId: string): Promise<Claim> {
  const response = await fetch(
    `${API_BASE}/api/claims/${claimId}/investigate`,
    {
      method: "POST",
      headers: personHeaders(),
    },
  );
  if (!response.ok) {
    throw new Error("Investigation failed");
  }
  return response.json();
}
export async function getProof(
  apiBase: string,
  evidenceId: string,
): Promise<Proof> {
  const response = await fetch(`${apiBase}/api/proof/${evidenceId}`, {
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error("Proof not found");
  }
  return response.json();
}
export function apiBaseUrl(): string {
  return API_BASE;
}
export type CanonicalClaim = {
  id: string;
  slug: string;
  text: string;
  createdAt: string;
  claims: Claim[];
};
export type EvidenceContribution = {
  id: string;
  claimId: string;
  contributorId: string;
  type: "support" | "contradict" | "context" | "correction";
  url: string;
  note: string;
  createdAt: string;
};
export async function discover(): Promise<{
  recent: Claim[];
  trending: Claim[];
}> {
  const response = await fetch(`${API_BASE}/api/discover`, {
    cache: "no-store",
  });
  if (!response.ok) throw new Error("Discover failed");
  return response.json();
}
export async function searchNetwork(query: string): Promise<CanonicalClaim[]> {
  const response = await fetch(
    `${API_BASE}/api/search?q=${encodeURIComponent(query)}`,
    { cache: "no-store" },
  );
  if (!response.ok) throw new Error("Search failed");
  return response.json();
}
export async function similarClaims(query: string): Promise<CanonicalClaim[]> {
  const response = await fetch(
    `${API_BASE}/api/claims/similar?q=${encodeURIComponent(query)}`,
    { cache: "no-store" },
  );
  if (!response.ok) throw new Error("Search failed");
  return response.json();
}
export async function getCanonicalClaim(slug: string): Promise<CanonicalClaim> {
  const response = await fetch(`${API_BASE}/api/claim/${slug}`, {
    cache: "no-store",
    headers: personHeaders(),
  });
  if (!response.ok) throw new Error("Claim not found");
  return response.json();
}
export async function addEvidence(
  claimId: string,
  payload: {
    type: string;
    url: string;
    note: string;
  },
): Promise<Claim> {
  const response = await fetch(`${API_BASE}/api/claims/${claimId}/evidence`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...personHeaders() },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || "Adding evidence failed");
  }
  return response.json();
}
export async function getDomainTrust(domain: string): Promise<DomainTrust> {
  const response = await fetch(
    `${API_BASE}/api/domain/${encodeURIComponent(domain)}/trust`,
    { cache: "no-store" },
  );
  if (!response.ok) throw new Error("Domain trust lookup failed");
  return response.json();
}
export async function prepareAnchor(
  evidenceId: string,
): Promise<SubmitCalldata> {
  const response = await fetch(
    `${API_BASE}/api/evidence/${evidenceId}/anchor/prepare`,
    {
      method: "POST",
      headers: personHeaders(),
    },
  );
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || "Couldn't prepare the anchor transaction");
  }
  return response.json();
}
export async function confirmAnchor(
  evidenceId: string,
  txHash: string,
): Promise<Evidence> {
  const response = await fetch(
    `${API_BASE}/api/evidence/${evidenceId}/anchor/confirm`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json", ...personHeaders() },
      body: JSON.stringify({ txHash }),
    },
  );
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || "Anchoring failed");
  }
  return response.json();
}
export async function forkClaim(claimId: string): Promise<Claim> {
  const response = await fetch(`${API_BASE}/api/claims/${claimId}/fork`, {
    method: "POST",
    headers: personHeaders(),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || "Fork failed");
  }
  return response.json();
}
export async function startInvestigation(claimText: string): Promise<Claim> {
  const captureResponse = await fetch(`${API_BASE}/api/captures`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...personHeaders() },
    body: JSON.stringify({
      url: "docsnap://verify-a-claim",
      title: claimText.slice(0, 80),
      scrapedText: claimText,
      capturedAt: new Date().toISOString(),
    }),
  });
  if (!captureResponse.ok) {
    const body = await captureResponse.json().catch(() => ({}));
    throw new Error(body.error || "Capture failed");
  }
  const evidence: Evidence = await captureResponse.json();
  const claimId = evidence.claims?.[0]?.id;
  if (!claimId) throw new Error("No claim extracted from that text");
  return investigateWithAuth(claimId);
}
async function investigateWithAuth(claimId: string): Promise<Claim> {
  const response = await fetch(
    `${API_BASE}/api/claims/${claimId}/investigate`,
    {
      method: "POST",
      headers: personHeaders(),
    },
  );
  if (!response.ok) throw new Error("Investigation failed");
  return response.json();
}
export async function publishClaim(
  claimId: string,
  visibility: "private" | "unlisted" | "public",
): Promise<Claim> {
  const response = await fetch(`${API_BASE}/api/claims/${claimId}/publish`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...personHeaders() },
    body: JSON.stringify({ visibility }),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || "Publish failed");
  }
  return response.json();
}
