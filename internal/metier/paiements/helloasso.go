package paiements

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HelloAsso est un client minimal de l'API Checkout v5 (stdlib uniquement).
// Seul module autorisé à effectuer des appels distants (exception prévue par
// la spec).
type HelloAsso struct {
	BaseURL string
	Slug    string
	Token   string
	hc      *http.Client
}

func NewHelloAsso(baseURL, slug, token string) *HelloAsso {
	return &HelloAsso{
		BaseURL: baseURL, Slug: slug, Token: token,
		hc: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *HelloAsso) configure() bool {
	return c.Slug != "" && c.Token != "" && c.BaseURL != ""
}

type checkoutIntentRequest struct {
	TotalAmount   int               `json:"totalAmount"` // centimes
	InitialAmount int               `json:"initialAmount"`
	ItemName      string            `json:"itemName"`
	BackRef       string            `json:"backRef"`
	ReturnURL     string            `json:"returnUrl"`
	Metadata      map[string]string `json:"metadata"`
}

type checkoutIntentResponse struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	RedirectURL string `json:"redirectUrl"`
}

// CreerCheckoutIntent crée un checkout-intent et renvoie l'URL de paiement.
func (c *HelloAsso) CreerCheckoutIntent(itemName string, montantEuros float64, backRef, returnURL string, metadata map[string]string) (string, error) {
	centimes := int(montantEuros*100 + 0.5)
	body, _ := json.Marshal(checkoutIntentRequest{
		TotalAmount: centimes, InitialAmount: centimes,
		ItemName: itemName, BackRef: backRef, ReturnURL: returnURL, Metadata: metadata,
	})
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/v5/organizations/%s/checkout-intents", c.BaseURL, c.Slug), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("helloasso checkout-intents: HTTP %d: %s", resp.StatusCode, string(b))
	}
	var out checkoutIntentResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	if out.URL != "" {
		return out.URL, nil
	}
	if out.RedirectURL != "" {
		return out.RedirectURL, nil
	}
	return "", fmt.Errorf("helloasso: aucune URL de checkout dans la réponse")
}

type paymentResponse struct {
	ID     int    `json:"id"`
	Status string `json:"status"` // ex. Authorized, Refused, Refunded...
}

// GetPayment relit un paiement côté API (confirmation du webhook : les
// notifications des comptes association ne sont pas signées).
func (c *HelloAsso) GetPayment(id int) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v5/payments/%d", c.BaseURL, id), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("helloasso payments: HTTP %d: %s", resp.StatusCode, string(b))
	}
	var out paymentResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	return out.Status, nil
}

// WebhookNotification est le corps attendu du webhook HelloAsso.
type WebhookNotification struct {
	EventType string `json:"eventType"`
	Data      struct {
		ID int `json:"id"`
	} `json:"data"`
}
