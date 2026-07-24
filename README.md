# DocSnap

DocSnap captures web evidence, extracts claims with AI, makes those claims searchable, and anchors tamper-evident commitments through Flare Confidential Compute.

## Bounty

Bounty 2 — Confidential Compute Apps

## Demo Flow

```text
Open webpage
-> Click Capture with DocSnap
-> Extension captures screenshot and scrapes page text
-> Go API hashes screenshot, text, URL, timestamp, and metadata
-> AI extracts structured claims
-> Claims are indexed for search
-> App submits commitments to Flare
-> FCC TEE validates the evidence package and signs a certificate
-> Dashboard shows claims, evidence status, TEE certificate, and Flare tx hash
-> Original evidence verifies
-> Modified screenshot, text, or claim fails verification
```

## Stack

- Chrome extension: Manifest V3, TypeScript, Vite
- Dashboard: Next.js, TypeScript, Tailwind CSS
- Backend: Go HTTP API
- Database target: PostgreSQL with full-text search and trigram indexes
- Storage target: S3-compatible object storage
- AI: Groq vision/text claim extraction
- Blockchain: Solidity contract on Flare Coston2
- Confidential compute: Go FCC extension

## Local First Run

```bash
docker compose -f infra/docker-compose.yml up -d postgres
cd services/api && go run ./cmd/api
bun install
bun run dev:web
bun run dev:extension
```

The API uses PostgreSQL for evidence and claim search and applies its schema automatically on startup.

## Required Credentials

Create `.env` from `.env.example`. Detailed setup is in `docs/ENV_SETUP.md`.

- `GROQ_API_KEY`: create an API key in the Groq console at `https://console.groq.com/keys`.
- `GROQ_MODEL`: defaults to `qwen/qwen3.6-27b`, Groq's current multimodal model used for screenshot + text claim extraction.
- `DOCSNAP_FLARE_PRIVATE_KEY`: private key for a funded Coston2 wallet.
- `DOCSNAP_TEE_REPORTER_PRIVATE_KEY`: optional private key for the contract's TEE reporter account. For the hackathon demo, set this to the key used as `DOCSNAP_TEE_REPORTER` during contract deployment so the API can record the certificate in the same flow.
- `DOCSNAP_FLARE_RPC_URL`: defaults to `https://coston2-api.flare.network/ext/C/rpc`.
- `DOCSNAP_CONTRACT_ADDRESS`: deployed `DocSnapAnchor` contract address on Coston2.
- `DOCSNAP_TEE_URL`: optional FCC extension URL, for example `http://localhost:8787`. If omitted, the API uses a local certificate simulator.
- `DATABASE_URL`: Postgres connection string.

## Contracts

```bash
cd contracts
forge test
DOCSNAP_TEE_REPORTER=0x0000000000000000000000000000000000000001 forge script script/DeployDocSnapAnchor.s.sol --rpc-url "$DOCSNAP_FLARE_RPC_URL" --broadcast
```

## FCC Extension

```bash
cd fcc/docsnap/go
go test ./...
go run ./cmd/extension
```

## Required Before Coston2 Deployment

- Start Docker daemon
- Install Foundry
- Fund a Coston2 deployer wallet
- Add Flare RPC URL and private key
- Get Coston2 FCC indexer credentials from Flare
- Configure an ngrok or cloudflared public endpoint for the FCC proxy
