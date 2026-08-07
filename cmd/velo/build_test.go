package main

import "testing"

func TestDarwinSigningIdentityUsesExplicitIdentity(t *testing.T) {
	creds := &darwinCreds{
		AppleID:         "developer@example.com",
		TeamID:          "TEAM123456",
		SigningIdentity: "Developer ID Application: Example Company (TEAM123456)",
	}

	if got := darwinSigningIdentity(creds); got != creds.SigningIdentity {
		t.Fatalf("darwinSigningIdentity() = %q, want %q", got, creds.SigningIdentity)
	}
}

func TestDarwinSigningIdentityKeepsLegacyFallback(t *testing.T) {
	creds := &darwinCreds{
		AppleID: "Example Company",
		TeamID:  "TEAM123456",
	}
	want := "Developer ID Application: Example Company (TEAM123456)"

	if got := darwinSigningIdentity(creds); got != want {
		t.Fatalf("darwinSigningIdentity() = %q, want %q", got, want)
	}
}
