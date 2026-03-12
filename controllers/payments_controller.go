package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"app/helpers"
	"app/models"
	"app/payments"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

type PaymentsController struct {
	service *payments.Service
}

func (c *PaymentsController) Service() *payments.Service {
	return c.service
}

type subscriptionRequestPayload struct {
	Name                 string                         `json:"name"`
	Email                string                         `json:"email"`
	CpfCnpj              string                         `json:"cpfcnpj"`
	Phone                string                         `json:"phone"`
	MobilePhone          string                         `json:"mobile_phone"`
	Address              string                         `json:"address"`
	AddressNumber        string                         `json:"address_number"`
	Complement           string                         `json:"complement"`
	Province             string                         `json:"province"`
	PostalCode           string                         `json:"postal_code"`
	AdditionalEmails     string                         `json:"additional_emails"`
	BillingType          string                         `json:"billing_type"`
	Value                float64                        `json:"value"`
	Cycle                string                         `json:"cycle"`
	NextDueDate          string                         `json:"next_due_date"`
	Description          string                         `json:"description"`
	EndDate              string                         `json:"end_date"`
	MaxPayments          int                            `json:"max_payments"`
	CreditCard           *payments.CreditCard           `json:"creditCard"`
	CreditCardHolderInfo *payments.CreditCardHolderInfo `json:"creditCardHolderInfo"`
}

func NewPaymentsController() (*PaymentsController, error) {
	cfg, err := payments.LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}

	repo := payments.NewPostgresRepository(models.DB)
	client := payments.NewAsaasClient(cfg)

	return &PaymentsController{service: payments.NewService(repo, client)}, nil
}

func (c *PaymentsController) RegisterRoutes(router chi.Router) {
	router.Route("/api/payments", func(r chi.Router) {
		r.Use(httprate.LimitByIP(30, time.Minute))

		r.Post("/webhook", c.handleWebhook)

		r.Group(func(protected chi.Router) {
			protected.Use(func(next http.Handler) http.Handler {
				return helpers.AuthDecorator(next.ServeHTTP)
			})

			protected.Post("/subscribe", c.handleSubscribe)
		})
	})

	router.Route("/app", func(r chi.Router) {
		r.Use(httprate.LimitByIP(30, time.Minute))
		r.Use(func(next http.Handler) http.Handler {
			return helpers.AuthDecorator(next.ServeHTTP)
		})

		r.Get("/billing", c.handleBillingGet)
		r.Post("/billing", c.handleBillingPost)
		r.Get("/billing/success", c.handleSubscriptionSuccess)
	})
}

func (c *PaymentsController) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	var payload subscriptionRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "payload inválido"})
		return
	}

	billingType := "CREDIT_CARD"
	cycle := strings.TrimSpace(payload.Cycle)
	nextDueDate := time.Now().Format("2006-01-02")
	if billingType == "" || cycle == "" || nextDueDate == "" || payload.Value <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "campos obrigatórios ausentes"})
		return
	}

	name := strings.TrimSpace(defaultString(payload.Name, user.Name))
	email := strings.TrimSpace(defaultString(payload.Email, user.Email))
	if name == "" || email == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "nome e e-mail são obrigatórios"})
		return
	}
	email = strings.ToLower(email)

	cpfDigits := onlyDigits(payload.CpfCnpj)
	if !isValidCPF(cpfDigits) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "cpf inválido"})
		return
	}

	phoneDigits := onlyDigits(payload.Phone)
	mobileDigits := onlyDigits(payload.MobilePhone)
	postalDigits := onlyDigits(payload.PostalCode)
	if mobileDigits == "" {
		mobileDigits = phoneDigits
	}

	existingSubscriptions, err := models.GetUserPayments(dbCtx, tx, user.ID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if len(existingSubscriptions) > 0 {
		respondJSON(w, http.StatusConflict, map[string]string{"error": "assinatura já criada"})
		return
	}

	customerReq := payments.CustomerRequest{
		Name:                 name,
		Email:                email,
		CpfCnpj:              cpfDigits,
		Phone:                phoneDigits,
		MobilePhone:          mobileDigits,
		Address:              strings.TrimSpace(payload.Address),
		AddressNumber:        strings.TrimSpace(payload.AddressNumber),
		Complement:           strings.TrimSpace(payload.Complement),
		Province:             strings.TrimSpace(payload.Province),
		PostalCode:           postalDigits,
		AdditionalEmails:     strings.TrimSpace(payload.AdditionalEmails),
		NotificationDisabled: true,
	}

	localCustomer, err := c.service.GetCustomerByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		localCustomer, _, err = c.service.RegisterCustomer(ctx, customerReq)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	var creditCard *payments.CreditCard
	var creditCardHolderInfo *payments.CreditCardHolderInfo
	if billingType == "CREDIT_CARD" {
		if payload.CreditCard == nil || payload.CreditCardHolderInfo == nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "dados do cartao obrigatorios"})
			return
		}

		holderInfo, err := buildCreditCardHolderInfo(
			payload.CreditCardHolderInfo.Name,
			payload.CreditCardHolderInfo.Email,
			payload.CreditCardHolderInfo.CpfCnpj,
			payload.CreditCardHolderInfo.PostalCode,
			payload.CreditCardHolderInfo.AddressNumber,
			payload.CreditCardHolderInfo.AddressComplement,
			payload.CreditCardHolderInfo.Phone,
			payload.CreditCardHolderInfo.MobilePhone,
		)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		cardHolderName := strings.TrimSpace(payload.CreditCard.HolderName)
		if cardHolderName == "" {
			cardHolderName = holderInfo.Name
		}
		card, err := buildCreditCard(cardHolderName, payload.CreditCard.Number, payload.CreditCard.ExpiryMonth, payload.CreditCard.ExpiryYear, payload.CreditCard.CCV)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		creditCardHolderInfo = holderInfo
		creditCard = card
	}

	subscriptionReq := payments.SubscriptionRequest{
		Customer:             localCustomer.ID,
		BillingType:          billingType,
		Value:                payload.Value,
		NextDueDate:          nextDueDate,
		Cycle:                cycle,
		Description:          strings.TrimSpace(payload.Description),
		EndDate:              strings.TrimSpace(payload.EndDate),
		MaxPayments:          payload.MaxPayments,
		CreditCard:           creditCard,
		CreditCardHolderInfo: creditCardHolderInfo,
	}

	localSubscription, remoteSubscription, err := c.service.CreateSubscription(ctx, subscriptionReq)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	link := models.UserPayment{
		UserID:                user.ID,
		PaymentCustomerID:     localCustomer.ID,
		PaymentSubscriptionID: localSubscription.ID,
	}

	if err := link.Create(dbCtx, tx); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"subscription_id": localSubscription.ID,
		"customer_id":     localCustomer.ID,
		"status":          remoteSubscription.Status,
	})
}

func (c *PaymentsController) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "corpo inválido"})
		return
	}

	if err := c.service.HandleWebhookPayload(ctx, payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (c *PaymentsController) handleBillingGet(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao carregar faturamento", Brand: "SUPER TEMPLATE", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	customer, err := models.GetUserPaymentCustomer(dbCtx, tx, user.ID)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao carregar dados", Brand: "SUPER TEMPLATE", Message: err.Error(), Path: r.URL.Path})
		return
	}

	data := map[string]any{
		"User":     user,
		"Customer": customer,
	}

	RenderTemplate(w, filepath.Join("app", "billing.ejs"), data)
}

func (c *PaymentsController) handleBillingPost(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		c.renderAlert(w, "danger", "Erro de banco de dados: "+err.Error())
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		c.renderAlert(w, "danger", "Sessão expirada.")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	cpfCnpj := strings.TrimSpace(r.FormValue("cpfcnpj"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	postalCode := strings.TrimSpace(r.FormValue("postal_code"))
	address := strings.TrimSpace(r.FormValue("address"))
	addressNumber := strings.TrimSpace(r.FormValue("address_number"))
	complement := strings.TrimSpace(r.FormValue("complement"))
	province := strings.TrimSpace(r.FormValue("province"))
	plan := strings.TrimSpace(r.FormValue("plan"))

	cardHolderName := strings.TrimSpace(r.FormValue("card_holder_name"))
	cardNumber := strings.TrimSpace(r.FormValue("card_number"))
	cardExpiryMonth := strings.TrimSpace(r.FormValue("card_expiry_month"))
	cardExpiryYear := strings.TrimSpace(r.FormValue("card_expiry_year"))
	cardCvv := strings.TrimSpace(r.FormValue("card_cvv"))

	cpfDigits := onlyDigits(cpfCnpj)
	phoneDigits := onlyDigits(phone)
	postalDigits := onlyDigits(postalCode)

	if name == "" || email == "" || cpfDigits == "" || postalDigits == "" || address == "" || addressNumber == "" || province == "" || plan == "" {
		c.renderAlert(w, "warning", "Por favor, preencha todos os campos obrigatórios para emissão e pagamento.")
		return
	}
	if !isValidCPF(cpfDigits) {
		c.renderAlert(w, "warning", "Informe um CPF válido (somente números).")
		return
	}
	if len(postalDigits) != 8 {
		c.renderAlert(w, "warning", "Informe um CEP válido.")
		return
	}
	if len(phoneDigits) < 10 {
		c.renderAlert(w, "warning", "Informe um telefone válido com DDD.")
		return
	}
	if cardHolderName == "" {
		cardHolderName = name
	}
	if cardNumber == "" || cardExpiryMonth == "" || cardExpiryYear == "" || cardCvv == "" {
		c.renderAlert(w, "warning", "Preencha todos os dados do cartao de credito.")
		return
	}
	email = strings.ToLower(email)

	// Define plan values
	value := 49.90
	cycle := "MONTHLY"
	description := "Assinatura Mensal Super Template Premium"
	if plan == "yearly" {
		value = 478.90
		cycle = "YEARLY"
		description = "Assinatura Anual Super Template Premium"
	}

	holderInfo, err := buildCreditCardHolderInfo(
		cardHolderName,
		email,
		cpfDigits,
		postalDigits,
		addressNumber,
		complement,
		phoneDigits,
		phoneDigits,
	)
	if err != nil {
		c.renderAlert(w, "warning", err.Error())
		return
	}

	cardInfo, err := buildCreditCard(cardHolderName, cardNumber, cardExpiryMonth, cardExpiryYear, cardCvv)
	if err != nil {
		c.renderAlert(w, "warning", err.Error())
		return
	}

	customerReq := payments.CustomerRequest{
		Name:                 name,
		Email:                email,
		CpfCnpj:              cpfDigits,
		Phone:                phoneDigits,
		MobilePhone:          phoneDigits,
		Address:              address,
		AddressNumber:        addressNumber,
		Complement:           complement,
		Province:             province,
		PostalCode:           postalDigits,
		NotificationDisabled: true,
		GroupName:            "Super Template",
	}

	localCustomer, err := c.service.GetCustomerByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			c.renderAlert(w, "danger", "Erro ao verificar cliente: "+err.Error())
			return
		}
		localCustomer, _, err = c.service.RegisterCustomer(ctx, customerReq)
		if err != nil {
			c.renderAlert(w, "danger", "Erro ao registrar no Asaas: "+err.Error())
			return
		}
	}
	customerID := localCustomer.ID

	subscriptionReq := payments.SubscriptionRequest{
		Customer:             customerID,
		BillingType:          "CREDIT_CARD",
		Value:                value,
		NextDueDate:          time.Now().Format("2006-01-02"),
		Cycle:                cycle,
		Description:          description,
		CreditCard:           cardInfo,
		CreditCardHolderInfo: holderInfo,
	}

	localSubscription, _, err := c.service.CreateSubscription(ctx, subscriptionReq)
	if err != nil {
		c.renderAlert(w, "danger", "Erro ao criar assinatura: "+err.Error())
		return
	}

	link := models.UserPayment{
		UserID:                user.ID,
		PaymentCustomerID:     customerID,
		PaymentSubscriptionID: localSubscription.ID,
	}

	if err := link.Create(dbCtx, tx); err != nil {
		c.renderAlert(w, "danger", "Erro ao salvar vínculo: "+err.Error())
		return
	}

	helpers.Redirect(w, r, "/app/billing/success")
	c.renderAlert(w, "success", "Assinatura criada com sucesso! Redirecionando...")
}

func (c *PaymentsController) handleSubscriptionSuccess(w http.ResponseWriter, r *http.Request) {
	// Esta URL serve para rastreamento de conversão e controle de pagamentos concluídos.
	ctx := r.Context()
	user, err := helpers.GetAuthUser(ctx, nil, r)
	if err == nil {
		log.Printf("[PURCHASE_CONFIRMED] UserID: %s, Email: %s", user.ID, user.Email)
	}

	RenderTemplate(w, filepath.Join("app", "billing_success.ejs"), map[string]any{
		"User": user,
	})
}

func (c *PaymentsController) renderAlert(w http.ResponseWriter, alertType, message string) {
	RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
		"Type":    alertType,
		"Message": message,
	})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func onlyDigits(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] >= '0' && value[i] <= '9' {
			builder.WriteByte(value[i])
		}
	}
	return builder.String()
}

func isValidCPF(cpfDigits string) bool {
	if len(cpfDigits) != 11 {
		return false
	}
	allSame := true
	for i := 1; i < len(cpfDigits); i++ {
		if cpfDigits[i] != cpfDigits[0] {
			allSame = false
			break
		}
	}
	return !allSame
}

func buildCreditCard(holderName, number, expiryMonth, expiryYear, ccv string) (*payments.CreditCard, error) {
	holderName = strings.TrimSpace(holderName)
	if holderName == "" {
		return nil, errors.New("Nome do titular do cartão é obrigatório.")
	}

	numberDigits := onlyDigits(number)
	if len(numberDigits) < 13 || len(numberDigits) > 19 {
		return nil, errors.New("Número do cartão inválido.")
	}

	monthDigits := onlyDigits(expiryMonth)
	if len(monthDigits) != 2 {
		return nil, errors.New("Mês de validade inválido.")
	}
	month, err := strconv.Atoi(monthDigits)
	if err != nil || month < 1 || month > 12 {
		return nil, errors.New("Mês de validade inválido.")
	}

	yearDigits := onlyDigits(expiryYear)
	if len(yearDigits) != 4 {
		return nil, errors.New("Ano de validade inválido.")
	}

	ccvDigits := onlyDigits(ccv)
	if len(ccvDigits) < 3 || len(ccvDigits) > 4 {
		return nil, errors.New("CVV inválido.")
	}

	return &payments.CreditCard{
		HolderName:  holderName,
		Number:      numberDigits,
		ExpiryMonth: monthDigits,
		ExpiryYear:  yearDigits,
		CCV:         ccvDigits,
	}, nil
}

func buildCreditCardHolderInfo(name, email, cpfCnpj, postalCode, addressNumber, complement, phone, mobilePhone string) (*payments.CreditCardHolderInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("Nome do titular do cartao e obrigatorio.")
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("Email do titular e obrigatorio.")
	}

	cpfDigits := onlyDigits(cpfCnpj)
	if !isValidCPF(cpfDigits) {
		return nil, errors.New("CPF do titular invalido.")
	}

	postalDigits := onlyDigits(postalCode)
	if len(postalDigits) != 8 {
		return nil, errors.New("CEP do titular invalido.")
	}

	addressNumber = strings.TrimSpace(addressNumber)
	if addressNumber == "" {
		return nil, errors.New("Numero do endereco do titular e obrigatorio.")
	}

	phoneDigits := onlyDigits(phone)
	if len(phoneDigits) < 10 {
		return nil, errors.New("Telefone do titular invalido.")
	}

	mobileDigits := onlyDigits(mobilePhone)
	if mobileDigits == "" {
		mobileDigits = phoneDigits
	}

	return &payments.CreditCardHolderInfo{
		Name:              name,
		Email:             email,
		CpfCnpj:           cpfDigits,
		PostalCode:        postalDigits,
		AddressNumber:     addressNumber,
		AddressComplement: strings.TrimSpace(complement),
		Phone:             phoneDigits,
		MobilePhone:       mobileDigits,
	}, nil
}
