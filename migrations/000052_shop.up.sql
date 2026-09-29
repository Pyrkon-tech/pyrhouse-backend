-- Organizer shop (shop.pyrhouse.space) — see docs/shop/PLAN.md in the workspace repo.
-- Shop accounts are separate from warehouse users (D2); confirmed orders become quests (D5).

CREATE TABLE shop_accounts (
    id            SERIAL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE CHECK (email = lower(email)),
    -- NULL until the first Google login; an allowlist entry is an account with google_sub IS NULL.
    google_sub    VARCHAR(255) UNIQUE,
    display_name  VARCHAR(255),
    avatar_url    TEXT,
    active        BOOLEAN NOT NULL DEFAULT true,
    access_source VARCHAR(16) NOT NULL CHECK (access_source IN ('domain', 'allowlist', 'invite')),
    created_by    INT REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ
);

CREATE TABLE shop_invites (
    id                 SERIAL PRIMARY KEY,
    token_hash         CHAR(64) NOT NULL UNIQUE, -- sha256 hex; the plaintext is shown once
    label              VARCHAR(255) NOT NULL,
    created_by         INT REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ NOT NULL,
    used_at            TIMESTAMPTZ,
    used_by_account_id INT REFERENCES shop_accounts(id) ON DELETE SET NULL,
    revoked_at         TIMESTAMPTZ
);

CREATE TABLE shop_products (
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    description   TEXT,
    image_url     TEXT,
    section       VARCHAR(100) NOT NULL DEFAULT '',
    sort_order    INT NOT NULL DEFAULT 0,
    category_id   INT NOT NULL REFERENCES item_category(id) ON DELETE RESTRICT,
    price         NUMERIC(10, 2) CHECK (price >= 0),
    max_per_order INT CHECK (max_per_order > 0),
    active        BOOLEAN NOT NULL DEFAULT true,
    created_by    INT REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE shop_delivery_windows (
    id         SERIAL PRIMARY KEY,
    kind       VARCHAR(16) NOT NULL CHECK (kind IN ('delivery', 'return')),
    starts_at  TIMESTAMPTZ NOT NULL,
    ends_at    TIMESTAMPTZ NOT NULL,
    label      VARCHAR(255) NOT NULL DEFAULT '',
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);

CREATE TABLE shop_orders (
    id                 SERIAL PRIMARY KEY,
    number             VARCHAR(16) GENERATED ALWAYS AS ('SKL-' || lpad(id::text, 4, '0')) STORED UNIQUE,
    account_id         INT NOT NULL REFERENCES shop_accounts(id),
    location_id        INT NOT NULL REFERENCES locations(id),
    location_note      TEXT,
    contact_name       VARCHAR(255) NOT NULL,
    contact_phone      VARCHAR(50),
    budget_owner       VARCHAR(255),
    delivery_window_id INT NOT NULL REFERENCES shop_delivery_windows(id),
    return_window_id   INT NOT NULL REFERENCES shop_delivery_windows(id),
    return_date        DATE,
    notes              TEXT,
    status             VARCHAR(16) NOT NULL DEFAULT 'submitted'
                       CHECK (status IN ('submitted', 'confirmed', 'rejected', 'cancelled')),
    status_reason      TEXT,
    decided_by         INT REFERENCES users(id) ON DELETE SET NULL,
    decided_at         TIMESTAMPTZ,
    version            INT NOT NULL DEFAULT 1, -- optimistic lock
    idempotency_key    VARCHAR(100),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_id, idempotency_key)
);
CREATE INDEX idx_shop_orders_account ON shop_orders(account_id);
CREATE INDEX idx_shop_orders_status ON shop_orders(status);

CREATE TABLE shop_order_items (
    id           SERIAL PRIMARY KEY,
    order_id     INT NOT NULL REFERENCES shop_orders(id) ON DELETE CASCADE,
    product_id   INT NOT NULL REFERENCES shop_products(id) ON DELETE RESTRICT,
    quantity     INT NOT NULL CHECK (quantity > 0),
    -- Snapshots taken when the order is submitted or edited
    product_name VARCHAR(255) NOT NULL,
    category_id  INT NOT NULL REFERENCES item_category(id),
    unit_price   NUMERIC(10, 2),
    UNIQUE (order_id, product_id)
);

CREATE TABLE shop_order_events (
    id         SERIAL PRIMARY KEY,
    order_id   INT NOT NULL REFERENCES shop_orders(id) ON DELETE CASCADE,
    actor_kind VARCHAR(16) NOT NULL CHECK (actor_kind IN ('account', 'user')),
    actor_id   INT NOT NULL,
    type       VARCHAR(32) NOT NULL,
    payload    JSONB NOT NULL DEFAULT '{}',
    at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_shop_order_events_order ON shop_order_events(order_id);

ALTER TABLE equipment_request_quests
    ADD COLUMN shop_order_id INT UNIQUE REFERENCES shop_orders(id) ON DELETE RESTRICT;

INSERT INTO app_settings (key, value, description) VALUES
    ('shop.show_prices', 'true', 'Shop: show prices to organizers (false = the API omits prices)'),
    ('shop.orders_open', 'false', 'Shop: organizers can submit and edit orders'),
    ('shop.orders_open_until', '', 'Shop: RFC3339 time after which orders close (empty = no deadline)'),
    ('shop.auth.domain_auto_join', 'true', 'Shop: Google Workspace accounts from shop.auth.auto_domains get in automatically'),
    ('shop.auth.auto_domains', 'pyrkon.pl', 'Shop: comma-separated Google Workspace domains (hd claim) for automatic access')
ON CONFLICT (key) DO NOTHING;
