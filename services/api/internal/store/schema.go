package store

const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS evidence (
  id text PRIMARY KEY,
  url text NOT NULL,
  domain text NOT NULL,
  title text NOT NULL DEFAULT '',
  company text NOT NULL DEFAULT '',
  case_id text NOT NULL DEFAULT '',
  user_id text NOT NULL DEFAULT '',
  screenshot_data_url text NOT NULL DEFAULT '',
  screenshot_object_key text NOT NULL DEFAULT '',
  scraped_text text NOT NULL DEFAULT '',
  screenshot_hash text NOT NULL,
  scraped_text_hash text NOT NULL,
  metadata_commitment text NOT NULL,
  claims_root text NOT NULL,
  evidence_commitment text NOT NULL,
  flare_tx_hash text NOT NULL,
  tee_certificate_hash text NOT NULL,
  tee_signature text NOT NULL,
  verification_status text NOT NULL,
  captured_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE evidence ADD COLUMN IF NOT EXISTS screenshot_data_url text NOT NULL DEFAULT '';
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS screenshot_object_key text NOT NULL DEFAULT '';
-- published_by: the logged-in web user who captured this (empty for
-- extension/anonymous captures, which keep anchoring via the shared key).
-- anchor_submitter: the wallet address that actually signed the on-chain
-- submitEvidence call — the shared signer's address for the fallback path,
-- or the owner's own wallet once they complete the wallet-anchor flow.
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS published_by text NOT NULL DEFAULT '';
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS anchor_submitter text NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS claims (
  id text PRIMARY KEY,
  evidence_id text NOT NULL REFERENCES evidence(id) ON DELETE CASCADE,
  text text NOT NULL,
  type text NOT NULL,
  confidence numeric NOT NULL,
  source_excerpt text NOT NULL,
  hash text NOT NULL,
  status text NOT NULL
);

CREATE INDEX IF NOT EXISTS evidence_domain_idx ON evidence(domain);
CREATE INDEX IF NOT EXISTS evidence_company_idx ON evidence(company);
CREATE INDEX IF NOT EXISTS evidence_case_id_idx ON evidence(case_id);
CREATE INDEX IF NOT EXISTS evidence_created_at_idx ON evidence(created_at DESC);
CREATE INDEX IF NOT EXISTS claims_text_trgm_idx ON claims USING gin(text gin_trgm_ops);
CREATE INDEX IF NOT EXISTS claims_type_idx ON claims(type);

-- DocSnap: atomic claim fields + investigation verdict, bolted directly
-- onto claims (one claim = one current verdict for P0, no separate
-- investigations table). entities/reasoning stored as JSON-encoded text
-- rather than jsonb — avoids pgx jsonb type plumbing for a single write
-- path that only ever round-trips through Go's own json package anyway.
ALTER TABLE claims ADD COLUMN IF NOT EXISTS subject text NOT NULL DEFAULT '';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS predicate text NOT NULL DEFAULT '';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS object text NOT NULL DEFAULT '';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS claim_date text NOT NULL DEFAULT '';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS location text NOT NULL DEFAULT '';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS entities text NOT NULL DEFAULT '[]';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS investigation_status text NOT NULL DEFAULT '';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS investigation_confidence numeric NOT NULL DEFAULT 0;
ALTER TABLE claims ADD COLUMN IF NOT EXISTS reasoning text NOT NULL DEFAULT '{}';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS investigated_at timestamptz;

CREATE TABLE IF NOT EXISTS sources (
  id text PRIMARY KEY,
  claim_id text NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
  url text NOT NULL,
  name text NOT NULL DEFAULT '',
  source_type text NOT NULL DEFAULT 'publication',
  star_rating int NOT NULL DEFAULT 1,
  relationship text NOT NULL DEFAULT 'unrelated',
  relevance numeric NOT NULL DEFAULT 0,
  captured_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sources_claim_id_idx ON sources(claim_id);
CREATE INDEX IF NOT EXISTS claims_investigation_status_idx ON claims(investigation_status);

-- DocSnap Phase 2: public discovery network. A claims row already IS an
-- investigation (one extracted claim, one verdict) — canonical_claims groups
-- multiple independent investigations of the same real-world claim under one
-- public URL instead of introducing a parallel "investigations" entity.
CREATE TABLE IF NOT EXISTS users (
  id text PRIMARY KEY,
  email text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  display_name text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
  token text PRIMARY KEY,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions(user_id);

CREATE TABLE IF NOT EXISTS canonical_claims (
  id text PRIMARY KEY,
  slug text NOT NULL UNIQUE,
  text text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS canonical_claims_text_trgm_idx ON canonical_claims USING gin(text gin_trgm_ops);

ALTER TABLE claims ADD COLUMN IF NOT EXISTS canonical_claim_id text REFERENCES canonical_claims(id) ON DELETE SET NULL;
ALTER TABLE claims ADD COLUMN IF NOT EXISTS visibility text NOT NULL DEFAULT 'private';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS published_by text NOT NULL DEFAULT '';
ALTER TABLE claims ADD COLUMN IF NOT EXISTS forked_from_claim_id text REFERENCES claims(id) ON DELETE SET NULL;
-- claims never had its own timestamp before Phase 2 (only the parent
-- evidence row did) — needed now to order multiple investigations/forks of
-- the same canonical claim chronologically.
ALTER TABLE claims ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS claims_canonical_claim_id_idx ON claims(canonical_claim_id);
CREATE INDEX IF NOT EXISTS claims_visibility_idx ON claims(visibility);

CREATE TABLE IF NOT EXISTS evidence_contributions (
  id text PRIMARY KEY,
  claim_id text NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
  contributor_id text NOT NULL,
  type text NOT NULL,
  url text NOT NULL DEFAULT '',
  note text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS evidence_contributions_claim_id_idx ON evidence_contributions(claim_id);
`
