package payments

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Service orchestrates local persistence and remote Asaas calls.
type Service struct {
	repo   *PostgresRepository
	client *AsaasClient
}

// NewService creates a payment service.
func NewService(repo *PostgresRepository, client *AsaasClient) *Service {
	return &Service{repo: repo, client: client}
}

// RegisterCustomer stores a local customer and creates it in Asaas.
func (s *Service) RegisterCustomer(ctx context.Context, req CustomerRequest) (CustomerRecord, CustomerResponse, error) {
	now := time.Now().UTC()
	local := CustomerRecord{
		ID:                   generateID(),
		Name:                 req.Name,
		Email:                req.Email,
		CpfCnpj:              req.CpfCnpj,
		Phone:                req.Phone,
		MobilePhone:          req.MobilePhone,
		Address:              req.Address,
		AddressNumber:        req.AddressNumber,
		Complement:           req.Complement,
		Province:             req.Province,
		PostalCode:           req.PostalCode,
		NotificationDisabled: req.NotificationDisabled,
		AdditionalEmails:     req.AdditionalEmails,
		GroupName:            req.GroupName,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	req.ExternalID = local.ID

	remote, err := s.client.CreateCustomer(ctx, req)
	if err != nil {
		return CustomerRecord{}, CustomerResponse{}, fmt.Errorf("falha ao criar cliente no Asaas: %w", err)
	}

	if err := s.repo.SaveCustomer(ctx, local); err != nil {
		return CustomerRecord{}, CustomerResponse{}, fmt.Errorf("falha ao salvar cliente local: %w", err)
	}

	return local, remote, nil
}

// CreatePayment persists the payment locally and in Asaas.
func (s *Service) CreatePayment(ctx context.Context, req PaymentRequest) (PaymentRecord, PaymentResponse, error) {
	customer, err := s.repo.FindCustomerByID(ctx, req.Customer)
	if err != nil {
		return PaymentRecord{}, PaymentResponse{}, fmt.Errorf("falha ao localizar cliente %s: %w", req.Customer, err)
	}

	remoteCustomer, err := s.client.GetCustomer(ctx, customer.ID)
	if err != nil {
		return PaymentRecord{}, PaymentResponse{}, fmt.Errorf("falha ao buscar cliente no Asaas para id %s: %w", req.Customer, err)
	}

	localID := generateID()
	req.ExternalID = localID
	asaasReq := req
	asaasReq.Customer = remoteCustomer.ID
	remote, err := s.client.CreatePayment(ctx, asaasReq)
	if err != nil {
		return PaymentRecord{}, PaymentResponse{}, fmt.Errorf("falha ao criar pagamento no Asaas: %w", err)
	}

	callbackSuccessURL := ""
	callbackAutoRedirect := false
	if req.Callback != nil {
		callbackSuccessURL = req.Callback.SuccessURL
		callbackAutoRedirect = req.Callback.AutoRedirect
	}

	now := time.Now().UTC()
	local := PaymentRecord{
		ID:                    localID,
		CustomerID:            customer.ID,
		BillingType:           req.BillingType,
		Value:                 req.Value,
		DueDate:               parseDate(req.DueDate),
		Description:           req.Description,
		InstallmentCount:      req.InstallmentCount,
		CallbackSuccessURL:    callbackSuccessURL,
		CallbackAutoRedirect:  callbackAutoRedirect,
		Status:                remote.Status,
		InvoiceURL:            remote.InvoiceURL,
		TransactionReceiptURL: remote.TransactionReceiptURL,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.repo.SavePayment(ctx, local); err != nil {
		return PaymentRecord{}, PaymentResponse{}, fmt.Errorf("falha ao salvar pagamento local: %w", err)
	}

	return local, remote, nil
}

// CreateSubscription persists the subscription locally and remotely.
func (s *Service) CreateSubscription(ctx context.Context, req SubscriptionRequest) (SubscriptionRecord, SubscriptionResponse, error) {
	customer, err := s.repo.FindCustomerByID(ctx, req.Customer)
	if err != nil {
		return SubscriptionRecord{}, SubscriptionResponse{}, fmt.Errorf("falha ao localizar cliente %s: %w", req.Customer, err)
	}

	remoteCustomer, err := s.client.GetCustomer(ctx, customer.ID)
	if err != nil {
		return SubscriptionRecord{}, SubscriptionResponse{}, fmt.Errorf("falha ao buscar cliente no Asaas para id %s: %w", req.Customer, err)
	}

	localID := generateID()
	req.ExternalID = localID
	asaasReq := req
	asaasReq.Customer = remoteCustomer.ID
	remote, err := s.client.CreateSubscription(ctx, asaasReq)
	if err != nil {
		return SubscriptionRecord{}, SubscriptionResponse{}, err
	}

	now := time.Now().UTC()
	local := SubscriptionRecord{
		ID:          localID,
		CustomerID:  customer.ID,
		BillingType: req.BillingType,
		Status:      remote.Status,
		Value:       req.Value,
		Cycle:       req.Cycle,
		NextDueDate: parseDate(req.NextDueDate),
		Description: req.Description,
		EndDate:     parseDate(req.EndDate),
		MaxPayments: req.MaxPayments,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.SaveSubscription(ctx, local); err != nil {
		return SubscriptionRecord{}, SubscriptionResponse{}, fmt.Errorf("falha ao salvar assinatura local: %w", err)
	}

	return local, remote, nil
}

func (s *Service) CancelSubscription(ctx context.Context, externalID string) (SubscriptionResponse, error) {
	if strings.TrimSpace(externalID) == "" {
		return SubscriptionResponse{}, errors.New("subscription id é obrigatório")
	}

	resp, err := s.client.CancelSubscription(ctx, externalID)
	if err != nil {
		return SubscriptionResponse{}, err
	}

	if err := s.repo.UpdateSubscriptionStatus(ctx, externalID, resp.Status); err != nil {
		return resp, err
	}

	if err := s.repo.CancelPaymentsForSubscription(ctx, externalID); err != nil {
		return resp, err
	}

	return resp, nil
}

// CreateInvoice persists the invoice locally and in Asaas.
func (s *Service) CreateInvoice(ctx context.Context, req InvoiceRequest) (InvoiceRecord, InvoiceResponse, error) {
	payment, err := s.repo.FindPaymentByID(ctx, req.Payment)
	if err != nil {
		return InvoiceRecord{}, InvoiceResponse{}, fmt.Errorf("falha ao localizar pagamento %s: %w", req.Payment, err)
	}

	remotePayment, err := s.client.GetPayment(ctx, payment.ID)
	if err != nil {
		return InvoiceRecord{}, InvoiceResponse{}, fmt.Errorf("falha ao buscar pagamento no Asaas para id %s: %w", req.Payment, err)
	}

	localID := req.ExternalID
	if localID == "" {
		localID = payment.ID
	}
	req.ExternalID = localID
	asaasReq := req
	asaasReq.Payment = remotePayment.ID
	remote, err := s.client.CreateInvoice(ctx, asaasReq)
	if err != nil {
		return InvoiceRecord{}, InvoiceResponse{}, fmt.Errorf("falha ao criar nota fiscal no Asaas: %w", err)
	}

	now := time.Now().UTC()
	local := InvoiceRecord{
		ID:                   localID,
		PaymentID:            payment.ID,
		ServiceDescription:   req.ServiceDescription,
		Observations:         req.Observations,
		Value:                req.Value,
		Deductions:           req.Deductions,
		EffectiveDate:        parseDate(req.EffectiveDate),
		MunicipalServiceID:   req.MunicipalServiceID,
		MunicipalServiceCode: req.MunicipalServiceCode,
		MunicipalServiceName: req.MunicipalServiceName,
		UpdatePayment:        req.UpdatePayment,
		TaxesRetainISS:       req.Taxes.RetainISS,
		TaxesCofins:          req.Taxes.Cofins,
		TaxesCsll:            req.Taxes.Csll,
		TaxesINSS:            req.Taxes.INSS,
		TaxesIR:              req.Taxes.IR,
		TaxesPIS:             req.Taxes.PIS,
		TaxesISS:             req.Taxes.ISS,
		Status:               remote.Status,
		PaymentLink:          remote.PaymentLink,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.repo.SaveInvoice(ctx, local); err != nil {
		return InvoiceRecord{}, InvoiceResponse{}, fmt.Errorf("falha ao salvar nota fiscal local: %w", err)
	}

	return local, remote, nil
}

// GetCustomerByEmail retrieves a saved customer by email.
func (s *Service) GetCustomerByEmail(ctx context.Context, email string) (CustomerRecord, error) {
	return s.repo.FindCustomerByEmail(ctx, email)
}

// HandleWebhookNotification updates local records based on webhook events.
func (s *Service) HandleWebhookNotification(ctx context.Context, event NotificationEvent) error {
	switch event.Event {
	case "PAYMENT_CREATED":
		if event.Payment == nil {
			return fmt.Errorf("payload de pagamento ausente")
		}
		if event.Payment.Subscription == "" {
			return nil
		}
		if event.Payment.ExternalReference != "" {
			if _, err := s.repo.FindPaymentByID(ctx, event.Payment.ExternalReference); err == nil {
				return nil
			} else if !errors.Is(err, sql.ErrNoRows) {
				return nil
				// return err
			}
		}

		subscription, err := s.client.GetSubscriptionByID(ctx, event.Payment.Subscription)
		if err != nil {
			return nil
			// return fmt.Errorf("falha ao buscar assinatura %s: %w", event.Payment.Subscription, err)
		}
		if subscription.ExternalID == "" {
			return nil
		}

		localSubscription, err := s.repo.FindSubscriptionByID(ctx, subscription.ExternalID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}

		localID := generateID()
		now := time.Now().UTC()
		localPayment := PaymentRecord{
			ID:                    localID,
			CustomerID:            localSubscription.CustomerID,
			SubscriptionID:        localSubscription.ID,
			BillingType:           event.Payment.BillingType,
			Value:                 event.Payment.Value,
			DueDate:               parseDate(event.Payment.DueDate),
			Description:           event.Payment.Description,
			InstallmentCount:      0,
			CallbackSuccessURL:    "",
			CallbackAutoRedirect:  false,
			Status:                event.Payment.Status,
			InvoiceURL:            event.Payment.InvoiceURL,
			TransactionReceiptURL: event.Payment.TransactionReceiptURL,
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		if err := s.repo.SavePayment(ctx, localPayment); err != nil {
			return fmt.Errorf("falha ao salvar pagamento local: %w", err)
		}
		if event.Payment.ID != "" && event.Payment.ExternalReference != localID {
			if err := s.client.UpdatePaymentExternalReference(ctx, event.Payment.ID, localID); err != nil {
				return fmt.Errorf("falha ao atualizar externalReference do pagamento: %w", err)
			}
		}
		return nil
	case "INVOICE_CREATED", "SUBSCRIPTION_CREATED":
		return nil
	case "PAYMENT_AUTHORIZED", "PAYMENT_APPROVED_BY_RISK_ANALYSIS", "PAYMENT_CONFIRMED", "PAYMENT_ANTICIPATED", "PAYMENT_DELETED", "PAYMENT_REFUNDED", "PAYMENT_REFUND_DENIED", "PAYMENT_CHARGEBACK_REQUESTED", "PAYMENT_AWAITING_CHARGEBACK_REVERSAL", "PAYMENT_DUNNING_REQUESTED", "PAYMENT_CHECKOUT_VIEWED", "PAYMENT_PARTIALLY_REFUNDED", "PAYMENT_SPLIT_DIVERGENCE_BLOCK", "PAYMENT_AWAITING_RISK_ANALYSIS", "PAYMENT_REPROVED_BY_RISK_ANALYSIS", "PAYMENT_UPDATED", "PAYMENT_RECEIVED", "PAYMENT_OVERDUE", "PAYMENT_RESTORED", "PAYMENT_REFUND_IN_PROGRESS", "PAYMENT_RECEIVED_IN_CASH_UNDONE", "PAYMENT_CHARGEBACK_DISPUTE", "PAYMENT_DUNNING_RECEIVED", "PAYMENT_BANK_SLIP_VIEWED", "PAYMENT_CREDIT_CARD_CAPTURE_REFUSED", "PAYMENT_SPLIT_CANCELLED", "PAYMENT_SPLIT_DIVERGENCE_BLOCK_FINISHED":
		if event.Payment == nil {
			return fmt.Errorf("payload de pagamento ausente")
		}
		clientPayment, err := s.client.GetPaymentByID(ctx, event.Payment.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		payment, err := s.repo.FindPaymentByID(ctx, clientPayment.ExternalReference)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		if err := s.repo.UpdatePaymentStatus(ctx, payment.ID, event.Payment.Status, event.Payment.InvoiceURL, event.Payment.TransactionReceiptURL); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return s.issueInvoiceForPayment(ctx, payment, *event.Payment)
	case "SUBSCRIPTION_INACTIVATED", "SUBSCRIPTION_SPLIT_DISABLED", "SUBSCRIPTION_SPLIT_DIVERGENCE_BLOCK_FINISHED", "SUBSCRIPTION_UPDATED", "SUBSCRIPTION_DELETED", "SUBSCRIPTION_SPLIT_DIVERGENCE_BLOCK":
		if event.Subscription == nil {
			return fmt.Errorf("payload de assinatura ausente")
		}
		if err := s.repo.UpdateSubscriptionStatus(ctx, event.Subscription.ExternalID, event.Subscription.Status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return nil
	case "INVOICE_SYNCHRONIZED", "INVOICE_PROCESSING_CANCELLATION", "INVOICE_CANCELLATION_DENIED", "INVOICE_UPDATED", "INVOICE_AUTHORIZED", "INVOICE_CANCELED", "INVOICE_ERROR":
		if event.Invoice == nil {
			return fmt.Errorf("payload de nota fiscal ausente")
		}
		if err := s.repo.UpdateInvoiceStatus(ctx, event.Invoice.ExternalID, event.Invoice.Status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return nil
	default:
		return nil
	}
}
func parseDate(value string) time.Time {
	// Asaas uses yyyy-mm-dd format; parsing errors return zero time for caller validation.
	t, _ := time.Parse("2006-01-02", value)
	return t
}

// ParseDateForTests exposes parseDate for integration tests without changing production API.
func ParseDateForTests(value string) time.Time {
	return parseDate(value)
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:])
}

func (s *Service) issueInvoiceForPayment(ctx context.Context, payment PaymentRecord, payload PaymentResponse) error {
	if _, err := s.repo.FindInvoiceByPaymentID(ctx, payment.ID); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	req := InvoiceRequest{
		Payment: payment.ID,
		ServiceDescription: func() string {
			if payment.Description != "" {
				return payment.Description
			}
			return fmt.Sprintf("Pagamento %s", payment.ID)
		}(),
		Observations:         "NOTA FISCAL EMITIDA POR EMPRESA OPTANTE DO SIMPLES NACIONAL CONFORME LEI COMPLEMENTAR 123/2006. NÃO GERA DIREITO A CRÉDITO DE I.P.I./ICMS.",
		ExternalID:           payment.ID,
		Value:                payment.Value,
		Deductions:           0,
		EffectiveDate:        time.Now().UTC().Format("2006-01-02"),
		MunicipalServiceCode: "01.03.01",
		MunicipalServiceName: "Processamento, armazenamento ou hospedagem de dados, textos, imagens, vídeos, páginas eletrônicas, aplicativos e sistemas de informação, entre outros formatos, e congêneres",
		UpdatePayment:        true,
		Taxes: InvoiceTaxes{
			RetainISS: false,
			Cofins:    0,
			Csll:      0,
			INSS:      0,
			IR:        0,
			PIS:       0,
			ISS:       5,
		},
	}

	if _, _, err := s.CreateInvoice(ctx, req); err != nil {
		return fmt.Errorf("falha ao emitir nota fiscal para o pagamento %s: %w", payment.ID, err)
	}

	return nil
}
