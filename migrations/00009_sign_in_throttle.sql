-- +goose Up
-- Failed sign-in attempts are counted per email, whether or not an account
-- exists, so the "N attempts left" message never reveals which emails are
-- registered. users.failed_attempts is kept for the instance admin's view.
CREATE TABLE sign_in_throttle (
    email        text        PRIMARY KEY CHECK (email = lower(email)),
    failures     integer     NOT NULL DEFAULT 0,
    locked_until timestamptz,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE sign_in_throttle;
