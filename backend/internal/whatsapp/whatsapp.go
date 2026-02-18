package whatsapp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Config holds credentials for sending WhatsApp messages (Twilio).
// Phone numbers must be E.164; From is the Twilio WhatsApp number (e.g. whatsapp:+14155238886).
type Config struct {
	AccountSid string
	AuthToken  string
	From       string // e.g. "whatsapp:+14155238886"
}

// Client sends WhatsApp messages via Twilio.
type Client struct {
	cfg    Config
	client *http.Client
}

// NewClient returns a WhatsApp client. If AccountSid or AuthToken is empty, SendReminder is a no-op and returns nil.
func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg, client: &http.Client{}}
}

// SendReminder sends a reminder text to the given phone (E.164).
// If WhatsApp is not configured (missing credentials), returns nil without sending.
func (c *Client) SendReminder(phone, patientName, dateStr, timeStr string) error {
	if c.cfg.AccountSid == "" || c.cfg.AuthToken == "" || c.cfg.From == "" {
		return nil
	}
	body := fmt.Sprintf("Lembrete: Amanhã (%s) às %s você tem consulta agendada para %s. Confirme sua presença se possível.", dateStr, timeStr, patientName)
	return c.send(phone, body)
}

func (c *Client) send(to, body string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("whatsapp: destinatário vazio")
	}
	if !strings.HasPrefix(to, "whatsapp:+") {
		to = "whatsapp:+" + strings.TrimLeft(to, "+")
	}
	from := c.cfg.From
	if !strings.HasPrefix(from, "whatsapp:") {
		from = "whatsapp:" + from
	}
	form := url.Values{}
	form.Set("To", to)
	form.Set("From", from)
	form.Set("Body", body)
	reqURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", c.cfg.AccountSid)
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.cfg.AccountSid, c.cfg.AuthToken)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	slurp, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("whatsapp: %s: read body: %w", resp.Status, err)
	}
	return fmt.Errorf("whatsapp: %s: %s", resp.Status, string(slurp))
}
