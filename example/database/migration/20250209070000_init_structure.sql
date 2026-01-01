-- +goose Up
-- +goose StatementBegin

CREATE TABLE users
(
    id            BIGSERIAL PRIMARY KEY,
    oid           VARCHAR(255) NOT NULL UNIQUE,
    username      VARCHAR(255) NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    totp_secret   VARCHAR(255) NULL,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    metadata      JSONB        NOT NULL,
    created_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE profiles
(
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT REFERENCES users (id) ON DELETE CASCADE NOT NULL,
    handle              VARCHAR(255)                                   NOT NULL UNIQUE,
    display_name        VARCHAR(255)                                   NOT NULL,
    bio                 TEXT                                           NULL,
    profile_picture_url VARCHAR(255)                                   NULL,
    is_public           BOOLEAN                                        NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMP                                      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP                                      NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE follows
(
    follower_profile_id BIGINT REFERENCES profiles (id) ON DELETE CASCADE NOT NULL,
    followed_profile_id BIGINT REFERENCES profiles (id) ON DELETE CASCADE NOT NULL,
    approved_at         TIMESTAMP                                         NULL,
    created_at          TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_profile_id, followed_profile_id)
);

CREATE TABLE posts
(
    id          BIGSERIAL PRIMARY KEY,
    profile_id  BIGINT REFERENCES profiles (id) ON DELETE CASCADE NOT NULL,
    caption     TEXT                                              NULL,
    visit_count INTEGER                                           NOT NULL DEFAULT 0,
    visibility  VARCHAR(30)                                       NOT NULL DEFAULT 'PUBLIC',
    created_at  TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE medias
(
    id         BIGSERIAL PRIMARY KEY,
    post_id    BIGINT REFERENCES posts (id) ON DELETE CASCADE NOT NULL,
    media_type VARCHAR(255)                                   NOT NULL,
    ordering   INTEGER                                        NOT NULL DEFAULT 0,
    created_at TIMESTAMP                                      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP                                      NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE comments
(
    id                BIGSERIAL PRIMARY KEY,
    post_id           BIGINT REFERENCES posts (id) ON DELETE CASCADE    NOT NULL,
    profile_id        BIGINT REFERENCES profiles (id) ON DELETE CASCADE NOT NULL,
    parent_comment_id BIGINT REFERENCES comments (id) ON DELETE CASCADE NULL,
    content           TEXT                                              NOT NULL,
    created_at        TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE post_reactions
(
    post_id       BIGINT REFERENCES posts (id) ON DELETE CASCADE    NOT NULL,
    profile_id    BIGINT REFERENCES profiles (id) ON DELETE CASCADE NOT NULL,
    reaction_name VARCHAR(255)                                      NOT NULL,
    created_at    TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (post_id, profile_id)
);

CREATE TABLE comment_reactions
(
    comment_id    BIGINT REFERENCES comments (id) ON DELETE CASCADE NOT NULL,
    profile_id    BIGINT REFERENCES profiles (id) ON DELETE CASCADE NOT NULL,
    reaction_name VARCHAR(255)                                      NOT NULL,
    created_at    TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP                                         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (comment_id, profile_id)
);


-- * auto-update function for updated_at timestamps
CREATE OR REPLACE FUNCTION auto_updated_at()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- * triggers to automatically update updated_at on updates
CREATE TRIGGER auto_updated_at_users
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

CREATE TRIGGER auto_updated_at_profiles
    BEFORE UPDATE
    ON profiles
    FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

CREATE TRIGGER auto_updated_at_follows
    BEFORE UPDATE
    ON follows
    FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

CREATE TRIGGER auto_updated_at_posts
    BEFORE UPDATE
    ON posts
    FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

CREATE TRIGGER auto_updated_at_comments
    BEFORE UPDATE
    ON comments
    FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

CREATE TRIGGER auto_updated_at_post_reactions
    BEFORE UPDATE
    ON post_reactions
    FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

CREATE TRIGGER auto_updated_at_comment_reactions
    BEFORE UPDATE
    ON comment_reactions
    FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS comment_reactions;
DROP TABLE IF EXISTS post_reactions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS media;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS follows;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS auto_updated_at();

-- +goose StatementEnd