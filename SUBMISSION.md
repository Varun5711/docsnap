# DocSnap

DocSnap captures web evidence, extracts claims with AI, investigates those claims against live web search with an AI-generated verdict, anchors tamper-evident commitments through Flare Confidential Compute, and publishes the result into a public, forkable investigation network.

## Submission Requirements

**Project name:** DocSnap

**Selected bounty:** Bounty 2 — Confidential Compute Apps

**Short product description:** DocSnap captures a webpage as tamper-evident evidence — screenshot, scraped text, and AI-extracted claims are hashed, certified inside a confidential-compute TEE, and anchored on Flare Coston2. Anyone with the proof link can independently re-derive the hash and confirm nothing changed since capture, without logging in or trusting DocSnap's own UI. An AI investigation layer checks each claim against live web search and returns a six-state verdict with cited sources, and a public network lets people publish, fork, and build on each other's investigations — with users able to sign the on-chain anchor transaction from their own wallet instead of a shared server key.

**Target user:** Anyone who needs to prove a webpage said something before it was edited or taken down — journalists, researchers, consumer-protection/scam-tracking communities, and compliance teams archiving marketing claims.

**Demo video:** [https://youtu.be/Y3q1AVlxikc](https://youtu.be/Y3q1AVlxikc)

**GitHub repo:** [https://github.com/Varun5711/docsnap](https://github.com/Varun5711/docsnap)

**How the project uses Flare:**
- Every capture is hashed into an `evidenceCommitment` and anchored on Coston2 via `DocSnapAnchor.sol` — `submitEvidence()` records the commitment and the submitter's address on-chain.
- `submitEvidence()` is permissionless, so a logged-in user can sign and send that transaction with their own wallet (MetaMask, EIP-1193) instead of DocSnap's shared server key — the server only builds the calldata, never touches the user's key. The server key remains as a fallback for anonymous/extension captures that have no wallet to ask.
- A second, separately-gated contract call — `recordTEECertificate()`, restricted to the TEE reporter address — anchors the confidential-compute certificate once it's been issued, independent of who submitted the evidence.

**What was newly built, ported, integrated, or improved during the program:**
- Claim investigation layer: live web search (Tavily) + AI verdict classification (Groq) producing a six-state verdict with cited, star-rated sources — entirely new.
- Public discovery network: real accounts, publish/private/unlisted/public visibility, canonical-claim deduplication (both at investigate-time and publish-time), fork ("build on this investigation"), and community evidence contributions with attribution and reporting/moderation — entirely new.
- Bring-your-own-wallet anchoring: users can sign the on-chain evidence submission themselves instead of going through the shared server key — entirely new; the contract required no changes since `submitEvidence()` was already permissionless.
- Domain trust signal: aggregates investigation verdicts by domain to flag domains with a pattern of contradicted claims — entirely new.
- Hardened the original evidence-capture pipeline (screenshot + scrape + AI claim extraction + TEE certification + Coston2 anchoring) that predates this submission window — fixed several real correctness bugs found during end-to-end testing (see commit history), including a case where forking a claim silently broke tamper verification on the original evidence.

**Smart contract address:** `DocSnapAnchor.sol` — Coston2 — [`0xdaC770BAEcC31149A1173Bc616eF90E6Acb1aC3c`](https://coston2-explorer.flare.network/address/0xdaC770BAEcC31149A1173Bc616eF90E6Acb1aC3c)

**Roadmap / next steps:**
- Login flow for the browser extension, so extension captures attribute to a real account automatically instead of requiring a manual fork/publish to claim ownership.
- Admin moderation queue for reported community evidence, beyond the current report-and-flag signal.
- Flare Mainnet deployment once bring-your-own-wallet anchoring has more real-world mileage on Coston2.

**Deployed on:** Coston2 (testnet).

**User acquisition / distribution / testing:** Tested internally end-to-end across two real accounts on Coston2 — real wallet-signed anchor transactions, cross-account fork and access-control checks, and the full community-evidence moderation flow, all confirmed against the live API and database, not just in code review. No external users yet; this is pre-launch.

**Early traction / community interest:** None yet to report honestly — pre-launch, no pilot users or partner conversations so far.

**X / Twitter:** [@docsnappp](https://x.com/docsnappp)

## System Architecture

```
CLIENT
  Chrome Extension (Manifest V3 + Vite)
  Next.js Web App — Discover · My Work · Investigations
  User's Browser Wallet (MetaMask / EIP-1193)
        │
        ├─ POST /api/captures ───────────────────┐
        ├─ POST /api/claims/:id/investigate ──────┤
        ├─ claims, evidence, discover, claim/:slug┤
        ├─ signup, login, publish, fork, evidence ┤
        │                                         v
        │                          Go HTTP API — net/http ServeMux
        │                          shared-key + session auth
        │                                         │
        │            ┌──────────┬─────────────────┼──────────────┬────────────────┐
        │            v          v                 v              v                v
        │       PostgreSQL   MinIO/S3         Groq (extract)  Groq (verdict)   Tavily
        │       evidence,    screenshots      vision + text   classify verdict search(claim
        │       claims,                       extraction      from results     text)
        │       sources,                                          │
        │       users,                                            │
        │       sessions,                                         v
        │       canonical_claims,                        POST /certify
        │       evidence_contributions                             │
        │                                                          v
        │                                                 FCC Extension (Go)
        │                                                 TEE certifier
        │                                                          │
        │                                          shared key: submitEvidence +
        │                                          recordTEECertificate
        │                                                          v
        └─ eth_sendTransaction (signed submitEvidence) ──> Coston2 — DocSnapAnchor.sol
                                                             build calldata for /
                                                             verify tx from ↑ (BYOW path)
```

Two independent things get anchored on Coston2, and two independent parties can sign the first one: the **evidence record** (`submitEvidence`, permissionless, records `msg.sender`) and its **TEE certificate** (`recordTEECertificate`, restricted to the TEE reporter key — always server-side, never the user's wallet).

## Data Model

**Relationships:**
- `evidence` 1—* `claims` (via `evidence_id`)
- `claims` 1—* `sources` (via `claim_id`)
- `claims` 1—* `evidence_contributions` (via `claim_id`)
- `claims` *—1 `canonical_claims` (via `canonical_claim_id`)
- `claims` *—0..1 `claims` (self-referential fork, via `forked_from_claim_id`)
- `users` 1—* `sessions` (via `user_id`)

**Key fields:**

| Table | Notable columns |
|---|---|
| `evidence` | `id`, `url`, `domain`, `evidence_commitment`, `flare_tx_hash`, `verification_status`, `published_by`, `anchor_submitter` |
| `claims` | `id`, `evidence_id`, `text`, `investigation_status`, `investigation_confidence`, `canonical_claim_id`, `visibility`, `published_by`, `forked_from_claim_id` |
| `sources` | `id`, `claim_id`, `url`, `star_rating`, `relationship`, `relevance` |
| `users` | `id`, `email`, `password_hash`, `display_name` |
| `sessions` | `token`, `user_id`, `expires_at` |
| `canonical_claims` | `id`, `slug`, `text` |
| `evidence_contributions` | `id`, `claim_id`, `contributor_id`, `type`, `url`, `note` |

`claims` and `evidence` both carry `published_by` (empty for anonymous/extension captures) and `claims` carries `visibility` (`private`/`unlisted`/`public`). `sources` are AI-retrieved during investigation; `evidence_contributions` are human-submitted and deliberately kept in a separate table so AI signal and human signal never get conflated.

## Flow: Evidence Capture & Certification

```
User            -> Extension/Web App : Capture this page
Extension       -> Extension         : Screenshot + scrape page text
Extension       -> Go API            : POST /api/captures
Go API          -> Groq (extraction) : Extract claims (vision + text)
Groq            -> Go API            : Structured claims JSON
Go API          -> Go API            : Hash screenshot, text, metadata, claims
Go API          -> MinIO             : Store screenshot
Go API          -> FCC Extension     : POST /certify (commitments)
FCC Extension   -> Go API            : TEE certificate + signature (off-chain)

  if anonymous or extension capture:
    Go API      -> Coston2           : submitEvidence(...)        [shared signer key]
    Go API      -> Coston2           : recordTEECertificate(...)  [TEE reporter key]
    Coston2     -> Go API            : transaction hash
  else logged-in web user:
    Go API      -> Go API            : status = pending_wallet_anchor
                                        (on-chain steps deferred)

Go API          -> Go API            : Save evidence + claims to Postgres
Go API          -> Extension/Web App : Evidence, claims, tx hash or pending status
```

The two on-chain calls only happen immediately for anonymous/extension captures, which have no wallet to ask. A logged-in web user's capture is saved and fully usable right away — investigation, proof-of-capture, everything — it just isn't on Coston2 yet.

## Flow: Claim Investigation (Verdict Generation)

```
User    -> Go API   : POST /api/claims/:id/investigate
Go API  -> Tavily    : search(claim text)
Tavily  -> Go API    : up to 5 results, 800 chars each

  if no results:
    Go API -> Postgres : status = UNVERIFIED
  else:
    Go API -> Groq (verdict) : classify verdict
                                (results wrapped in <untrusted-content> delimiters)
    Groq   -> Go API         : status, confidence, knowns/unknowns/conflicts,
                                per-source ratings
    Go API -> Postgres       : SaveInvestigation

Go API  -> User      : Claim with verdict + reasoning + sources
```

Search results are untrusted, model-facing input — both the extraction and verdict prompts wrap them in explicit `<untrusted-content>` delimiters so a page that says "ignore previous instructions" in its own text can't steer the classifier.

**Verdict states:**
```
[start] --no search results--> UNVERIFIED
[start] --results found--> classified
classified --> SUPPORTED
classified --> LIKELY_SUPPORTED
classified --> MIXED
classified --> LIKELY_CONTRADICTED
classified --> CONTRADICTED
```

## Evidence Commitment

```
Screenshot data URL ──sha256──> screenshotHash ─────┐
Scraped text ──sha256──> scrapedTextHash ────────────┤
URL+company+case+user+timestamp ──sha256──> metadataCommitment ─┼──> evidenceCommitment
Claims ──sha256 each, sorted, joined──> claimsRoot ──────────────┘
```

Changing a single byte of the screenshot, text, or any claim changes its hash, which changes `claimsRoot` or `screenshotHash`/`scrapedTextHash`, which changes `evidenceCommitment` — the value anchored on Coston2 and checked on verify.

## Flow: Bring-Your-Own-Wallet Anchoring

`submitEvidence()` on `DocSnapAnchor.sol` is permissionless and records `msg.sender` — no contract change was needed to let a logged-in user sign it themselves instead of the shared server key. The API only ever builds the calldata; it never sees the user's private key.

```
Web App      -> Go API       : POST /api/evidence/:id/anchor/prepare
Go API       -> Go API       : Pack submitEvidence calldata from stored commitments
Go API       -> Web App      : {to, data, chainId}
Web App      -> User's Wallet: eth_requestAccounts, switch to Coston2
User's Wallet-> Coston2      : eth_sendTransaction (signed by user)
Coston2      -> User's Wallet: txHash
Web App      -> Go API       : POST /api/evidence/:id/anchor/confirm {txHash}
Go API       -> Coston2      : Fetch receipt + tx (poll up to 30s)
Go API       -> Go API       : Verify target + calldata match expected
Go API       -> FCC Extension: recordTEECertificate [TEE reporter key, still server-side]
Go API       -> Go API       : Save txHash, status=certified, anchor_submitter=user's address
Go API       -> Web App      : Updated evidence
```

If `confirm` times out on a slow block but the tx lands moments later, re-submitting the same `txHash` is safe — it just re-verifies — so the UI offers a "Recheck status" retry instead of forcing a brand-new signature.

## Flow: Verification

```
Dashboard -> Go API   : POST /api/verify {evidenceId}
Go API    -> Postgres : Load stored evidence
Go API    -> MinIO    : Load screenshot if needed
Go API    -> Go API   : Recompute evidenceCommitment

  if commitment matches stored value:
    Go API -> Dashboard : verified = true, status = verified
  else:
    Go API -> Dashboard : verified = false, status = tampered
```

## Accounts & Dual Auth

One `auth()` middleware accepts either credential — the extension's machine key never needs to become a person, and a person never needs an API key:

```
Incoming request
  └─ Authorization header:
       ├─ matches DOCSNAP_API_KEY  → fullAccess = true (extension, shared key)
       ├─ valid session token      → viewerID = user.id (logged-in web user)
       ├─ public route prefix      → no credential required
       │                             (/proof, /discover, /claim/:slug, ...)
       └─ none of the above        → 401
```

`resolveViewer()` returns `(fullAccess, viewerID)` from that same check — used for per-row `canView` on private claims, ownership checks on wallet-anchoring and publish, and scoping `/my-work` to the logged-in user's own captures.

## Discovery Network: Publish, Fork & Canonical Claims

A `claims` row already *is* one investigation (one extracted claim, one verdict). `canonical_claims` is a thin grouping row so independent investigations of the same real-world claim share one public URL instead of a heavier parallel entity.

```
Capture → Claim (private, or unlisted if anonymous)
  → POST /claims/:id/publish {private | unlisted | public}
    → Similar canonical_claim already exists? (trigram similarity)
        yes → Link to existing canonical_claims row
        no  → Mint new canonical_claims row + slug
    → /claim/:slug — all linked investigations
    → /api/discover — recent + trending public claims

Existing investigation --"Build on this investigation"--> POST /claims/:id/fork
  → Copy text + sources into a new claim
    (forked_from_claim_id set, owner = forker, visibility = private)
  → Independent afterward — evidence added to the fork doesn't touch the parent
```

## Community Evidence Contributions

`evidence_contributions` is separate from the AI-retrieved `sources` table — a person, not the verdict pipeline, attaching a URL and a note. There is deliberately no truth-voting mechanic.

```
Logged-in user --POST /claims/:id/evidence--> Go API
  → type ∈ {support, contradict, context, correction}?
      yes → Save to evidence_contributions
      no  → 400
```

## Domain Trust Signal

A domain isn't "false" — its claims keep getting contradicted. Rolled up from data that already exists; no new table.

```
claims JOIN evidence WHERE domain = X
  → total investigated, contradicted, supported
  → total >= 3?
      no  → label = none (not enough signal)
      yes → contradicted / total > 0.5?
              yes → label = low_trust
              no  → supported > 0 AND contradicted > 0?
                      yes → label = inconsistent
                      no  → label = none
```

Exposed at `GET /api/domain/:domain/trust`, rendered as a badge wherever a domain appears — with a per-domain request cache on the frontend so one claims list doesn't fire a duplicate fetch per row.

## Access Control & Visibility

| Visibility | Who can view |
|---|---|
| `private` | The owner only (single account — no team/group sharing) |
| `unlisted` | Anyone with the direct link |
| `public` | Unlisted, plus listed on Discover and search |

Anonymous/extension captures default to `unlisted`, not `private` — an owner-only claim with no owner would be permanently unviewable by anyone, including the person who just captured it.

## API Surface

| Route | Auth | Purpose |
|---|---|---|
| `POST /api/captures` | shared key or session | Capture a page: extract, hash, certify, anchor (or defer to wallet) |
| `GET /api/claims` | shared key or session | Search — scoped to the caller's own captures when logged in |
| `POST /api/claims/:id/investigate` | shared key or session | Run Tavily + Groq verdict generation |
| `GET /api/investigations/:id` | public (per-row `canView`) | One claim's full verdict + sources |
| `GET /api/proof/:id` | public | Evidence commitments + on-chain status, no login |
| `POST /api/auth/signup` / `login` / `logout`, `GET /api/auth/me` | — | Account + session lifecycle |
| `GET /api/discover`, `GET /api/search`, `GET /api/claims/similar` | public | Network read surface |
| `GET /api/claim/:slug` | public (per-row `canView`) | Canonical claim rollup |
| `POST /api/claims/:id/evidence` / `fork` / `publish` | session required | Contribute, fork, or publish an investigation |
| `POST /api/evidence/:id/anchor/prepare` / `confirm` | session required, owner only | Bring-your-own-wallet anchoring |
| `GET /api/domain/:domain/trust` | public | Domain trust rollup |

## Stack

- Chrome extension: Manifest V3, TypeScript, Vite
- Web app: Next.js, TypeScript, Tailwind CSS
- Backend: Go HTTP API (stdlib `net/http`, `pgx`)
- Database: PostgreSQL with full-text search and trigram indexes
- Storage: MinIO / S3-compatible object storage
- AI: Groq — a reasoning-capable multimodal model for claim extraction, a plain instruct model for verdict classification, with a local rule-based fallback extractor
- Search: Tavily for live web search backing each investigation
- Blockchain: Solidity contract on Flare Coston2, callable by a shared server key or a user's own wallet
- Confidential compute: Go FCC extension (standalone TEE certifier)
- Wallet: raw EIP-1193 (`window.ethereum`) — no wagmi/viem/ethers dependency
