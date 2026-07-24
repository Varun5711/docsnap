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
- AI target: multimodal claim extraction API
- Blockchain: Solidity contract on Flare Coston2
- Confidential compute: Go FCC extension

## Local First Run

```bash
go run ./services/api/cmd/api
npm install
npm run dev:web
npm run dev:extension
```

The API starts with an in-memory store and simulated Flare/FCC anchoring so capture, search, and verification can be demoed before Docker, Foundry, and Coston2 credentials are ready.

## Required Before Coston2 Deployment

- Start Docker daemon
- Install Foundry
- Fund a Coston2 deployer wallet
- Add Flare RPC URL and private key
- Get Coston2 FCC indexer credentials from Flare
- Configure an ngrok or cloudflared public endpoint for the FCC proxy

