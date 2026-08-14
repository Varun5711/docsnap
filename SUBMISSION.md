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
