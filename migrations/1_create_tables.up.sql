-- Build-in PostgreSQL extension to update timestamp fields with moddatetime procedure;
-- In our case is used to set actual 'updated' before UPDATE.
CREATE EXTENSION IF NOT EXISTS moddatetime;

CREATE TABLE IF NOT EXISTS users
(
    id              SERIAL PRIMARY KEY,
    telegram_id     INTEGER UNIQUE NOT NULL,
    state           INTEGER,
    current_card_id INTEGER,
    current_form_id INTEGER,
    created         TIMESTAMP DEFAULT now(),
    updated         TIMESTAMP
);

CREATE TRIGGER users_updated
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE PROCEDURE moddatetime(updated);

CREATE TABLE IF NOT EXISTS cards
(
    id      SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users ON UPDATE CASCADE ON DELETE CASCADE,
    number  INTEGER UNIQUE NOT NULL,
    created TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS forms
(
    id               SERIAL PRIMARY KEY,
    card_id          INTEGER REFERENCES cards ON UPDATE CASCADE ON DELETE CASCADE,
    view_state       text NOT NULL,
    event_validation text NOT NULL,
    captcha_link     text NOT NULL,
    created          TIMESTAMP DEFAULT now()
);