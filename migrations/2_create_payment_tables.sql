CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TABLE IF EXISTS user_payment_subscriptions CASCADE;
DROP TABLE IF EXISTS payment_invoices CASCADE;
DROP TABLE IF EXISTS payment_payments CASCADE;
DROP TABLE IF EXISTS payment_subscriptions CASCADE;
DROP TABLE IF EXISTS payment_customers CASCADE;
DROP TABLE IF EXISTS user_payments CASCADE;

CREATE TABLE IF NOT EXISTS payment_customers (
    id VARCHAR PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    email TEXT DEFAULT '',
    cpfcnpj TEXT DEFAULT '',
    phone TEXT DEFAULT '',
    mobile_phone TEXT DEFAULT '',
    address TEXT DEFAULT '',
    address_number TEXT DEFAULT '',
    complement TEXT DEFAULT '',
    province TEXT DEFAULT '',
    postal_code TEXT DEFAULT '',
    notification_disabled BOOLEAN NOT NULL DEFAULT FALSE,
    additional_emails TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_timestamp_payment_customers
BEFORE UPDATE ON payment_customers
FOR EACH ROW EXECUTE PROCEDURE update_timestamp();

CREATE TABLE IF NOT EXISTS payment_subscriptions (
    id VARCHAR PRIMARY KEY,
    customer_id VARCHAR NOT NULL REFERENCES payment_customers(id) ON DELETE CASCADE,
    billing_type TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    value NUMERIC NOT NULL DEFAULT 0,
    cycle TEXT NOT NULL DEFAULT '',
    next_due_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT NOT NULL DEFAULT '',
    end_date TIMESTAMP DEFAULT NULL,
    max_payments INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_timestamp_payment_subscriptions
BEFORE UPDATE ON payment_subscriptions
FOR EACH ROW EXECUTE PROCEDURE update_timestamp();

CREATE TABLE IF NOT EXISTS payment_payments (
    id VARCHAR PRIMARY KEY,
    customer_id VARCHAR NOT NULL REFERENCES payment_customers(id) ON DELETE CASCADE,
    subscription_id VARCHAR REFERENCES payment_subscriptions(id) ON DELETE CASCADE,
    billing_type TEXT NOT NULL DEFAULT '',
    value NUMERIC NOT NULL DEFAULT 0,
    due_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT NOT NULL DEFAULT '',
    installment_count INTEGER NOT NULL DEFAULT 0,
    callback_success_url TEXT NOT NULL DEFAULT '',
    callback_auto_redirect BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT '',
    invoice_url TEXT NOT NULL DEFAULT '',
    transaction_receipt_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_timestamp_payment_payments
BEFORE UPDATE ON payment_payments
FOR EACH ROW EXECUTE PROCEDURE update_timestamp();

CREATE TABLE IF NOT EXISTS payment_invoices (
    id VARCHAR PRIMARY KEY,
    payment_id VARCHAR NOT NULL REFERENCES payment_payments(id) ON DELETE CASCADE,
    service_description TEXT NOT NULL DEFAULT '',
    observations TEXT NOT NULL DEFAULT '',
    value NUMERIC NOT NULL DEFAULT 0,
    deductions NUMERIC NOT NULL DEFAULT 0,
    effective_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    municipal_service_id TEXT NOT NULL DEFAULT '',
    municipal_service_code TEXT NOT NULL DEFAULT '',
    municipal_service_name TEXT NOT NULL DEFAULT '',
    update_payment BOOLEAN NOT NULL DEFAULT FALSE,
    taxes_retain_iss BOOLEAN NOT NULL DEFAULT FALSE,
    taxes_cofins NUMERIC NOT NULL DEFAULT 0,
    taxes_csll NUMERIC NOT NULL DEFAULT 0,
    taxes_inss NUMERIC NOT NULL DEFAULT 0,
    taxes_ir NUMERIC NOT NULL DEFAULT 0,
    taxes_pis NUMERIC NOT NULL DEFAULT 0,
    taxes_iss NUMERIC NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT '',
    payment_link TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_timestamp_payment_invoices
BEFORE UPDATE ON payment_invoices
FOR EACH ROW EXECUTE PROCEDURE update_timestamp();

CREATE TABLE IF NOT EXISTS user_payment_subscriptions (
    id VARCHAR PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payment_customer_id VARCHAR NOT NULL REFERENCES payment_customers(id) ON DELETE CASCADE,
    payment_subscription_id VARCHAR NOT NULL REFERENCES payment_subscriptions(id) ON DELETE CASCADE,
    deleted_at TIMESTAMP DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_timestamp_user_payment_subscriptions
BEFORE UPDATE ON user_payment_subscriptions
FOR EACH ROW EXECUTE PROCEDURE update_timestamp();
