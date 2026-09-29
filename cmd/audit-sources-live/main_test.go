package main

import "testing"

func TestAuditDecisionKeepsTLSFallbackBlocked(t *testing.T) {
	got := auditDecision("tls_unverified_http_200", "http_200")
	want := "audited_reachable_with_security_or_policy_restriction_still_blocked"
	if got != want {
		t.Fatalf("auditDecision = %q, want %q", got, want)
	}
}

func TestAuditDecisionKeepsHTTPRestrictedSourceBlocked(t *testing.T) {
	got := auditDecision("http_403", "http_200")
	want := "audited_unreachable_or_restricted_still_blocked"
	if got != want {
		t.Fatalf("auditDecision = %q, want %q", got, want)
	}
}
