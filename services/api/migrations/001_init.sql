CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS evidence (
  id text PRIMARY KEY,
  url text NOT NULL,
  domain text NOT NULL,
  title text NOT NULL DEFAULT '',
  company text NOT NULL DEFAULT '',
  case_id text NOT NULL DEFAULT '',
  user_id text NOT NULL DEFAULT '',
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

