package identity

import "testing"

func TestPasswordHasherRoundTrip(t *testing.T) {
	t.Parallel()

	hasher := NewPasswordHasher(testPasswordParams())
	encoded, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	valid, err := hasher.Verify(encoded, "correct horse battery staple")
	if err != nil || !valid {
		t.Fatalf("Verify(correct) = %v, %v", valid, err)
	}
	valid, err = hasher.Verify(encoded, "wrong password")
	if err != nil || valid {
		t.Fatalf("Verify(wrong) = %v, %v", valid, err)
	}
}

func TestPasswordHasherRejectsUnsafeParameters(t *testing.T) {
	t.Parallel()

	hasher := NewPasswordHasher(testPasswordParams())
	valid, err := hasher.Verify(
		"$argon2id$v=19$m=999999,t=3,p=2$bmV0c2NvcGUtc2FsdCE$bm90LWEtaGFzaC1idXQtbG9uZw",
		"password",
	)
	if err == nil || valid {
		t.Fatalf("Verify() = %v, %v, want unsafe parameter error", valid, err)
	}
}

func testPasswordParams() PasswordParams {
	return PasswordParams{
		Memory: 8 * 1024, Iterations: 1, Parallelism: 1,
		SaltLength: 16, KeyLength: 32,
	}
}
