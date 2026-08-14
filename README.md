# DocSnap

DocSnap captures web evidence, extracts claims with AI, investigates those claims against live web search with an AI-generated verdict, anchors tamper-evident commitments through Flare Confidential Compute, and publishes the result into a public, forkable investigation network.

## Bounty

Bounty 2 — Confidential Compute Apps

## System Architecture

```mermaid
graph TB
    subgraph Client
        EXT["Chrome Extension<br/>Manifest V3 + Vite"]
        WEB["Next.js Web App<br/>Discover · My Work · Investigations"]
        WALLET["User's Browser Wallet<br/>MetaMask / EIP-1193"]
    end

    subgraph API["Go HTTP API — httpapi.Server"]
        SRV["net/http ServeMux<br/>shared-key + session auth"]
    end

    subgraph Data
        PG[("PostgreSQL<br/>evidence · claims · sources · users<br/>sessions · canonical_claims · contributions")]
        S3[("MinIO / S3<br/>screenshots")]
    end

    GROQEX["Groq<br/>claim extraction"]
    GROQVD["Groq<br/>verdict generation"]
    TAVILY["Tavily<br/>web search"]
    FCC["FCC Extension (Go)<br/>TEE certifier"]
    CHAIN["Coston2<br/>DocSnapAnchor.sol"]

    EXT -->|"POST /api/captures"| SRV
    EXT -->|"POST /api/claims/:id/investigate"| SRV
    WEB -->|"claims, evidence, discover, claim/:slug"| SRV
    WEB -->|"signup, login, publish, fork, evidence"| SRV
    WALLET -->|"eth_sendTransaction<br/>(signed submitEvidence)"| CHAIN

    SRV --> PG
    SRV --> S3
    SRV -->|"vision + text extraction"| GROQEX
    SRV -->|"search(claim text)"| TAVILY
    SRV -->|"classify verdict from results"| GROQVD
    SRV -->|"POST /certify"| FCC
    SRV -->|"shared key: submitEvidence + recordTEECertificate"| CHAIN
    SRV -->|"build calldata for / verify tx from"| WALLET
```

Two independent things get anchored on Coston2, and two independent parties can sign the first one: the **evidence record** (`submitEvidence`, permissionless, records `msg.sender`) and its **TEE certificate** (`recordTEECertificate`, restricted to the TEE reporter key — always server-side, never the user's wallet). See [Bring-Your-Own-Wallet Anchoring](#bring-your-own-wallet-anchoring).

## Data Model

```mermaid
erDiagram
    evidence ||--o{ claims : "evidence_id"
    claims ||--o{ sources : "claim_id"
    claims ||--o{ evidence_contributions : "claim_id"
    claims }o--|| canonical_claims : "canonical_claim_id"
    claims }o--o| claims : "forked_from_claim_id"
    users ||--o{ sessions : "user_id"

    evidence {
        text id PK
        text url
        text domain
        text evidence_commitment
        text flare_tx_hash
        text verification_status
        text published_by
        text anchor_submitter
    }
    claims {
        text id PK
        text evidence_id FK
        text text
        text investigation_status
        numeric investigation_confidence
        text canonical_claim_id FK
        text visibility
        text published_by
        text forked_from_claim_id FK
    }
    sources {
        text id PK
        text claim_id FK
        text url
        int star_rating
        text relationship
        numeric relevance
    }
    users {
        text id PK
        text email
        text password_hash
        text display_name
    }
    sessions {
        text token PK
        text user_id FK
        timestamptz expires_at
    }
    canonical_claims {
        text id PK
        text slug
        text text
    }
    evidence_contributions {
        text id PK
        text claim_id FK
        text contributor_id
        text type
        text url
        text note
    }
```

`claims` and `evidence` both carry `published_by` (the owning user, empty for anonymous/extension captures) and a `visibility` on claims (`private` / `unlisted` / `public`) — see [Access Control](#access-control--visibility). `sources` are AI-retrieved during investigation; `evidence_contributions` are human-submitted and deliberately kept in a separate table (see [Community Evidence](#community-evidence-contributions)).

## Evidence Capture & Certification

```mermaid
sequenceDiagram
    participant U as User
    participant X as Extension / Web App
    participant A as Go API
    participant G as Groq (extraction)
    participant S as MinIO
    participant T as FCC Extension
    participant F as Coston2 Contract

    U->>X: Capture this page
    X->>X: Screenshot + scrape page text
    X->>A: POST /api/captures
    A->>G: Extract claims (vision + text)
    G-->>A: Structured claims JSON
    A->>A: Hash screenshot, text, metadata, claims
    A->>S: Store screenshot
    A->>T: POST /certify (commitments)
    T-->>A: TEE certificate + signature (off-chain)
    alt anonymous or extension capture
        A->>F: submitEvidence(...) [shared signer key]
        A->>F: recordTEECertificate(...) [TEE reporter key]
        F-->>A: transaction hash
    else logged-in web user
        A->>A: status = pending_wallet_anchor<br/>(on-chain steps deferred)
    end
    A->>A: Save evidence + claims to Postgres
    A-->>X: Evidence, claims, tx hash or pending status
```

The two on-chain calls only happen immediately for anonymous/extension captures, which have no wallet to ask. A logged-in web user's capture is saved and fully usable right away — investigation, proof-of-capture, everything — it just isn't on Coston2 yet.

## Claim Investigation (Verdict Generation)

```mermaid
sequenceDiagram
    participant U as User
    participant A as Go API
    participant TV as Tavily
    participant GV as Groq (verdict)
    participant P as Postgres

    U->>A: POST /api/claims/:id/investigate
    A->>TV: search(claim text)
    TV-->>A: up to 5 results, 800 chars each
    alt no results
        A-->>P: status = UNVERIFIED
    else results found
        A->>GV: classify verdict<br/>(results wrapped in untrusted-content delimiters)
        GV-->>A: status, confidence, knowns/unknowns/conflicts, per-source ratings
        A->>P: SaveInvestigation
    end
    A-->>U: Claim with verdict + reasoning + sources
```

Search results are untrusted, model-facing input — both the extraction and verdict prompts wrap them in explicit `<untrusted-content>` delimiters so a page that says "ignore previous instructions" in its own text can't steer the classifier.

```mermaid
stateDiagram-v2
    [*] --> UNVERIFIED: no search results
    [*] --> classified: results found
    classified --> SUPPORTED
    classified --> LIKELY_SUPPORTED
    classified --> MIXED
    classified --> LIKELY_CONTRADICTED
    classified --> CONTRADICTED
```

## Evidence Commitment

```mermaid
graph LR
    SS["Screenshot data URL"] -->|sha256| SH["screenshotHash"]
    TXT["Scraped text"] -->|sha256| TH["scrapedTextHash"]
    META["URL + company + case + user + timestamp"] -->|sha256| MC["metadataCommitment"]
    CLAIMS["Claims"] -->|"sha256 each, sorted, joined"| CR["claimsRoot"]
    SH --> EC["evidenceCommitment"]
    TH --> EC
    MC --> EC
    CR --> EC
```

Changing a single byte of the screenshot, text, or any claim changes its hash, which changes `claimsRoot` or `screenshotHash`/`scrapedTextHash`, which changes `evidenceCommitment` — the value anchored on Coston2 and checked on verify.

## Bring-Your-Own-Wallet Anchoring

`submitEvidence()` on `DocSnapAnchor.sol` is permissionless and records `msg.sender` — so no contract change was needed to let a logged-in user sign it themselves instead of the shared server key. The API only ever builds the calldata; it never sees the user's private key.

```mermaid
sequenceDiagram
    participant U as User's Wallet
    participant W as Web App
    participant A as Go API
    participant C as Coston2 Contract
    participant T as FCC Extension

    W->>A: POST /api/evidence/:id/anchor/prepare
    A->>A: Pack submitEvidence calldata<br/>from stored commitments
    A-->>W: {to, data, chainId}
    W->>U: eth_requestAccounts, switch to Coston2
    U->>C: eth_sendTransaction (signed by user)
    C-->>U: txHash
    W->>A: POST /api/evidence/:id/anchor/confirm {txHash}
    A->>C: Fetch receipt + tx (poll up to 30s)
    A->>A: Verify target + calldata match expected
    A->>T: recordTEECertificate [TEE reporter key, still server-side]
    A->>A: Save txHash, status=certified,<br/>anchor_submitter=user's address
    A-->>W: Updated evidence
```

If `confirm` times out on a slow block but the tx lands moments later, re-submitting the same `txHash` is safe — it just re-verifies — so the UI offers a "Recheck status" retry instead of forcing a brand new signature.

## Verification

```mermaid
sequenceDiagram
    participant D as Dashboard
    participant A as Go API
    participant P as Postgres
    participant S as MinIO

    D->>A: POST /api/verify {evidenceId}
    A->>P: Load stored evidence
    A->>S: Load screenshot if needed
    A->>A: Recompute evidenceCommitment
    alt commitment matches stored value
        A-->>D: verified = true, status = verified
    else commitment differs
        A-->>D: verified = false, status = tampered
    end
```

## Accounts & Dual Auth

One `auth()` middleware accepts either credential — the extension's machine key never needs to become a person, and a person never needs an API key:

```mermaid
graph LR
    REQ["Incoming request"] --> CHECK{"Authorization header"}
    CHECK -->|"matches DOCSNAP_API_KEY"| FULL["fullAccess = true<br/>(extension, shared key)"]
    CHECK -->|"valid session token"| USER["viewerID = user.id<br/>(logged-in web user)"]
    CHECK -->|"public route prefix"| PASS["no credential required<br/>(/proof, /discover, /claim/:slug, ...)"]
    CHECK -->|"none of the above"| REJECT["401"]
```

`resolveViewer()` returns `(fullAccess, viewerID)` from that same check — used everywhere a person's identity matters beyond the blanket gate: per-row `canView` for private claims, ownership checks on wallet-anchoring and publish, and scoping `/my-work` search results to the logged-in user's own captures.

## Discovery Network: Publish, Fork & Canonical Claims

A `claims` row already *is* one investigation (one extracted claim, one verdict). `canonical_claims` is a thin grouping row so independent investigations of the same real-world claim — different captures, different researchers, forks — share one public URL instead of a heavier parallel entity.

```mermaid
flowchart TD
    CAP["Capture → Claim<br/>(private, or unlisted if anonymous)"] --> PUB["POST /claims/:id/publish<br/>{private | unlisted | public}"]
    PUB --> SIM{"Similar canonical_claim<br/>already exists?<br/>(trigram similarity)"}
    SIM -->|yes| LINK["Link to existing canonical_claims row"]
    SIM -->|no| MINT["Mint new canonical_claims row + slug"]
    LINK --> PAGE["/claim/:slug — all linked investigations"]
    MINT --> PAGE
    PAGE --> DISCOVER["/api/discover — recent + trending public claims"]

    ORIG["Existing investigation"] -->|"Build on this investigation"| FORK["POST /claims/:id/fork"]
    FORK --> COPY["Copy text + sources into a new claim<br/>forked_from_claim_id set, owner = forker,<br/>visibility = private"]
    COPY --> INDEP["Independent afterward —<br/>evidence added to the fork doesn't touch the parent"]
```

## Community Evidence Contributions

`evidence_contributions` is separate from the AI-retrieved `sources` table — a person, not the verdict pipeline, attaching a URL and a note. Types are `support` / `contradict` / `context` / `correction`; there is deliberately no truth-voting mechanic.

```mermaid
graph LR
    U["Logged-in user"] -->|"POST /claims/:id/evidence"| A["Go API"]
    A --> V{"type ∈ {support, contradict,<br/>context, correction}?"}
    V -->|yes| SAVE["Save to evidence_contributions"]
    V -->|no| REJECT["400"]
```

## Domain Trust Signal

A domain isn't "false" — its claims keep getting contradicted. Rolled up from data that already exists; no new table.

```mermaid
flowchart LR
    Q["claims JOIN evidence<br/>WHERE domain = X"] --> COUNT["total investigated,<br/>contradicted, supported"]
    COUNT --> GUARD{"total ≥ 3?"}
    GUARD -->|no| NONE["label = none<br/>(not enough signal)"]
    GUARD -->|yes| RATIO{"contradicted / total > 0.5?"}
    RATIO -->|yes| LOW["label = low_trust"]
    RATIO -->|no| MIXED{"supported > 0 AND<br/>contradicted > 0?"}
    MIXED -->|yes| INCON["label = inconsistent"]
    MIXED -->|no| NONE2["label = none"]
```

Exposed at `GET /api/domain/:domain/trust`, rendered as a badge wherever a domain already appears — with a per-domain request cache on the frontend so one claims list doesn't fire a duplicate fetch per row.

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
| `POST /api/auth/signup` `login` `logout` `GET /api/auth/me` | — | Account + session lifecycle |
| `GET /api/discover` `GET /api/search` `GET /api/claims/similar` | public | Network read surface |
| `GET /api/claim/:slug` | public (per-row `canView`) | Canonical claim rollup |
| `POST /api/claims/:id/evidence` `fork` `publish` | session required | Contribute, fork, or publish an investigation |
| `POST /api/evidence/:id/anchor/prepare` `confirm` | session required, owner only | Bring-your-own-wallet anchoring |
| `GET /api/domain/:domain/trust` | public | Domain trust rollup |

## Stack

- Chrome extension: Manifest V3, TypeScript, Vite
- Web app: Next.js, TypeScript, Tailwind CSS
- Backend: Go HTTP API (stdlib `net/http`, `pgx`)
- Database: PostgreSQL with full-text search and trigram indexes
- Storage: MinIO / S3-compatible object storage
- AI: Groq for claim extraction and verdict classification (two distinct models — a reasoning-capable multimodal model for extraction, a plain instruct model for verdict classification), with a local rule-based fallback extractor
- Search: Tavily for live web search backing each investigation
- Blockchain: Solidity contract on Flare Coston2, callable by a shared server key or a user's own wallet
- Confidential compute: Go FCC extension (standalone TEE certifier)
- Wallet: raw EIP-1193 (`window.ethereum`) — no wagmi/viem/ethers dependency

## Local First Run

```bash
docker compose -f infra/docker-compose.yml up -d postgres minio
bun install
bun run dev:api        # Go API on :8080, hot reload via air
bun run dev:web        # Web app on :3000
bun run build:extension && bun run dev:extension
cd fcc/docsnap/go && go run ./cmd/extension  # optional standalone TEE, :8787
```

The API applies its Postgres schema automatically on startup and creates the MinIO bucket if it doesn't exist.

## Required Credentials

Create `.env` from `.env.example`.

- `GROQ_API_KEY`: create an API key in the Groq console at `https://console.groq.com/keys`.
- `GROQ_MODEL`: defaults to `qwen/qwen3.6-27b`, a reasoning-capable multimodal model used for screenshot + text claim extraction.
- `GROQ_VERDICT_MODEL`: defaults to `llama-3.3-70b-versatile` — a plain instruct model, deliberately not the reasoning model above; verdict classification is a tight-latency, tight-token-budget task where a reasoning model's hidden "thinking" tokens blow the free-tier rate limit for no accuracy benefit.
- `TAVILY_API_KEY`: create an API key at `https://tavily.com` — powers the live web search behind every investigation.
- `DOCSNAP_API_KEY`: shared secret for the extension's machine calls (Bearer token). Real people authenticate with session tokens instead — see [Accounts & Dual Auth](#accounts--dual-auth).
- `DOCSNAP_FLARE_PRIVATE_KEY`: private key for a funded Coston2 wallet (the shared/fallback submitter).
- `DOCSNAP_TEE_REPORTER_PRIVATE_KEY`: private key for the contract's TEE reporter account, funded separately — it sends the `recordTEECertificate` transaction, always server-side regardless of who submitted the evidence.
- `DOCSNAP_FLARE_RPC_URL`: defaults to `https://coston2-api.flare.network/ext/C/rpc`.
- `DOCSNAP_CONTRACT_ADDRESS`: deployed `DocSnapAnchor` contract address on Coston2.
- `DOCSNAP_TEE_URL`: FCC extension URL, defaults to `http://localhost:8787`. If empty, the API falls back to an in-process certificate simulator.
- `DOCSNAP_S3_*`: MinIO/S3 endpoint, credentials, and bucket. Defaults match `infra/docker-compose.yml`.
- `DATABASE_URL`: Postgres connection string.

## Contracts

```bash
cd contracts
forge install foundry-rs/forge-std --no-git  # first time only
forge test
DOCSNAP_TEE_REPORTER=0xYourTEEReporterAddress forge script script/DeployDocSnapAnchor.s.sol --rpc-url "$DOCSNAP_FLARE_RPC_URL" --private-key "$DOCSNAP_FLARE_PRIVATE_KEY" --broadcast
```

## FCC Extension

```bash
cd fcc/docsnap/go
go test ./...
go run ./cmd/extension
```

## Before Coston2 Deployment

- Docker daemon running (`docker compose -f infra/docker-compose.yml up -d postgres minio`)
- Foundry installed (`curl -L https://foundry.paradigm.xyz | bash && foundryup`)
- Two funded Coston2 wallets — deployer/submitter and TEE reporter — via `cast wallet new` and the [Coston2 faucet](https://faucet.flare.network)
- `DOCSNAP_FLARE_RPC_URL`, `DOCSNAP_FLARE_PRIVATE_KEY`, `DOCSNAP_TEE_REPORTER_PRIVATE_KEY` set in `.env`
