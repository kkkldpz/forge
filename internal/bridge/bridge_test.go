package bridge

import (
	"testing"
)

func TestGlobalBridge(t *testing.T) {
	b1 := GlobalBridge()
	b2 := GlobalBridge()

	if b1 != b2 {
		t.Error("GlobalBridge should return the same instance")
	}
}

func TestBridge_Enable(t *testing.T) {
	b := GlobalBridge()

	err := b.Enable(18792, "test-secret")
	if err != nil {
		t.Errorf("Enable failed: %v", err)
	}

	if !b.IsEnabled() {
		t.Error("Bridge should be enabled after Enable()")
	}

	err = b.Enable(18793, "another-secret")
	if err == nil {
		t.Error("Expected error when enabling already enabled bridge")
	}
}

func TestBridge_Disable(t *testing.T) {
	b := GlobalBridge()
	b.Enable(18792, "test-secret")
	b.Disable()

	if b.IsEnabled() {
		t.Error("Bridge should be disabled after Disable()")
	}
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret"
	peerID := "peer-123"

	token := GenerateToken(secret, peerID)

	if !ValidateToken(secret, peerID, token) {
		t.Error("Valid token should pass validation")
	}

	if ValidateToken(secret, peerID, "invalid-token") {
		t.Error("Invalid token should fail validation")
	}

	if ValidateToken("wrong-secret", peerID, token) {
		t.Error("Token with wrong secret should fail validation")
	}
}

func TestBridge_ListPeers(t *testing.T) {
	b := GlobalBridge()

	peers := b.ListPeers()
	if len(peers) != 0 {
		t.Errorf("Expected 0 peers, got %d", len(peers))
	}
}
