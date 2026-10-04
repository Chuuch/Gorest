ALTER TABLE organizations
  ADD COLUMN legal_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN registration_number TEXT NOT NULL DEFAULT '',
  ADD COLUMN vat_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN address_line1 TEXT NOT NULL DEFAULT '',
  ADD COLUMN address_line2 TEXT NOT NULL DEFAULT '',
  ADD COLUMN city TEXT NOT NULL DEFAULT '',
  ADD COLUMN postal_code TEXT NOT NULL DEFAULT '',
  ADD COLUMN country TEXT NOT NULL DEFAULT '',
  ADD COLUMN default_vat_rate_bps INTEGER NOT NULL DEFAULT 2000,
  ADD COLUMN bank_iban TEXT NOT NULL DEFAULT '',
  ADD COLUMN bank_bic TEXT NOT NULL DEFAULT '',
  ADD COLUMN bank_name TEXT NOT NULL DEFAULT '';

ALTER TABLE clients
  ADD COLUMN legal_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN vat_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN address_line1 TEXT NOT NULL DEFAULT '',
  ADD COLUMN address_line2 TEXT NOT NULL DEFAULT '',
  ADD COLUMN city TEXT NOT NULL DEFAULT '',
  ADD COLUMN postal_code TEXT NOT NULL DEFAULT '',
  ADD COLUMN country TEXT NOT NULL DEFAULT '';


ALTER TABLE invoices
  ADD COLUMN seller_legal_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN seller_registration_number TEXT NOT NULL DEFAULT '',
  ADD COLUMN seller_vat_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN seller_address_line1 TEXT NOT NULL DEFAULT '',
  ADD COLUMN seller_address_line2 TEXT NOT NULL DEFAULT '',
  ADD COLUMN seller_city TEXT NOT NULL DEFAULT '',
  ADD COLUMN seller_postal_code TEXT NOT NULL DEFAULT '',
  ADD COLUMN seller_country TEXT NOT NULL DEFAULT '',
  ADD COLUMN buyer_legal_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN buyer_vat_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN buyer_address_line1 TEXT NOT NULL DEFAULT '',
  ADD COLUMN buyer_address_line2 TEXT NOT NULL DEFAULT '',
  ADD COLUMN buyer_city TEXT NOT NULL DEFAULT '',
  ADD COLUMN buyer_postal_code TEXT NOT NULL DEFAULT '',
  ADD COLUMN buyer_country TEXT NOT NULL DEFAULT '',
  ADD COLUMN vat_regimie TEXT NOT NULL DEFAULT 'untaxed',
  ADD COLUMN vat_rate_bps INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN subtotal_cents INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN vat_cents INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN bank_iban TEXT NOT NULL DEFAULT '',
  ADD COLUMN bank_bic TEXT NOT NULL DEFAULT '';

UPDATE invoices
SET
  seller_legal_name = organization_name,
  buyer_legal_name = client_name,
  subtotal_cents = total_cents,
  vat_cents = 0,
  vat_regime = 'untaxed',
  vat_rate_bps = 0
WHERE subtotal_cents = 0 AND total_cents > 0;
