package payments

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PostgresRepository persists data in a PostgreSQL database.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository builds a repository backed by PostgreSQL.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// EnsureSchema creates database tables when they do not exist.
func (r *PostgresRepository) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS payment_customers (
id UUID PRIMARY KEY,
name TEXT NOT NULL,
email TEXT DEFAULT '',
cpfCnpj TEXT DEFAULT '',
            phone TEXT DEFAULT '',
            mobile_phone TEXT DEFAULT '',
            address TEXT DEFAULT '',
            address_number TEXT DEFAULT '',
            complement TEXT DEFAULT '',
            province TEXT DEFAULT '',
            postal_code TEXT DEFAULT '',
            notification_disabled BOOLEAN NOT NULL DEFAULT FALSE,
            additional_emails TEXT DEFAULT '',
            created_at TIMESTAMPTZ NOT NULL,
            updated_at TIMESTAMPTZ NOT NULL
);`,
		`CREATE TABLE IF NOT EXISTS payment_payments (
id UUID PRIMARY KEY,
customer_id UUID NOT NULL REFERENCES payment_customers(id),
subscription_id UUID DEFAULT NULL,
billing_type TEXT NOT NULL,
value NUMERIC NOT NULL,
due_date TIMESTAMPTZ NOT NULL,
            description TEXT DEFAULT '',
            installment_count INTEGER NOT NULL DEFAULT 0,
            callback_success_url TEXT DEFAULT '',
            callback_auto_redirect BOOLEAN NOT NULL DEFAULT FALSE,
            status TEXT DEFAULT '',
            invoice_url TEXT DEFAULT '',
            transaction_receipt_url TEXT DEFAULT '',
            created_at TIMESTAMPTZ NOT NULL,
            updated_at TIMESTAMPTZ NOT NULL
);`,
		`ALTER TABLE payment_payments ADD COLUMN IF NOT EXISTS subscription_id UUID;`,
		`CREATE TABLE IF NOT EXISTS payment_subscriptions (
id UUID PRIMARY KEY,
customer_id UUID NOT NULL REFERENCES payment_customers(id),
billing_type TEXT NOT NULL,
status TEXT DEFAULT '',
value NUMERIC NOT NULL,
            cycle TEXT NOT NULL,
            next_due_date TIMESTAMPTZ NOT NULL,
            description TEXT DEFAULT '',
            end_date TIMESTAMPTZ,
            max_payments INTEGER NOT NULL DEFAULT 0,
            created_at TIMESTAMPTZ NOT NULL,
            updated_at TIMESTAMPTZ NOT NULL
);`,
		`CREATE TABLE IF NOT EXISTS payment_invoices (
id UUID PRIMARY KEY,
payment_id UUID NOT NULL REFERENCES payment_payments(id),
service_description TEXT NOT NULL,
observations TEXT NOT NULL,
            value NUMERIC NOT NULL,
            deductions NUMERIC NOT NULL DEFAULT 0,
            effective_date TIMESTAMPTZ NOT NULL,
            municipal_service_id TEXT DEFAULT '',
            municipal_service_code TEXT DEFAULT '',
            municipal_service_name TEXT NOT NULL,
            update_payment BOOLEAN NOT NULL DEFAULT FALSE,
            taxes_retain_iss BOOLEAN NOT NULL DEFAULT FALSE,
            taxes_cofins NUMERIC NOT NULL DEFAULT 0,
            taxes_csll NUMERIC NOT NULL DEFAULT 0,
            taxes_inss NUMERIC NOT NULL DEFAULT 0,
            taxes_ir NUMERIC NOT NULL DEFAULT 0,
            taxes_pis NUMERIC NOT NULL DEFAULT 0,
            taxes_iss NUMERIC NOT NULL DEFAULT 0,
            status TEXT DEFAULT '',
            payment_link TEXT DEFAULT '',
            created_at TIMESTAMPTZ NOT NULL,
            updated_at TIMESTAMPTZ NOT NULL
        );`,
	}

	for _, stmt := range stmts {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("falha na migra\u00e7\u00e3o do schema: %w", err)
		}
	}
	return nil
}

// SaveCustomer inserts a new customer.
func (r *PostgresRepository) SaveCustomer(ctx context.Context, customer CustomerRecord) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO payment_customers (
id,
name,
email,
cpfCnpj,
phone,
mobile_phone,
address,
address_number,
complement,
province,
postal_code,
notification_disabled,
additional_emails,
created_at,
updated_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
`,
		customer.ID,
		customer.Name,
		customer.Email,
		customer.CpfCnpj,
		customer.Phone,
		customer.MobilePhone,
		customer.Address,
		customer.AddressNumber,
		customer.Complement,
		customer.Province,
		customer.PostalCode,
		customer.NotificationDisabled,
		customer.AdditionalEmails,
		customer.CreatedAt,
		customer.UpdatedAt,
	)
	return err
}

// FindCustomerByID returns a customer record by ID.
func (r *PostgresRepository) FindCustomerByID(ctx context.Context, id string) (CustomerRecord, error) {
	var customer CustomerRecord
	row := r.db.QueryRowContext(ctx, `
SELECT
id,
name,
email,
cpfCnpj,
phone,
mobile_phone,
address,
address_number,
complement,
province,
postal_code,
notification_disabled,
additional_emails,
created_at,
updated_at
FROM payment_customers
WHERE id = $1
`, id)
	if err := row.Scan(
		&customer.ID,
		&customer.Name,
		&customer.Email,
		&customer.CpfCnpj,
		&customer.Phone,
		&customer.MobilePhone,
		&customer.Address,
		&customer.AddressNumber,
		&customer.Complement,
		&customer.Province,
		&customer.PostalCode,
		&customer.NotificationDisabled,
		&customer.AdditionalEmails,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	); err != nil {
		return CustomerRecord{}, err
	}
	return customer, nil
}

func (r *PostgresRepository) FindCustomerByEmail(ctx context.Context, email string) (CustomerRecord, error) {
	var customer CustomerRecord
	row := r.db.QueryRowContext(ctx, `
SELECT
id,
name,
email,
cpfCnpj,
phone,
mobile_phone,
address,
address_number,
complement,
province,
postal_code,
notification_disabled,
additional_emails,
created_at,
updated_at
FROM payment_customers
WHERE lower(email) = lower($1)
LIMIT 1
`, email)
	if err := row.Scan(
		&customer.ID,
		&customer.Name,
		&customer.Email,
		&customer.CpfCnpj,
		&customer.Phone,
		&customer.MobilePhone,
		&customer.Address,
		&customer.AddressNumber,
		&customer.Complement,
		&customer.Province,
		&customer.PostalCode,
		&customer.NotificationDisabled,
		&customer.AdditionalEmails,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	); err != nil {
		return CustomerRecord{}, err
	}
	return customer, nil
}

// SavePayment inserts a new payment row.
func (r *PostgresRepository) SavePayment(ctx context.Context, payment PaymentRecord) error {
	var subscriptionID any
	if payment.SubscriptionID != "" {
		subscriptionID = payment.SubscriptionID
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO payment_payments (
id,
customer_id,
subscription_id,
billing_type,
value,
due_date,
description,
installment_count,
callback_success_url,
callback_auto_redirect,
status,
invoice_url,
transaction_receipt_url,
created_at,
updated_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
`,
		payment.ID,
		payment.CustomerID,
		subscriptionID,
		payment.BillingType,
		payment.Value,
		payment.DueDate,
		payment.Description,
		payment.InstallmentCount,
		payment.CallbackSuccessURL,
		payment.CallbackAutoRedirect,
		payment.Status,
		payment.InvoiceURL,
		payment.TransactionReceiptURL,
		payment.CreatedAt,
		payment.UpdatedAt,
	)
	return err
}

// UpdatePaymentStatus updates the status and links of a payment.
func (r *PostgresRepository) UpdatePaymentStatus(ctx context.Context, id, status, invoiceURL, receiptURL string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE payment_payments SET status=$1, invoice_url=$2, transaction_receipt_url=$3, updated_at=$4 WHERE id=$5`,
		status,
		invoiceURL,
		receiptURL,
		time.Now().UTC(),
		id,
	)
	if err != nil {
		return err
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FindPaymentByID returns a payment record by ID.
func (r *PostgresRepository) FindPaymentByID(ctx context.Context, id string) (PaymentRecord, error) {
	var payment PaymentRecord
	row := r.db.QueryRowContext(ctx, `
SELECT
id,
customer_id,
subscription_id,
billing_type,
value,
due_date,
description,
installment_count,
callback_success_url,
callback_auto_redirect,
status,
invoice_url,
transaction_receipt_url,
created_at,
updated_at
FROM payment_payments
WHERE id = $1
`, id)
	var subscriptionID sql.NullString
	if err := row.Scan(
		&payment.ID,
		&payment.CustomerID,
		&subscriptionID,
		&payment.BillingType,
		&payment.Value,
		&payment.DueDate,
		&payment.Description,
		&payment.InstallmentCount,
		&payment.CallbackSuccessURL,
		&payment.CallbackAutoRedirect,
		&payment.Status,
		&payment.InvoiceURL,
		&payment.TransactionReceiptURL,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	); err != nil {
		return PaymentRecord{}, err
	}
	if subscriptionID.Valid {
		payment.SubscriptionID = subscriptionID.String
	}
	return payment, nil
}

// SaveSubscription inserts a subscription row.
func (r *PostgresRepository) SaveSubscription(ctx context.Context, subscription SubscriptionRecord) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO payment_subscriptions (
id,
customer_id,
billing_type,
status,
value,
cycle,
next_due_date,
description,
end_date,
max_payments,
created_at,
updated_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
`,
		subscription.ID,
		subscription.CustomerID,
		subscription.BillingType,
		subscription.Status,
		subscription.Value,
		subscription.Cycle,
		subscription.NextDueDate,
		subscription.Description,
		subscription.EndDate,
		subscription.MaxPayments,
		subscription.CreatedAt,
		subscription.UpdatedAt,
	)
	return err
}

// FindSubscriptionByID returns a subscription record by ID.
func (r *PostgresRepository) FindSubscriptionByID(ctx context.Context, id string) (SubscriptionRecord, error) {
	var subscription SubscriptionRecord
	row := r.db.QueryRowContext(ctx, `
SELECT
id,
customer_id,
billing_type,
status,
value,
cycle,
next_due_date,
description,
end_date,
max_payments,
created_at,
updated_at
FROM payment_subscriptions
WHERE id = $1
`, id)
	if err := row.Scan(
		&subscription.ID,
		&subscription.CustomerID,
		&subscription.BillingType,
		&subscription.Status,
		&subscription.Value,
		&subscription.Cycle,
		&subscription.NextDueDate,
		&subscription.Description,
		&subscription.EndDate,
		&subscription.MaxPayments,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	); err != nil {
		return SubscriptionRecord{}, err
	}
	return subscription, nil
}

// UpdateSubscriptionStatus updates the subscription status locally.
func (r *PostgresRepository) UpdateSubscriptionStatus(ctx context.Context, id, status string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE payment_subscriptions SET status=$1, updated_at=$2 WHERE id=$3`, status, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) CancelPaymentsForSubscription(ctx context.Context, subscriptionID string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE payment_payments
SET status = 'CANCELED',
    updated_at = $1
WHERE subscription_id = $2
  AND status NOT IN ('RECEIVED', 'CONFIRMED', 'RECEIVED_IN_CASH')
`, time.Now().UTC(), subscriptionID)
	return err
}

// SaveInvoice inserts an invoice row.
func (r *PostgresRepository) SaveInvoice(ctx context.Context, invoice InvoiceRecord) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO payment_invoices (
id,
payment_id,
service_description,
observations,
value,
deductions,
effective_date,
municipal_service_id,
municipal_service_code,
municipal_service_name,
update_payment,
taxes_retain_iss,
taxes_cofins,
taxes_csll,
taxes_inss,
taxes_ir,
taxes_pis,
taxes_iss,
status,
payment_link,
created_at,
updated_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
`,
		invoice.ID,
		invoice.PaymentID,
		invoice.ServiceDescription,
		invoice.Observations,
		invoice.Value,
		invoice.Deductions,
		invoice.EffectiveDate,
		invoice.MunicipalServiceID,
		invoice.MunicipalServiceCode,
		invoice.MunicipalServiceName,
		invoice.UpdatePayment,
		invoice.TaxesRetainISS,
		invoice.TaxesCofins,
		invoice.TaxesCsll,
		invoice.TaxesINSS,
		invoice.TaxesIR,
		invoice.TaxesPIS,
		invoice.TaxesISS,
		invoice.Status,
		invoice.PaymentLink,
		invoice.CreatedAt,
		invoice.UpdatedAt,
	)
	return err
}

// FindInvoiceByPaymentID returns the first invoice linked to a payment.
func (r *PostgresRepository) FindInvoiceByPaymentID(ctx context.Context, paymentID string) (InvoiceRecord, error) {
	var invoice InvoiceRecord
	row := r.db.QueryRowContext(ctx, `
SELECT
id,
payment_id,
service_description,
observations,
value,
deductions,
effective_date,
municipal_service_id,
municipal_service_code,
municipal_service_name,
update_payment,
taxes_retain_iss,
taxes_cofins,
taxes_csll,
taxes_inss,
taxes_ir,
taxes_pis,
taxes_iss,
status,
payment_link,
created_at,
updated_at
FROM payment_invoices
WHERE payment_id = $1
LIMIT 1
`, paymentID)
	if err := row.Scan(
		&invoice.ID,
		&invoice.PaymentID,
		&invoice.ServiceDescription,
		&invoice.Observations,
		&invoice.Value,
		&invoice.Deductions,
		&invoice.EffectiveDate,
		&invoice.MunicipalServiceID,
		&invoice.MunicipalServiceCode,
		&invoice.MunicipalServiceName,
		&invoice.UpdatePayment,
		&invoice.TaxesRetainISS,
		&invoice.TaxesCofins,
		&invoice.TaxesCsll,
		&invoice.TaxesINSS,
		&invoice.TaxesIR,
		&invoice.TaxesPIS,
		&invoice.TaxesISS,
		&invoice.Status,
		&invoice.PaymentLink,
		&invoice.CreatedAt,
		&invoice.UpdatedAt,
	); err != nil {
		return InvoiceRecord{}, err
	}

	return invoice, nil
}

// UpdateInvoiceStatus updates invoice status locally.
func (r *PostgresRepository) UpdateInvoiceStatus(ctx context.Context, id, status string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE payment_invoices SET status=$1, updated_at=$2 WHERE id=$3`, status, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
