export type Claim = {
  id: string;
  evidenceId: string;
  text: string;
  type: string;
  confidence: number;
  sourceExcerpt: string;
  hash: string;
  status: string;
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
  capturedAt: string;
  createdAt: string;
  claims: Claim[];
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

const API_BASE = process.env.NEXT_PUBLIC_DOCSNAP_API_URL ?? "http://localhost:8080";

export function screenshotUrl(evidenceId: string): string {
  return `${API_BASE}/api/evidence/${evidenceId}/screenshot`;
}

export async function searchClaims(params: URLSearchParams): Promise<SearchResult> {
  const response = await fetch(`${API_BASE}/api/claims?${params.toString()}`, { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Search failed");
  }
  return response.json();
}

export async function createCapture(payload: Record<string, unknown>): Promise<Evidence> {
  const response = await fetch(`${API_BASE}/api/captures`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    throw new Error("Capture failed");
  }
  return response.json();
}

export async function verifyEvidence(payload: Record<string, unknown>): Promise<VerifyResult> {
  const response = await fetch(`${API_BASE}/api/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    throw new Error("Verification failed");
  }
  return response.json();
}
