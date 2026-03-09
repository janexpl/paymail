package internal

import "testing"

func TestParseEmployeeDirectoryBuildsLookup(t *testing.T) {
	payload := []byte(`{"employees":[{"name":"anna","email":"anna@example.com"},{"name":"bob","email":"bob@example.com"}]}`)

	dir, err := ParseEmployeeDirectory(payload)
	if err != nil {
		t.Fatalf("ParseEmployeeDirectory() returned error: %v", err)
	}

	email, ok := dir.EmailByName("anna")
	if !ok {
		t.Fatal("expected anna email to be present")
	}
	if email != "anna@example.com" {
		t.Fatalf("unexpected email: %s", email)
	}
}

func TestParseEmployeeDirectoryRejectsDuplicateNames(t *testing.T) {
	payload := []byte(`{"employees":[{"name":"anna","email":"anna@example.com"},{"name":"anna","email":"other@example.com"}]}`)

	if _, err := ParseEmployeeDirectory(payload); err == nil {
		t.Fatal("expected duplicate employee validation error")
	}
}
