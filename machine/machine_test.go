package machine_test

import (
	"strings"
	"testing"

	"github.com/Valentin-Kaiser/go-core/machine"
)

func TestGeneratorBasic(t *testing.T) {
	g := machine.New().WithCPU().WithSystemUUID().WithMotherboard().WithMAC().WithDisk()

	id, err := g.ID()
	if err != nil {
		t.Fatalf("ID() error = %v", err)
	}

	if len(id) != 64 {
		t.Errorf("ID() returned ID of length %d, expected 64", len(id))
	}

	// Test consistency
	id2, err := g.ID()
	if err != nil {
		t.Fatalf("ID() second call error = %v", err)
	}

	if id != id2 {
		t.Error("ID() returned different IDs on consecutive calls")
	}
}

func TestGeneratorWithSalt(t *testing.T) {
	salt := "test-salt"
	g := machine.New().WithCPU().WithSystemUUID().WithSalt(salt)

	id, err := g.ID()
	if err != nil {
		t.Fatalf("ID() with salt error = %v", err)
	}

	if len(id) != 64 {
		t.Errorf("ID() returned ID of length %d, expected 64", len(id))
	}

	// Different salts should produce different IDs
	g2 := machine.New().WithCPU().WithSystemUUID().WithSalt("different-salt")
	id2, err := g2.ID()
	if err != nil {
		t.Fatalf("ID() with different salt error = %v", err)
	}

	if id == id2 {
		t.Error("ID() should return different IDs for different salts")
	}
}

func TestGeneratorValidate(t *testing.T) {
	g := machine.New().WithCPU().WithSystemUUID()

	id, err := g.ID()
	if err != nil {
		t.Fatalf("ID() error = %v", err)
	}

	valid, err := g.Validate(id)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !valid {
		t.Error("Validate() returned false for valid ID")
	}

	// Test with invalid ID
	valid, err = g.Validate("invalid-id")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if valid {
		t.Error("Validate() returned true for invalid ID")
	}
}

func TestVMFriendly(t *testing.T) {
	g := machine.New().VMFriendly().WithSalt("vm-test")

	id, err := g.ID()
	if err != nil {
		t.Fatalf("VMFriendly().ID() error = %v", err)
	}

	if len(id) != 64 {
		t.Errorf("VMFriendly().ID() returned ID of length %d, expected 64", len(id))
	}

	// Test that it's different from full hardware
	g2 := machine.New().WithCPU().WithSystemUUID().WithMotherboard().WithMAC().WithDisk().WithSalt("vm-test")
	id2, err := g2.ID()
	if err != nil {
		t.Fatalf("Full hardware ID() error = %v", err)
	}

	if id == id2 {
		t.Error("VMFriendly() should produce different ID from full hardware")
	}
}

func TestNoIdentifiersError(t *testing.T) {
	g := machine.New() // No identifiers enabled

	_, err := g.ID()
	if err == nil {
		t.Error("ID() should return error when no identifiers are enabled")
	}

	expectedError := "no hardware identifiers found with current configuration"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("ID() error should contain %q, got %q", expectedError, err.Error())
	}
}

func TestGeneratorChaining(t *testing.T) {
	// Test that method chaining works
	g := machine.New().
		WithSalt("chain-test").
		WithCPU().
		WithSystemUUID().
		WithMotherboard()

	id, err := g.ID()
	if err != nil {
		t.Fatalf("Chained generator ID() error = %v", err)
	}

	if len(id) != 64 {
		t.Errorf("Chained generator ID() returned ID of length %d, expected 64", len(id))
	}

	// Verify it validates correctly
	valid, err := g.Validate(id)
	if err != nil {
		t.Fatalf("Chained generator Validate() error = %v", err)
	}

	if !valid {
		t.Error("Chained generator Validate() returned false for valid ID")
	}
}
