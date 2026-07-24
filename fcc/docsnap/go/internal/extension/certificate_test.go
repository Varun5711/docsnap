package extension

import "testing"

func TestCertifyRejectsInvalidCommitment(t *testing.T) {
	_, err := Certify(CertifyRequest{EvidenceID: "ev_1", EvidenceCommitment: "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCertifyReturnsCertificate(t *testing.T) {
	req := CertifyRequest{
		EvidenceID:         "ev_1",
		EvidenceCommitment: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ScreenshotHash:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ScrapedTextHash:    "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		MetadataCommitment: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		ClaimsRoot:         "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	}

	result, err := Certify(req)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted {
		t.Fatal("expected accepted")
	}
	if len(result.CertificateHash) != 64 {
		t.Fatalf("unexpected certificate hash %q", result.CertificateHash)
	}
	if result.Signature == "" || result.PublicKey == "" {
		t.Fatal("expected signature and public key")
	}
}
