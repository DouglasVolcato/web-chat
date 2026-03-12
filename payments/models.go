package payments

import "time"

// CustomerRecord represents a customer stored in the local database.
type CustomerRecord struct {
	ID                   string
	Name                 string
	Email                string
	CpfCnpj              string
	Phone                string
	MobilePhone          string
	Address              string
	AddressNumber        string
	Complement           string
	Province             string
	PostalCode           string
	NotificationDisabled bool
	AdditionalEmails     string
	GroupName            string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// PaymentRecord represents a payment persisted locally.
type PaymentRecord struct {
	ID                    string
	CustomerID            string
	SubscriptionID        string
	BillingType           string
	Value                 float64
	DueDate               time.Time
	Description           string
	InstallmentCount      int
	CallbackSuccessURL    string
	CallbackAutoRedirect  bool
	Status                string
	InvoiceURL            string
	TransactionReceiptURL string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// SubscriptionRecord represents a subscription persisted locally.
type SubscriptionRecord struct {
	ID          string
	CustomerID  string
	BillingType string
	Status      string
	Value       float64
	Cycle       string
	NextDueDate time.Time
	Description string
	EndDate     time.Time
	MaxPayments int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// InvoiceRecord represents an invoice persisted locally.
type InvoiceRecord struct {
	ID                   string
	PaymentID            string
	ServiceDescription   string
	Observations         string
	Value                float64
	Deductions           float64
	EffectiveDate        time.Time
	MunicipalServiceID   string
	MunicipalServiceCode string
	MunicipalServiceName string
	UpdatePayment        bool
	TaxesRetainISS       bool
	TaxesCofins          float64
	TaxesCsll            float64
	TaxesINSS            float64
	TaxesIR              float64
	TaxesPIS             float64
	TaxesISS             float64
	Status               string
	PaymentLink          string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
