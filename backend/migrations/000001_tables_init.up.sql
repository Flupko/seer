CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email CITEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,

    password_hash BYTEA,
    provider_id TEXT NOT NULL CHECK (provider_id IN ('credentials', 'google', 'twitch')),
    provider_user_id TEXT NOT NULL, -- sub when external provider, otherwise email for credentials

    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    profile_image_key TEXT,

    hidden BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,

    status TEXT NOT NULL CHECK (status IN ('pending_email_verification', 'pending_profile_completion', 'activated')),

    version INTEGER NOT NULL DEFAULT 1,

    CONSTRAINT users_unique_provider_user_id UNIQUE (provider_id, provider_user_id),
    CONSTRAINT users_providers_credentials_password CHECK(provider_id != 'credentials' OR password_hash IS NOT NULL)
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    hash bytea NOT NULL UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,

    ip_first TEXT,
    ip_last TEXT,

    user_agent TEXT,
    client_os TEXT,
    client_browser TEXT,
    client_device TEXT,

    geo_country TEXT, 
    geo_city TEXT
);

CREATE INDEX idx_sessions_active ON sessions(user_id, last_used_at DESC);


CREATE TABLE tokens (
    hash BYTEA PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    scope TEXT NOT NULL CHECK(scope IN ('authentication', 'email_verification', 'profile_completion', 'password_reset'))
);

-------------------------------------------------------------------------------------------------------------

CREATE TABLE ledger_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id), -- null for house's accounts
    account_type TEXT NOT NULL CHECK (account_type IN ('custody', 'liability', 'house', 'owner_withdrawal')),
    balance BIGINT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL CHECK(currency IN ('USDT')),
    allow_negative_balance BOOLEAN NOT NULL,
    allow_positive_balance BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    version BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT user_account_has_user CHECK (
        (account_type IN ('house', 'liability') AND user_id IS NULL) OR 
        (account_type IN ('custody', 'liability') AND user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_user_account_currency_type ON ledger_accounts(user_id, currency, account_type) WHERE account_type IN ('custody', 'liability');


CREATE TABLE ledger_transfers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_account_id UUID NOT NULL REFERENCES ledger_accounts(id),
    to_account_id UUID NOT NULL REFERENCES ledger_accounts(id),
    amount_minor BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    idempotency_key TEXT NOT NULL,
    CONSTRAINT ledger_transfers_integrity CHECK (amount_minor > 0 AND from_account_id != to_account_id),
    CONSTRAINT ledger_transfers_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX ON ledger_transfers(from_account_id);
CREATE INDEX ON ledger_transfers(to_account_id);
CREATE INDEX ON ledger_transfers(created_at);


CREATE TABLE ledger_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES ledger_accounts(id),
    transfer_id UUID NOT NULL REFERENCES ledger_transfers(id),
    amount_minor BIGINT NOT NULL,
    account_previous_balance BIGINT NOT NULL,
    account_current_balance BIGINT NOT NULL,
    account_version BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ledger_entries_balance CHECK (account_current_balance = account_previous_balance + amount_minor)
);

CREATE INDEX idx_ledger_entries_account ON ledger_entries(account_id);
CREATE INDEX idx_ledger_entries_transfer ON ledger_entries(transfer_id);

-- Ensure ledger_entries are immutable
CREATE OR REPLACE FUNCTION ledger_entries_immutable() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'ledger_entries are immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ledger_entries_no_update_tgr
BEFORE UPDATE ON ledger_entries
FOR EACH ROW EXECUTE FUNCTION ledger_entries_immutable();

CREATE TRIGGER ledger_entries_no_delete_tgr
BEFORE DELETE ON ledger_entries
FOR EACH ROW EXECUTE FUNCTION ledger_entries_immutable();

-- Function to make a transfer
CREATE OR REPLACE FUNCTION ledger_check_account_balance_constraints(account ledger_accounts) RETURNS VOID AS $$
BEGIN
    -- If account doesn't allow negative balance and balance is negative, raise an error
    IF NOT account.allow_negative_balance AND (account.balance < 0) THEN
        RAISE EXCEPTION 'ledger_account_policy Account (id=%, type=%) does not allow negative balance', account.id, account.account_type;
    END IF;

    -- If account doesn't allow positive balance and balance is positive, raise an error
    IF NOT account.allow_positive_balance AND (account.balance > 0) THEN
        RAISE EXCEPTION 'ledger_account_policy Account (id=%, type=%) does not allow positive balance', account.id, account.account_type;
    END IF;
END;
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION ledger_create_transfer(from_account_id UUID, to_account_id UUID, amount BIGINT, idem TEXT)
RETURNS UUID AS $$
DECLARE
from_account ledger_accounts;
to_account ledger_accounts;
transfer_id UUID;
BEGIN

    -- Preliminary checks
    IF amount <= 0 THEN
        RAISE EXCEPTION 'invalid_amount Amount (%) must be positive', amount;
    END IF;

    IF from_account_id = to_account_id THEN
        RAISE EXCEPTION 'same_account Cannot transfer to the same account (id=%)', from_account_id;
    END IF;

    SELECT * INTO from_account FROM ledger_accounts 
    WHERE id = from_account_id FOR UPDATE;
    
    SELECT * INTO to_account FROM ledger_accounts 
    WHERE id = to_account_id FOR UPDATE;

    IF from_account IS NULL THEN
        RAISE EXCEPTION 'account_not_found From account (id=%) does not exist', from_account_id;
    END IF;

    IF to_account IS NULL THEN
        RAISE EXCEPTION 'account_not_found To account (id=%) does not exist', to_account_id;
    END IF;


    IF from_account.currency != to_account.currency THEN
        RAISE EXCEPTION 'different_currencies Cannot transfer between different currencies (% and %)', from_account.currency, to_account.currency;
    END IF;

    -- Update balances
    UPDATE ledger_accounts
    SET balance = balance - amount, version = version + 1
    WHERE ledger_accounts.id = from_account_id
    RETURNING * INTO from_account;

    -- Check balance constraints for the source account
    PERFORM ledger_check_account_balance_constraints(from_account);

    UPDATE ledger_accounts
    SET balance = balance + amount, version = version + 1
    WHERE ledger_accounts.id = to_account_id
    RETURNING * INTO to_account;

    -- Check balance constraints for the destination account
    PERFORM ledger_check_account_balance_constraints(to_account);

    INSERT INTO ledger_transfers(from_account_id, to_account_id, amount_minor, idempotency_key)
    VALUES (from_account_id, to_account_id, amount, idem)
    ON CONFLICT (idempotency_key) DO NOTHING
    RETURNING ledger_transfers.id INTO transfer_id;

    IF transfer_id IS NULL THEN
        RAISE EXCEPTION 'idempotency_key Idempotency key already used %', idem;
    END IF;

    -- Create entry for the source account (negative amount)
    INSERT INTO ledger_entries(account_id, transfer_id, amount_minor, account_previous_balance, account_current_balance, account_version)
    VALUES (from_account_id, transfer_id, - amount, from_account.balance + amount, from_account.balance, from_account.version);

    -- Create entry for the destination account (positive amount)
    INSERT INTO ledger_entries(account_id, transfer_id, amount_minor, account_previous_balance, account_current_balance, account_version)
    VALUES (to_account_id, transfer_id, amount, to_account.balance - amount, to_account.balance, to_account.version);

    RETURN transfer_id;

END;
$$ LANGUAGE plpgsql;



-----------------------------------------------------------------------------------------------------------

CREATE TABLE deposits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Transfer FROM company custody cash account TO user's liability account
    ledger_transfer_id UUID NOT NULL REFERENCES ledger_transfers(id),
    
    source_currency TEXT NOT NULL CHECK(source_currency IN ('USDT', 'BTC', 'ETH', 'SOL', 'USD', 'EUR')),
    source_amount_minor BIGINT NOT NULL,
    source_tx_hash TEXT NOT NULL, -- Blockchain transaction hash
    source_address TEXT NOT NULL,

    settled_currency TEXT NOT NULL CHECK (settled_currency IN ('USDT')),
    settled_amount_minor BIGINT NOT NULL,

    payment_provider TEXT NOT NULL CHECK(payment_provider IN ('TODO')),
    provider_transaction_id TEXT NOT NULL, -- internal transaction id in provider's system

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE withdraws (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Transfer FROM user's liability account TO company custody cash account
    ledger_transfer_id UUID NOT NULL REFERENCES ledger_transfers(id),

    destination_currency TEXT NOT NULL CHECK (destination_currency IN ('USDT', 'BTC', 'ETH', 'SOL')),
    destination_amount_minor BIGINT NOT NULL,
    destination_tx_hash TEXT NOT NULL, -- Blockchain transaction hash
    destination_address TEXT NOT NULL,

    status TEXT NOT NULL CHECK (status IN ('progressing', 'finished', 'cancelled'))
);

-----------------------------------------------------------------------------------------------------------


CREATE TABLE chat_rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    label TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE
);

-- Only records user messages. System messages are not persisted
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), -- allows us to differ saving messages to DB, while still having the soon to be primary key
    user_id UUID NOT NULL REFERENCES users(id),
    chat_id UUID NOT NULL REFERENCES chat_rooms(id),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    is_deleted BOOLEAN NOT NULL DEFAULT false,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX chat_messages_user ON chat_messages(user_id);
CREATE INDEX idx_chat_messages_room_time ON chat_messages(chat_id, created_at DESC);

-----------------------------------------------------------------------------------------------------------

CREATE TABLE markets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'opened', 'paused', 'settling', 'resolved', 'cancelled')),
    
    house_ledger_account_id UUID NOT NULL REFERENCES ledger_accounts(id),
    q0_seeding BIGINT NOT NULL, -- seeding (= nb initial shares for each outcome) when market opens
    alpha_ppm BIGINT NOT NULL, -- TODO SWITCH TO NUMERIC IF NECESSARY // alpha applied in parts per million (= e-6) 
    fee_ppm BIGINT NOT NULL, -- fee applied to each bet in ppm, ie fee_ppm = 500000 <-> 0.5000000 <-> 5%
    volume_cents BIGINT NOT NULL DEFAULT 0,
    volume_24h BIGINT NOT NULL DEFAULT 0,

    outcome_sort TEXT NOT NULL CHECK (outcome_sort IN ('price', 'position')),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    close_time TIMESTAMPTZ,
    version BIGINT NOT NULL DEFAULT 0 -- for concurrent updates
);

CREATE INDEX idx_markets_search ON markets USING GIN (to_tsvector('simple', name || ' ' || description ));
CREATE INDEX idx_markets_status_close_time ON markets(status, close_time);
CREATE INDEX idx_markets_close_time ON markets (close_time ASC);
CREATE INDEX idx_markets_volume_cents ON markets (volume_cents DESC);
CREATE INDEX idx_markets_created_at ON markets (created_at DESC);
CREATE INDEX idx_markets_volume_24h ON markets (volume_24h DESC);


-- Prevent re-opening an already settled market
CREATE OR REPLACE FUNCTION prevent_reopen_final_status() RETURNS TRIGGER AS $$
BEGIN
  IF OLD.status IN ('resolved','cancelled') AND NEW.status <> OLD.status THEN
    RAISE EXCEPTION 'market % status % is final, cannot change to status %', OLD.id, OLD.status, NEW.status;
  END IF;
  RETURN NEW;
END; 
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_markets_final_status
BEFORE UPDATE ON markets
FOR EACH ROW
EXECUTE FUNCTION prevent_reopen_final_status();

CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT UNIQUE,
    position BIGINT NOT NULL,
    label TEXT NOT NULL,
    CONSTRAINT categories_position_unique UNIQUE(position)
);

CREATE TABLE categories_market (
    id BIGSERIAL PRIMARY KEY, 
    market_id UUID NOT NULL REFERENCES markets(id),
    category_id BIGINT NOT NULL REFERENCES categories(id),
    CONSTRAINT categories_market_unique UNIQUE (market_id, category_id)
);

CREATE TABLE outcomes (
    id BIGSERIAL PRIMARY KEY,
    market_id UUID NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    position BIGINT NOT NULL,
    quantity BIGINT NOT NULL, -- number of shares bought of this outcome
    volume_cents BIGINT NOT NULL DEFAULT 0, -- how much money was put in this outcome
    CONSTRAINT outcomes_market_name_unique UNIQUE (market_id, name),
    CONSTRAINT outcomes_market_position_unique UNIQUE (market_id, position),
    CONSTRAINT outcomes_positive_qty CHECK (quantity >= 0) 
);

CREATE INDEX idx_outcomes_market_id ON outcomes(market_id);


CREATE TABLE market_resolutions (
    id BIGSERIAL PRIMARY KEY,
    market_id UUID NOT NULL REFERENCES markets(id),
    winning_outcome_id BIGINT NOT NULL REFERENCES outcomes(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT market_resolutions_market_unique UNIQUE(market_id)
);

CREATE TABLE market_cancellations (
  id BIGSERIAL PRIMARY KEY,
  market_id UUID NOT NULL REFERENCES markets(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (market_id)
);

CREATE INDEX idx_market_resolutions_winning_outcome ON market_resolutions(winning_outcome_id);


CREATE TABLE bets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ledger_account_id UUID NOT NULL REFERENCES ledger_accounts(id),
    outcome_id BIGINT NOT NULL REFERENCES outcomes(id),
    payout_cents BIGINT NOT NULL, -- number of shares bought, 1 share = payout of 1 cent, gives the payout
    total_price_paid_cents BIGINT NOT NULL, -- price paid to buy the shares (includes the fee applied)
    fee_paid_cents BIGINT NOT NULL, -- fee deduced from the price paid to calculate payout
    fee_ppm BIGINT NOT NULL, -- fee in percentage, applied to the input price
    price_ppm BIGINT NOT NULL, -- avg price in ppm (parts per million) at which the shares were bought
    idempotency_key TEXT NOT NULL,
    placed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT bet_idempotency_key_unique UNIQUE (idempotency_key),
    CONSTRAINT bets_positive_payout CHECK (payout_cents >= 0) 
);

CREATE INDEX idx_bet_outcome_ledger_account ON bets(outcome_id, ledger_account_id);
CREATE INDEX idx_bets_ledger_account_placed_at ON bets(ledger_account_id, placed_at DESC);
CREATE INDEX idx_bets_outcome_placed_at ON bets(outcome_id, placed_at DESC);

-----------------------------------------------------------------------------------------------------------

CREATE TABLE outcome_price_history (
    id BIGSERIAL PRIMARY KEY,
    outcome_id BIGINT NOT NULL REFERENCES outcomes(id),
    price_ppm BIGINT NOT NULL,
    time_recorded TIMESTAMPTZ NOT NULL DEFAULT now()
);


-----------------------------------------------------------------------------------------------------------

CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    type TEXT NOT NULL CHECK (type IN ('bet_won')),
    data JSONB NOT NULL, -- additional data depending on the notification type
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_is_read ON notifications(user_id, is_read, created_at DESC);




-----------------------------------------------------------------------------------------------------------

CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    market_id UUID NOT NULL REFERENCES markets(id),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    is_deleted BOOLEAN NOT NULL DEFAULT false,
    deleted_at TIMESTAMPTZ,

    parent_id BIGINT REFERENCES comments(id),
    depth BIGINT NOT NULL
);

-----------------------------------------------------------------------------------------------------------

CREATE TABLE mutes (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    reason TEXT NOT NULL,
    effective_until TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_mutes_user_effective_until ON mutes(user_id, effective_until DESC);

CREATE TABLE mute_chat_messages (
    id BIGSERIAL PRIMARY KEY,
    chat_message_id UUID NOT NULL REFERENCES chat_messages(id),
    mute_id BIGINT NOT NULL REFERENCES mutes(id),
    CONSTRAINT mute_messages_unique UNIQUE (chat_message_id, mute_id)
);

CREATE INDEX idx_mute_chat_messages_mute ON mute_chat_messages(mute_id);

CREATE TABLE mute_comments (
    id BIGSERIAL PRIMARY KEY,
    comment_id BIGINT NOT NULL REFERENCES comments(id),
    mute_id BIGINT NOT NULL REFERENCES mutes(id),
    CONSTRAINT mute_comments_unique UNIQUE (comment_id, mute_id)
);

CREATE INDEX idx_mute_comments_mute ON mute_comments(mute_id);

CREATE TABLE report_comments (
    id BIGSERIAL PRIMARY KEY,
    reporter_user_id UUID NOT NULL REFERENCES users(id),
    comment_id BIGINT NOT NULL REFERENCES comments(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT report_comments_unique UNIQUE (reporter_user_id, comment_id)
);

CREATE INDEX idx_report_comments_comment ON report_comments(comment_id);


-----------------------------------------------------------------------------------------------------------

CREATE TABLE bonus_offers (
    id BIGSERIAL PRIMARY KEY,
    match_percent BIGINT NOT NULL, -- bonus percentage compared to depo, if 100 -> bonus is 100% of depo (= x2)
    max_bonus_cents BIGINT NOT NULL, -- limit/cap on the claimable bonus amount
    fee_base_ppm BIGINT NOT NULL, -- fee that was used as a base to determine the wagering target, useful if needs weights later (avoid to simplify)
    wagering_base TEXT NOT NULL CHECK (wagering_base IN ('deposit_plus_bonus', 'bonus')), -- base on which wager requirement is computed
    wagering_times BIGINT NOT NULL, -- Multiplicator of base amount to compute waging requirement
    max_bet_cents BIGINT NOT NULL, -- maximum bet amount per market to qualify for completing wagering requirement
    expiry_interval INTERVAL, -- optionnal, give a max time frame (beginning when activated) to spend the bonus 
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_bonus (
    id BIGSERIAL PRIMARY KEY,
    bonus_offer_id BIGINT NOT NULL REFERENCES bonus_offers(id),
    user_id UUID NOT NULL REFERENCES users(id),
    deposit_id UUID NOT NULL REFERENCES deposits(id),
    status TEXT NOT NULL CHECK(status IN ('active', 'completed', 'expired')),
    bonus_granted_cents BIGINT NOT NULL CHECK (bonus_granted_cents >= 0),
    wagering_target_cents BIGINT NOT NULL CHECK (wagering_target_cents >= 0),
    wagering_progress_cents BIGINT NOT NULL DEFAULT 0 CHECK (wagering_progress_cents >= 0),
    expires_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT user_deposit_unique UNIQUE (user_id, deposit_id)
);

CREATE TABLE bonus_wager (
    id BIGSERIAL PRIMARY KEY,
    user_bonus_id BIGINT NOT NULL REFERENCES user_bonus(id),
    bet_id UUID NOT NULL REFERENCES bets(id),
    contrib_wagering_target_cents BIGINT NOT NULL,
    CONSTRAINT user_bonus_bet_unique UNIQUE (user_bonus_id, bet_id)
);

CREATE TABLE bonus_wager_outcome ( -- Defines how much 
    id BIGSERIAL PRIMARY KEY
);