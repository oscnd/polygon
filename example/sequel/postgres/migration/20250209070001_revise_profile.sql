-- +goose Up
-- +goose StatementBegin

-- remove profiles.profile_picture_url
ALTER TABLE profiles
DROP COLUMN IF EXISTS profile_picture_url;

-- Change profiles.bio from TEXT to VARCHAR(255)
ALTER TABLE profiles
ALTER COLUMN bio TYPE VARCHAR(255);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- add profiles.profile_picture_url back
ALTER TABLE profiles
ADD COLUMN profile_picture_url VARCHAR(255) NULL;

-- revert profiles.bio back to TEXT
ALTER TABLE profiles
ALTER COLUMN bio TYPE TEXT;

-- +goose StatementEnd