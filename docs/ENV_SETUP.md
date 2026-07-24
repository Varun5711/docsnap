# DocSnap Environment Setup

Copy `.env.example` to `.env` and fill these values.

## Required For Local App

`DATABASE_URL`

Default:

```bash
postgres://docsnap:docsnap@localhost:5432/docsnap?sslmode=disable
```

Run Postgres with:

```bash
docker compose -f infra/docker-compose.yml up -d postgres
```

`DOCSNAP_API_ADDR`

Default:

```bash
:8080
```

`DOCSNAP_APP_ORIGIN`

Default:

```bash
http://localhost:3000
```

## Required For Groq Claim Extraction

`GROQ_API_KEY`

Get it from:

```text
https://console.groq.com/keys
```

Create an API key, copy it once, and put it in `.env`.

`GROQ_MODEL`

Default:

```bash
qwen/qwen3.6-27b
```

Groq currently documents this as the multimodal model for screenshot/image plus text extraction.

`GROQ_BASE_URL`

Default:

```bash
https://api.groq.com/openai/v1
```

## Required For Flare Coston2 Anchoring

`DOCSNAP_FLARE_MODE`

Use:

```bash
coston2
```

Keep as `simulated` if you want local demo mode without sending transactions.

`DOCSNAP_FLARE_RPC_URL`

Default:

```bash
https://coston2-api.flare.network/ext/C/rpc
```

Flare Coston2 chain ID is `114`.

`DOCSNAP_FLARE_PRIVATE_KEY`

Private key of the wallet that submits `submitEvidence`.

How to get it:

```bash
cast wallet new
```

Fund the wallet with C2FLR from:

```text
https://faucet.flare.network
```

`DOCSNAP_TEE_REPORTER_PRIVATE_KEY`

Private key of the wallet allowed to call `recordTEECertificate`.

For the hackathon demo, deploy `DocSnapAnchor` with this wallet address as `DOCSNAP_TEE_REPORTER`, then put the private key here so the API can submit the certification transaction after evidence submission.

`DOCSNAP_CONTRACT_ADDRESS`

Address returned after deploying `DocSnapAnchor`.

Deploy with:

```bash
cd contracts
DOCSNAP_TEE_REPORTER=0xYourTEEReporterAddress forge script script/DeployDocSnapAnchor.s.sol --rpc-url "$DOCSNAP_FLARE_RPC_URL" --private-key "$DOCSNAP_FLARE_PRIVATE_KEY" --broadcast
```

## Optional FCC Extension

`DOCSNAP_TEE_URL`

Use this when running the DocSnap FCC extension separately:

```bash
http://localhost:8787
```

Start it with:

```bash
cd fcc/docsnap/go
go run ./cmd/extension
```

If this is empty, the API uses the same certificate format through a local certifier so development still works.

## Storage

`DOCSNAP_STORAGE_MODE`

Use:

```bash
minio
```

Set to `local` to store screenshots as plain files under `DOCSNAP_STORAGE_PATH` instead.

`DOCSNAP_S3_ENDPOINT`, `DOCSNAP_S3_ACCESS_KEY`, `DOCSNAP_S3_SECRET_KEY`, `DOCSNAP_S3_BUCKET`, `DOCSNAP_S3_USE_SSL`

Default values match `infra/docker-compose.yml`'s `minio` service. Start it with:

```bash
docker compose -f infra/docker-compose.yml up -d minio
```

The bucket is created automatically on API startup if it doesn't exist. Console is at `http://localhost:9001` (user/pass `docsnap` / `docsnap-secret`).

