package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// AsaasClient handles authenticated HTTP communication with the Asaas API.
type AsaasClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewAsaasClient creates an AsaasClient using the provided configuration.
func NewAsaasClient(cfg Config) *AsaasClient {
	return &AsaasClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    cfg.APIURL,
		token:      cfg.APIToken,
	}
}

// CustomerRequest represents the payload for creating a customer in Asaas.
type CustomerRequest struct {
	Name                 string `json:"name"`
	Email                string `json:"email,omitempty"`
	CpfCnpj              string `json:"cpfCnpj,omitempty"`
	Phone                string `json:"phone,omitempty"`
	MobilePhone          string `json:"mobilePhone,omitempty"`
	Address              string `json:"address,omitempty"`
	AddressNumber        string `json:"addressNumber,omitempty"`
	Complement           string `json:"complement,omitempty"`
	Province             string `json:"province,omitempty"`
	PostalCode           string `json:"postalCode,omitempty"`
	ExternalID           string `json:"externalReference,omitempty"`
	NotificationDisabled bool   `json:"notificationDisabled,omitempty"`
	AdditionalEmails     string `json:"additionalEmails,omitempty"`
	GroupName            string `json:"groupName,omitempty"`
}

// CustomerResponse is the subset of Asaas response used by this module.
type CustomerResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	ExternalID string `json:"externalReference"`
}

type CustomerListResponse struct {
	Data []CustomerResponse `json:"data"`
}

// PaymentRequest represents the payload for creating a payment.
type PaymentRequest struct {
	Customer         string           `json:"customer"`
	BillingType      string           `json:"billingType"`
	Value            float64          `json:"value"`
	DueDate          string           `json:"dueDate"`
	Description      string           `json:"description,omitempty"`
	InstallmentCount int              `json:"installmentCount,omitempty"`
	ExternalID       string           `json:"externalReference,omitempty"`
	Callback         *PaymentCallback `json:"callback,omitempty"`
}

type PaymentCallback struct {
	SuccessURL   string `json:"successUrl"`
	AutoRedirect bool   `json:"autoRedirect"`
}

// PaymentResponse represents the relevant payment details returned by Asaas.
type PaymentResponse struct {
	ID                    string  `json:"id"`
	Customer              string  `json:"customer"`
	BillingType           string  `json:"billingType"`
	Value                 float64 `json:"value"`
	Status                string  `json:"status"`
	Description           string  `json:"description,omitempty"`
	DueDate               string  `json:"dueDate,omitempty"`
	ExternalReference     string  `json:"externalReference"`
	Subscription          string  `json:"subscription,omitempty"`
	InvoiceURL            string  `json:"invoiceUrl,omitempty"`
	TransactionReceiptURL string  `json:"transactionReceiptUrl,omitempty"`
}

type PaymentListResponse struct {
	Data []PaymentResponse `json:"data"`
}

type CreditCard struct {
	HolderName  string `json:"holderName"`
	Number      string `json:"number"`
	ExpiryMonth string `json:"expiryMonth"`
	ExpiryYear  string `json:"expiryYear"`
	CCV         string `json:"ccv"`
}

type CreditCardHolderInfo struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	CpfCnpj           string `json:"cpfCnpj"`
	PostalCode        string `json:"postalCode"`
	AddressNumber     string `json:"addressNumber"`
	AddressComplement string `json:"addressComplement,omitempty"`
	Phone             string `json:"phone"`
	MobilePhone       string `json:"mobilePhone,omitempty"`
}

// SubscriptionRequest represents creation of an Asaas subscription.
type SubscriptionRequest struct {
	Customer             string                `json:"customer"`
	BillingType          string                `json:"billingType"`
	Value                float64               `json:"value"`
	NextDueDate          string                `json:"nextDueDate"`
	Cycle                string                `json:"cycle"`
	ExternalID           string                `json:"externalReference,omitempty"`
	Description          string                `json:"description,omitempty"`
	EndDate              string                `json:"endDate,omitempty"`
	MaxPayments          int                   `json:"maxPayments,omitempty"`
	CreditCard           *CreditCard           `json:"creditCard,omitempty"`
	CreditCardHolderInfo *CreditCardHolderInfo `json:"creditCardHolderInfo,omitempty"`
}

// SubscriptionResponse captures required subscription fields.
type SubscriptionResponse struct {
	ID         string  `json:"id"`
	Customer   string  `json:"customer"`
	Status     string  `json:"status"`
	Value      float64 `json:"value"`
	ExternalID string  `json:"externalReference"`
}

type SubscriptionListResponse struct {
	Data []SubscriptionResponse `json:"data"`
}

// InvoiceRequest represents the payload to create an invoice in Asaas.
type InvoiceRequest struct {
	Payment              string       `json:"payment"`
	ServiceDescription   string       `json:"serviceDescription"`
	Observations         string       `json:"observations"`
	ExternalID           string       `json:"externalReference,omitempty"`
	Value                float64      `json:"value"`
	Deductions           float64      `json:"deductions"`
	EffectiveDate        string       `json:"effectiveDate"`
	MunicipalServiceID   string       `json:"municipalServiceId,omitempty"`
	MunicipalServiceCode string       `json:"municipalServiceCode,omitempty"`
	MunicipalServiceName string       `json:"municipalServiceName"`
	UpdatePayment        bool         `json:"updatePayment,omitempty"`
	Taxes                InvoiceTaxes `json:"taxes"`
}

type InvoiceTaxes struct {
	RetainISS bool    `json:"retainIss"`
	Cofins    float64 `json:"cofins"`
	Csll      float64 `json:"csll"`
	INSS      float64 `json:"inss"`
	IR        float64 `json:"ir"`
	PIS       float64 `json:"pis"`
	ISS       float64 `json:"iss"`
}

// InvoiceResponse captures invoice fields from Asaas.
type InvoiceResponse struct {
	ID          string  `json:"id"`
	Customer    string  `json:"customer"`
	Status      string  `json:"status"`
	Value       float64 `json:"value"`
	ExternalID  string  `json:"externalReference"`
	PaymentLink string  `json:"paymentLink"`
}

type InvoiceListResponse struct {
	Data []InvoiceResponse `json:"data"`
}

// NotificationEvent represents webhook payloads sent by Asaas.
type NotificationEvent struct {
	Event        string                `json:"event"`
	Payment      *PaymentResponse      `json:"payment,omitempty"`
	Invoice      *InvoiceResponse      `json:"invoice,omitempty"`
	Subscription *SubscriptionResponse `json:"subscription,omitempty"`
}

type AsaasError struct {
	StatusCode  int
	Code        string
	Description string
	Raw         string
}

func (e *AsaasError) Error() string {
	if strings.TrimSpace(e.Description) != "" {
		return e.Description
	}
	if strings.TrimSpace(e.Raw) != "" {
		return fmt.Sprintf("erro do Asaas %d: %s", e.StatusCode, e.Raw)
	}
	return fmt.Sprintf("erro do Asaas %d", e.StatusCode)
}

func (c *AsaasClient) doRequest(ctx context.Context, method, endpoint string, payload any, v any) error {
	return c.doRequestWithQuery(ctx, method, endpoint, nil, payload, v)
}

func (c *AsaasClient) doRequestWithQuery(ctx context.Context, method, endpoint string, query url.Values, payload any, v any) error {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("URL base inv\u00e1lida: %w", err)
	}
	pathPart, rawQuery, hasQuery := strings.Cut(endpoint, "?")
	base.Path = path.Join(base.Path, pathPart)
	if hasQuery {
		base.RawQuery = rawQuery
	}
	if len(query) > 0 {
		if base.RawQuery == "" {
			base.RawQuery = query.Encode()
		} else {
			base.RawQuery = base.RawQuery + "&" + query.Encode()
		}
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("falha ao serializar payload: %w", err)
		}
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, base.String(), body)
	if err != nil {
		return fmt.Errorf("falha ao criar requisi\u00e7\u00e3o: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("access_token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha na requisi\u00e7\u00e3o: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		var asaasError struct {
			Errors []struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(respBody, &asaasError); err == nil {
			if len(asaasError.Errors) > 0 && strings.TrimSpace(asaasError.Errors[0].Description) != "" {
				return &AsaasError{
					StatusCode:  resp.StatusCode,
					Code:        asaasError.Errors[0].Code,
					Description: asaasError.Errors[0].Description,
				}
			}
		}
		return &AsaasError{
			StatusCode: resp.StatusCode,
			Raw:        string(respBody),
		}
	}

	if v == nil {
		return nil
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("falha ao decodificar resposta: %w", err)
	}
	return nil
}

// CreateCustomer sends a request to create a customer.
func (c *AsaasClient) CreateCustomer(ctx context.Context, req CustomerRequest) (CustomerResponse, error) {
	var resp CustomerResponse
	err := c.doRequest(ctx, http.MethodPost, "customers", req, &resp)
	return resp, err
}

// GetCustomer retrieves a customer.
func (c *AsaasClient) GetCustomer(ctx context.Context, id string) (CustomerResponse, error) {
	var resp CustomerListResponse
	query := url.Values{}
	query.Set("externalReference", id)
	err := c.doRequestWithQuery(ctx, http.MethodGet, "customers", query, nil, &resp)
	if err != nil {
		return CustomerResponse{}, err
	}
	if len(resp.Data) == 0 {
		return CustomerResponse{}, fmt.Errorf("cliente n\u00e3o encontrado para externalReference=%s", id)
	}
	return resp.Data[0], nil
}

// CreatePayment creates a payment for a customer.
func (c *AsaasClient) CreatePayment(ctx context.Context, req PaymentRequest) (PaymentResponse, error) {
	var resp PaymentResponse
	err := c.doRequest(ctx, http.MethodPost, "payments", req, &resp)
	return resp, err
}

// GetPayment retrieves a payment by external reference.
func (c *AsaasClient) GetPayment(ctx context.Context, id string) (PaymentResponse, error) {
	var resp PaymentListResponse
	query := url.Values{}
	query.Set("externalReference", id)
	err := c.doRequestWithQuery(ctx, http.MethodGet, "payments", query, nil, &resp)
	if err != nil {
		return PaymentResponse{}, err
	}
	if len(resp.Data) == 0 {
		return PaymentResponse{}, fmt.Errorf("pagamento n\u00e3o encontrado para externalReference=%s", id)
	}
	return resp.Data[0], nil
}

func (c *AsaasClient) GetPaymentByID(ctx context.Context, id string) (PaymentResponse, error) {
	var resp PaymentResponse
	endpoint := path.Join("payments", id)
	err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &resp)
	return resp, err
}

// UpdatePaymentExternalReference updates the external reference for a payment.
func (c *AsaasClient) UpdatePaymentExternalReference(ctx context.Context, id, externalReference string) error {
	payload := struct {
		ExternalReference string `json:"externalReference"`
	}{ExternalReference: externalReference}
	endpoint := path.Join("payments", id)
	return c.doRequest(ctx, http.MethodPost, endpoint, payload, nil)
}

// CreateSubscription creates a recurring subscription.
func (c *AsaasClient) CreateSubscription(ctx context.Context, req SubscriptionRequest) (SubscriptionResponse, error) {
	var resp SubscriptionResponse
	err := c.doRequest(ctx, http.MethodPost, "subscriptions", req, &resp)
	return resp, err
}

// GetSubscription retrieves a subscription by external reference.
func (c *AsaasClient) GetSubscription(ctx context.Context, externalReference string) (SubscriptionResponse, error) {
	var resp SubscriptionListResponse
	query := url.Values{}
	query.Set("externalReference", externalReference)
	err := c.doRequestWithQuery(ctx, http.MethodGet, "subscriptions", query, nil, &resp)
	if err != nil {
		return SubscriptionResponse{}, err
	}
	if len(resp.Data) == 0 {
		return SubscriptionResponse{}, fmt.Errorf("assinatura n\u00e3o encontrada para externalReference=%s", externalReference)
	}
	return resp.Data[0], nil
}

// GetSubscriptionByID retrieves a subscription by its Asaas ID.
func (c *AsaasClient) GetSubscriptionByID(ctx context.Context, id string) (SubscriptionResponse, error) {
	var resp SubscriptionResponse
	endpoint := path.Join("subscriptions", id)
	err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &resp)
	return resp, err
}

// CancelSubscription cancels a subscription in Asaas.
func (c *AsaasClient) CancelSubscription(ctx context.Context, externalReference string) (SubscriptionResponse, error) {
	subscription, err := c.GetSubscription(ctx, externalReference)
	if err != nil {
		return SubscriptionResponse{}, err
	}
	endpoint := path.Join("subscriptions", subscription.ID)
	var resp SubscriptionResponse
	err = c.doRequest(ctx, http.MethodDelete, endpoint, nil, &resp)
	return resp, err
}

// CreateInvoice creates an invoice for a customer.
func (c *AsaasClient) CreateInvoice(ctx context.Context, req InvoiceRequest) (InvoiceResponse, error) {
	var resp InvoiceResponse
	err := c.doRequest(ctx, http.MethodPost, "invoices", req, &resp)
	return resp, err
}

// GetInvoice retrieves an invoice by external reference.
func (c *AsaasClient) GetInvoice(ctx context.Context, externalReference string) (InvoiceResponse, error) {
	var resp InvoiceListResponse
	query := url.Values{}
	query.Set("externalReference", externalReference)
	err := c.doRequestWithQuery(ctx, http.MethodGet, "invoices", query, nil, &resp)
	if err != nil {
		return InvoiceResponse{}, err
	}
	if len(resp.Data) == 0 {
		return InvoiceResponse{}, fmt.Errorf("nota fiscal n\u00e3o encontrada para externalReference=%s", externalReference)
	}
	return resp.Data[0], nil
}
