package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type UserPayment struct {
	ID                    string       `json:"id"`
	UserID                string       `json:"user_id"`
	PaymentCustomerID     string       `json:"payment_customer_id"`
	PaymentSubscriptionID string       `json:"payment_subscription_id"`
	DeletedAt             sql.NullTime `json:"deleted_at"`
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
}

type SubscriptionOverview struct {
	SubscriptionID string       `json:"subscription_id"`
	CustomerID     string       `json:"customer_id"`
	Status         string       `json:"status"`
	Cycle          string       `json:"cycle"`
	Value          float64      `json:"value"`
	NextDueDate    sql.NullTime `json:"next_due_date"`
	Description    string       `json:"description"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

type PaymentHistory struct {
	PaymentID      string         `json:"payment_id"`
	SubscriptionID sql.NullString `json:"subscription_id"`
	DueDate        sql.NullTime   `json:"due_date"`
	Value          float64        `json:"value"`
	Status         string         `json:"status"`
	Description    string         `json:"description"`
	InvoiceURL     string         `json:"invoice_url"`
	PaymentLink    sql.NullString `json:"payment_link"`
	ReceiptURL     string         `json:"receipt_url"`
	InvoiceID      sql.NullString `json:"invoice_id"`
	InvoiceStatus  sql.NullString `json:"invoice_status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type PaymentCustomer struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Email                string    `json:"email"`
	CpfCnpj              string    `json:"cpfcnpj"`
	Phone                string    `json:"phone"`
	MobilePhone          string    `json:"mobile_phone"`
	Address              string    `json:"address"`
	AddressNumber        string    `json:"address_number"`
	Complement           string    `json:"complement"`
	Province             string    `json:"province"`
	PostalCode           string    `json:"postal_code"`
	NotificationDisabled bool      `json:"notification_disabled"`
	AdditionalEmails     string    `json:"additional_emails"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (p *UserPayment) Create(ctx context.Context, tx *sql.Tx) error {
	p.ID = uuid.NewString()

	query := `
        insert into user_payment_subscriptions (
            id,
            user_id,
            payment_customer_id,
            payment_subscription_id
        ) values (
            $1,
            $2,
            $3,
            $4
        )
    `

	_, err := ExecContext(
		tx,
		ctx,
		query,
		p.ID,
		p.UserID,
		p.PaymentCustomerID,
		p.PaymentSubscriptionID,
	)

	return err
}

func (p *UserPayment) Update(ctx context.Context, tx *sql.Tx) error {
	query := `
        update
            user_payment_subscriptions
        set
            payment_customer_id = $1,
            payment_subscription_id = $2
        where
            id = $3
            and deleted_at is null
    `

	_, err := ExecContext(
		tx,
		ctx,
		query,
		p.PaymentCustomerID,
		p.PaymentSubscriptionID,
		p.ID,
	)

	return err
}

func (p *UserPayment) Delete(ctx context.Context, tx *sql.Tx) error {
	query := `
        update
            user_payment_subscriptions
        set
            deleted_at = NOW()
        where
            id = $1
            and deleted_at is null
    `

	_, err := ExecContext(tx, ctx, query, p.ID)
	return err
}

func GetUserPayment(ctx context.Context, tx *sql.Tx, id string) (*UserPayment, error) {
	query := `
        select
            id,
            user_id,
            payment_customer_id,
            payment_subscription_id,
            deleted_at,
            created_at,
            updated_at
        from user_payment_subscriptions
        where id = $1
            and deleted_at is null
    `

	row := QueryRowContext(tx, ctx, query, id)

	var payment UserPayment
	err := row.Scan(
		&payment.ID,
		&payment.UserID,
		&payment.PaymentCustomerID,
		&payment.PaymentSubscriptionID,
		&payment.DeletedAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func GetUserPayments(ctx context.Context, tx *sql.Tx, userID string) ([]UserPayment, error) {
	query := `
        select
            id,
            user_id,
            payment_customer_id,
            payment_subscription_id,
            deleted_at,
            created_at,
            updated_at
        from user_payment_subscriptions
        where user_id = $1
            and deleted_at is null
    `

	rows, err := QueryContext(tx, ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []UserPayment

	for rows.Next() {
		var payment UserPayment

		err = rows.Scan(
			&payment.ID,
			&payment.UserID,
			&payment.PaymentCustomerID,
			&payment.PaymentSubscriptionID,
			&payment.DeletedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		payments = append(payments, payment)
	}

	return payments, nil
}

func GetUserSubscriptionOverview(ctx context.Context, tx *sql.Tx, userID string) ([]SubscriptionOverview, error) {
	query := `
        select
            ps.id,
            ps.customer_id,
            ps.status,
            ps.cycle,
            ps.value,
            ps.next_due_date,
            ps.description,
            ps.created_at,
            ps.updated_at
        from user_payment_subscriptions ups
        join payment_subscriptions ps on ps.id = ups.payment_subscription_id
        where ups.user_id = $1
            and ups.deleted_at is null
    `

	rows, err := QueryContext(tx, ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []SubscriptionOverview

	for rows.Next() {
		var subscription SubscriptionOverview

		err = rows.Scan(
			&subscription.SubscriptionID,
			&subscription.CustomerID,
			&subscription.Status,
			&subscription.Cycle,
			&subscription.Value,
			&subscription.NextDueDate,
			&subscription.Description,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, subscription)
	}

	return subscriptions, nil
}

func GetUserPaymentHistory(ctx context.Context, tx *sql.Tx, userID string) ([]PaymentHistory, error) {
	query := `
        select
            pp.id,
            pp.subscription_id,
            pp.due_date,
            pp.value,
            pp.status,
            pp.description,
            pp.invoice_url,
            pp.transaction_receipt_url,
            pi.id,
            pi.status,
            pi.payment_link,
            pp.created_at,
            pp.updated_at
        from user_payment_subscriptions ups
        join payment_payments pp on pp.subscription_id = ups.payment_subscription_id
        left join payment_invoices pi on pi.payment_id = pp.id
        where ups.user_id = $1
            and ups.deleted_at is null
        order by coalesce(pp.due_date, pp.created_at) desc
    `

	rows, err := QueryContext(tx, ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []PaymentHistory

	for rows.Next() {
		var item PaymentHistory

		if err := rows.Scan(
			&item.PaymentID,
			&item.SubscriptionID,
			&item.DueDate,
			&item.Value,
			&item.Status,
			&item.Description,
			&item.InvoiceURL,
			&item.ReceiptURL,
			&item.InvoiceID,
			&item.InvoiceStatus,
			&item.PaymentLink,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}

		history = append(history, item)
	}

	return history, nil
}

func GetLastPaidSubscriptionDueDate(ctx context.Context, tx *sql.Tx, userID string) (sql.NullTime, error) {
	query := `
        select
            max(pp.due_date)
        from user_payment_subscriptions ups
        join payment_payments pp on pp.subscription_id = ups.payment_subscription_id
        where ups.user_id = $1
            and pp.status in ('RECEIVED', 'CONFIRMED', 'RECEIVED_IN_CASH')
    `

	row := QueryRowContext(tx, ctx, query, userID)
	var due sql.NullTime
	err := row.Scan(&due)
	if err != nil {
		return sql.NullTime{}, err
	}
	return due, nil
}

func GetPaymentSubscriptionCycle(ctx context.Context, tx *sql.Tx, subscriptionID string) (string, error) {
	query := `
        select
            cycle
        from payment_subscriptions
        where id = $1
        limit 1
    `
	row := QueryRowContext(tx, ctx, query, subscriptionID)
	var cycle sql.NullString
	if err := row.Scan(&cycle); err != nil {
		return "", err
	}
	if !cycle.Valid {
		return "", nil
	}
	return cycle.String, nil
}

func GetUserPaymentCustomer(ctx context.Context, tx *sql.Tx, userID string) (*PaymentCustomer, error) {
	query := `
        select
            pc.id,
            pc.name,
            pc.email,
            pc.cpfcnpj,
            pc.phone,
            pc.mobile_phone,
            pc.address,
            pc.address_number,
            pc.complement,
            pc.province,
            pc.postal_code,
            pc.notification_disabled,
            pc.additional_emails,
            pc.created_at,
            pc.updated_at
        from user_payment_subscriptions ups
        join payment_customers pc on pc.id = ups.payment_customer_id
        where ups.user_id = $1
            and ups.deleted_at is null
        limit 1
    `

	row := QueryRowContext(tx, ctx, query, userID)

	var pc PaymentCustomer
	err := row.Scan(
		&pc.ID,
		&pc.Name,
		&pc.Email,
		&pc.CpfCnpj,
		&pc.Phone,
		&pc.MobilePhone,
		&pc.Address,
		&pc.AddressNumber,
		&pc.Complement,
		&pc.Province,
		&pc.PostalCode,
		&pc.NotificationDisabled,
		&pc.AdditionalEmails,
		&pc.CreatedAt,
		&pc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &pc, nil
}

func IsUserPaymentCurrent(ctx context.Context, tx *sql.Tx, userID string, reference time.Time) (bool, error) {
	query := `
        select
            exists (
                select 1
                from user_payment_subscriptions ups
                join payment_subscriptions ps on ps.id = ups.payment_subscription_id
                join payment_payments pp on pp.subscription_id = ups.payment_subscription_id
                where ups.user_id = $1
                    and ups.deleted_at is null
                    and ps.status = 'ACTIVE'
                    and pp.status in ('RECEIVED', 'CONFIRMED', 'RECEIVED_IN_CASH')
                    and date_trunc('month', pp.due_date) >= date_trunc('month', $2::timestamptz)
            )
    `

	row := QueryRowContext(tx, ctx, query, userID, reference)
	var current bool
	if err := row.Scan(&current); err != nil {
		return false, err
	}

	return current, nil
}
