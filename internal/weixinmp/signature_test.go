package weixinmp

import "testing"

func TestSignature(t *testing.T) {
	got := Signature("testtoken", "1600000000", "nonce")
	want := "1282e75efd4abadbbda81cb879697196c4f90fb8"
	if got != want {
		t.Fatalf("Signature() = %q, want %q", got, want)
	}
}

func TestVerifySignatureCaseInsensitive(t *testing.T) {
	sig := "1282E75EFD4ABADBBDA81CB879697196C4F90FB8"
	if !VerifySignature(sig, "testtoken", "1600000000", "nonce") {
		t.Fatalf("VerifySignature() = false, want true")
	}
}

