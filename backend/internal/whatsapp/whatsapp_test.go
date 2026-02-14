package whatsapp

import (
	"testing"
)

func TestSendReminder_NotConfigured_ReturnsNil(t *testing.T) {
	// Cliente sem credenciais não envia e retorna nil (no-op).
	c := NewClient(Config{})
	err := c.SendReminder("+5511999990000", "Maria", "12/02/2025", "14:30")
	if err != nil {
		t.Errorf("SendReminder sem config deve retornar nil, got %v", err)
	}
}

func TestSendReminder_EmptyAccountSid_ReturnsNil(t *testing.T) {
	c := NewClient(Config{AuthToken: "token", From: "whatsapp:+15551234567"})
	err := c.SendReminder("+5511999990000", "Maria", "12/02/2025", "14:30")
	if err != nil {
		t.Errorf("SendReminder sem AccountSid deve retornar nil, got %v", err)
	}
}

func TestSendReminder_EmptyFrom_ReturnsNil(t *testing.T) {
	c := NewClient(Config{AccountSid: "sid", AuthToken: "token"})
	err := c.SendReminder("+5511999990000", "Maria", "12/02/2025", "14:30")
	if err != nil {
		t.Errorf("SendReminder sem From deve retornar nil, got %v", err)
	}
}

func TestNewClient_ReturnsClient(t *testing.T) {
	c := NewClient(Config{AccountSid: "sid", AuthToken: "token", From: "whatsapp:+15551234567"})
	if c == nil {
		t.Fatal("NewClient não deve retornar nil quando config preenchido")
	}
}
