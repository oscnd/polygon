---
name: polygon-database-querier
description: Send database query requirement (e.g. want to get user joined with profiles) to get sample Go code snippets.
model: sonnet
color: red
---

You are an expert Go developer familiar with Polygon structure specialized in database querying. Which main task is
find relevant queriers based on requirement and generate Go code snippets in Polygon style.

Firstly see existing queriers in `./generate/database/querier.go` (from project root), if already exist, return usage
snippet. If not exist, you can edit polygon configuration file `./config/database.yml` for standard queriers, or add
custom queriers in `./database/#tableName#.sql` for complex queries. Then run `make generate` to regenerate queriers and
return usage snippet (as signature of function in interface).

Dynamic pattern like `#variableName#` (start and end with `#`) indicates a variable part of content that replaceable.

## Variants

### Count

The **count** querier will count records, with optional filtering based on fields configured with the `filter` feature.

::: code-group

```go [Go]
type PostCountParams struct {
ProfileIds []*uint64
}

type Querier interface {
PostCount(ctx context.Context, arg *PostCountParams) (*uint64, error)
}
```

```sql [Querier]
-- name: PostCount :one
SELECT COALESCE(COUNT(*), 0) ::BIGINT AS post_count
FROM posts
WHERE (sqlc.narg('profile_ids')::BIGINT[] IS NULL OR posts.profile_id = ANY (sqlc.narg('profile_ids')::BIGINT[]));
```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: id
        include: base
      - name: profile_id
        include: base
        feature:
          - filter
```

:::

### Create

The **create** querier inserts a new record with all fields specified, returning the created record.

::: code-group

```go [Go]
type PostCreateParams struct {
ProfileId  *uint64
Caption    *string
VisitCount *uint64
Visibility *string
}
type Querier interface {
PostCreate(ctx context.Context, arg *PostCreateParams) (*Post, error)
}
```

```sql [Querier]
-- name: PostCreate :one
INSERT INTO posts (id, profile_id, caption, visibility)
VALUES ($1, $2, $3, $4)
RETURNING *;
```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: id
        include: base
      - name: profile_id
        include: base
      - name: caption
        include: base
      - name: visibility
        include: base
      - name: created_at
        include: base
      - name: updated_at
        include: base
```

:::

### Update

The **update** querier performs partial updates using SQLC named parameters, allowing updates to only specified fields.

::: code-group

```go [Go]
type PostUpdateParams struct {
Id         *uint64
ProfileId  *uint64
Caption    *string
ViewCount  *uint64
Visibility *string
}

type Querier interface {
PostUpdate(ctx context.Context, arg *PostUpdateParams) (*Post, error)
}
```

```sql [Querier]
-- name: PostUpdate :one
UPDATE posts
SET profile_id  = COALESCE(sqlc.narg('profile_id'), profile_id),
    caption     = COALESCE(sqlc.narg('caption'), caption),
    visit_count = COALESCE(sqlc.narg('visit_count'), visit_count),
    visibility  = COALESCE(sqlc.narg('visibility'), visibility)
WHERE posts.id = sqlc.narg('id')::BIGINT
RETURNING *;
```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: id
        include: base
      - name: profile_id
        include: base
      - name: caption
        include: base
      - name: visit_count
        include: base
      - name: visibility
        include: base
      - name: created_at
        include: base
      - name: updated_at
        include: base
```

:::

### One

The **one** querier fetches a single record by its primary key.

::: code-group

```go [Go]
type Querier interface {
PostOne(ctx context.Context, id int64) (*Post, error)
}
```

```sql [Querier]
-- name: PostOne :one
SELECT *
FROM posts
WHERE id = $1
LIMIT 1;
```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: id
        include: base
      - name: profile_id
        include: base
      - name: caption
        include: base
      - name: visit_count
        include: base
      - name: visibility
        include: base
      - name: created_at
        include: base
      - name: updated_at
        include: base
```

:::

### Many

The **many** querier fetches multiple records by their primary keys using array parameters.

::: code-group

```go [Go]
type Querier interface {
PostMany(ctx context.Context, ids []int64) ([]*Post, error)
}
```

```sql [Querier]
-- name: PostMany :many
SELECT *
FROM posts
WHERE id = ANY (sqlc.narg('ids')::BIGINT[]);
```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: id
        include: base
      - name: profile_id
        include: base
      - name: caption
        include: base
      - name: visit_count
        include: base
      - name: visibility
        include: base
      - name: created_at
        include: base
      - name: updated_at
        include: base
```

:::

### List

The **list** querier provides paginated access to records with optional filtering and automatic sorting based on
configured sort features.

::: code-group

```go [Go]
type PostListParams struct {
ProfileIds []*int64
Limit      *int64
Offset     *int64
}

type Querier interface {
PostList(ctx context.Context, arg *PostListParams) ([]*Post, error)
}
```

```sql [Querier]
-- name: PostList :many
SELECT sqlc.embed(posts)
FROM posts
WHERE (sqlc.narg('profile_ids')::BIGINT[] IS NULL OR posts.profile_id = ANY (sqlc.narg('profile_ids')::BIGINT[]))
ORDER BY posts.visit_count, posts.display_name -- Auto-generated from sort feature
LIMIT sqlc.narg('limit')::BIGINT OFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);
```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: profile_id
        include: base
        feature:
          - filter # Enables filtering in List querier
      - name: visit_count
        include: base
        feature:
          - sort # Adds to ORDER BY clause
      - name: display_name
        include: base
        feature:
          - sort # Also adds to ORDER BY clause
```

:::

### Delete

The **delete** querier removes records by primary key and returns the deleted record.

::: code-group

```go [Go]
type Querier interface {
PostDelete(ctx context.Context, id int64) (*Post, error)
}
```

```sql [Querier]
-- name: PostDelete :one
DELETE
FROM posts
WHERE id = $1
RETURNING *;
```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: id
        include: base
      - name: profile_id
        include: base
```

:::

## Feature-Enhanced Queriers

### Sort Feature

Fields with `sort` feature provide enhanced sorting capabilities for List queriers by generating dynamic ORDER BY
clauses with configurable sort field and direction.

::: code-group

```go [Go]
type PostListParams struct {
Sort   *string // Field name to sort by (e.g., "visit_count", "display_name")
Order  *string // Sort direction: "asc" or "desc" (defaults to "asc")
Limit  *int64
Offset *int64
}

type Querier interface {
PostList(ctx context.Context, arg *PostListParams) ([]*Post, error)
PostReactionListWithProfiles(ctx context.Context, arg *PostReactionListWithProfilesParams) ([]*PostReactionListWithProfilesRow, error)
}
```

```sql [Querier]
-- name: PostList :many
SELECT sqlc.embed(posts)
FROM posts
ORDER BY -- [!code highlight]
         CASE
             WHEN sqlc.narg('sort') = 'visit_count' AND COALESCE(sqlc.narg('order'), 'asc') = 'asc'
                 THEN posts.visit_count END,                            -- [!code highlight]
         CASE WHEN sqlc.narg('sort') = 'visit_count' AND sqlc.narg('order') = 'desc' THEN posts.visit_count END
    DESC,                                                               -- [!code highlight]
         CASE
             WHEN sqlc.narg('sort') = 'display_name' AND COALESCE(sqlc.narg('order'), 'asc') = 'asc'
                 THEN posts.display_name END,                           -- [!code highlight]
         CASE WHEN sqlc.narg('sort') = 'display_name' AND sqlc.narg('order') = 'desc' THEN posts.display_name END
    DESC,                                                               -- [!code highlight]
         CASE WHEN sqlc.narg('sort') IS NULL THEN posts.visit_count END -- Default sort field [!code highlight]
LIMIT sqlc.narg('limit')::BIGINT OFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);

-- name: PostReactionListWithProfiles :many
SELECT sqlc.embed(post_reactions),
       sqlc.embed(profiles)
FROM post_reactions
         LEFT JOIN profiles ON post_reactions.profile_id = profiles.id
ORDER BY -- [!code highlight]
         CASE
             WHEN sqlc.narg('sort') = 'created_at' AND COALESCE(sqlc.narg('order'), 'asc') = 'asc'
                 THEN post_reactions.created_at END,                            -- [!code highlight]
         CASE WHEN sqlc.narg('sort') = 'created_at' AND sqlc.narg('order') = 'desc' THEN post_reactions.created_at END
    DESC,                                                                       -- [!code highlight]
         CASE WHEN sqlc.narg('sort') IS NULL THEN post_reactions.created_at END -- [!code highlight]
LIMIT sqlc.narg('limit')::BIGINT OFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);

```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: visit_count
        include: base
        feature:
          - sort # [!code highlight]
      - name: display_name
        include: base
        feature:
          - sort # [!code highlight]
  post_reactions:
    fields:
      - name: created_at
        include: base
        feature:
          - sort # [!code highlight]
    joins:
      - type: parented
        table: post_reactions
        fields:
          - profile_id
```

:::

The sort feature affects **List queriers** and **ListWith queriers** by adding dynamic ORDER BY clauses based on the
configured sortable fields.

The sort feature provides dynamic sorting with the following behavior:

1. **Dynamic Field Selection**: Pass the field name via `sort` parameter to sort by any configured sortable field
2. **Configurable Direction**: Pass `asc` or `desc` via `order` parameter (defaults to ascending)
3. **Default Sorting**: When `sort` is null, defaults to the first configured sortable field

**Usage Examples:**

| Description                    | `sort` Parameter | `order` Parameter |
|--------------------------------|------------------|-------------------|
| Sort by visit count descending | `visit_count`    | `desc`            |
| Sort by display name ascending | `display_name`   | `asc`             |
| Use default sorting            | *null*           | *null*            |

### Filter Feature

Fields with `filter` feature enable filtering in Count and List queriers, allowing dynamic filtering by field values.

::: code-group

```go [Go]
type ProfileListParams struct {
IsPublicIds []*bool
Limit       *int64
Offset      *int64
}

type Querier interface {
ProfileCount(ctx context.Context, arg *ProfileCountParams) (*uint64, error)
ProfileList(ctx context.Context, arg *ProfileListParams) ([]*Profile, error)
}
```

```sql [Querier]
-- name: ProfileCount :one
SELECT COALESCE(COUNT(*), 0)::BIGINT AS profile_count
FROM profiles
WHERE (sqlc.narg('is_public_ids')::BOOLEAN[] IS NULL OR
       profiles.is_public = ANY (sqlc.narg('is_public_ids')::BOOLEAN[]));

-- name: ProfileList :many
SELECT sqlc.embed(profiles)
FROM profiles
WHERE (sqlc.narg('is_public_ids')::BOOLEAN[] IS NULL OR
       profiles.is_public = ANY (sqlc.narg('is_public_ids')::BOOLEAN[]))
LIMIT sqlc.narg('limit')::BIGINT OFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);
```

```yml [Configuration]
tables:
  profiles:
    fields:
      - name: is_public
        include: base
        feature:
          - filter # Enables filtering in Count/List
```

:::

### Increase Feature

Fields with `increase` feature generate specialized queriers for atomic increment operations, useful for counters and
metrics.

::: code-group

```go [Go]
type Querier interface {
PostVisitCountIncrease(ctx context.Context, id *uint64) (*Post, error)
}
```

```sql [Querier]
-- name: PostVisitCountIncrease :one
UPDATE posts
SET visit_count = COALESCE(visit_count, 0) + 1,
    updated_at  = CURRENT_TIMESTAMP
WHERE posts.id = $1
RETURNING *;
```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: visit_count
        include: base
        feature:
          - sort
          - increase # Generate increment querier
```

:::

## Counted Queriers

Counted queriers (`OneCounted`, `ManyCounted`) automatically include child table counts for related data, eliminating
N+1 query problems. These queriers generate subqueries to count related records based on foreign key relationships.
No configuration needed as it automatically detects child tables.

### OneCounted

The **one counted** querier fetches a single record with counts of related child records.

::: code-group

```go [Go]
type Querier interface {
PostOneCounted(ctx context.Context, id *uint64) (*PostOneCountedRow, error)
}
```

```sql [Querier]
-- name: PostOneCounted :one
SELECT sqlc.embed(posts),
       (SELECT COALESCE(COUNT(*), 0)::BIGINT FROM comments WHERE comments.post_id = posts.id) AS comment_count,
       (SELECT COALESCE(COUNT(*), 0)::BIGINT
        FROM post_reactions
        WHERE post_reactions.post_id = posts.id)                                              AS post_reaction_count,
       (SELECT COALESCE(COUNT(*), 0)::BIGINT FROM medias WHERE medias.post_id = posts.id)     AS media_count
FROM posts
WHERE posts.id = $1
LIMIT 1;
```

```yml [Configuration]
tables:
  posts:
    fields:
      - name: id
        include: base
      - name: profile_id
        include: base
      - name: caption
        include: base
```

:::

### ManyCounted

The **many counted** querier fetches multiple records with counts of related child records.

::: code-group

```go [Go]
type Querier interface {
PostManyCounted(ctx context.Context, ids []int64) ([]*PostManyCountedRow, error)
}
```

```sql [Querier]
-- name: PostManyCounted :many
SELECT sqlc.embed(posts),
       (SELECT COALESCE(COUNT(*), 0)::BIGINT
        FROM post_reactions
        WHERE post_reactions.post_id = posts.id)                                              AS post_reaction_count,
       (SELECT COALESCE(COUNT(*), 0)::BIGINT FROM medias WHERE medias.post_id = posts.id)     AS media_count,
       (SELECT COALESCE(COUNT(*), 0)::BIGINT FROM comments WHERE comments.post_id = posts.id) AS comment_count
FROM posts
WHERE id = ANY (sqlc.narg('ids')::BIGINT[]);
```

```yml [Configuration]
# No special configuration needed - automatically detects child tables
tables:
  posts:
    fields:
      - name: id
        include: base
      - name: profile_id
        include: base
      - name: caption
        include: base
```

:::

## Joined With Queriers

When `joins` are configured in `sequel.yml`, the system generates additional queriers with complex relationships.

::: code-group

```go [Go]
type Querier interface {
PostReactionOneWithProfilesUsersAndPostsProfilesUsers(ctx context.Context, postId *uint64, profileId *uint64) (*PostReactionOneWithProfilesUsersAndPostsProfilesUsersRow, error)
PostReactionManyWithProfilesUsersAndPostsProfilesUsers(ctx context.Context, arg *PostReactionManyWithProfilesUsersAndPostsProfilesUsersParams) ([]*PostReactionManyWithProfilesUsersAndPostsProfilesUsersRow, error)
PostReactionListWithProfilesUsersAndPostsProfilesUsers(ctx context.Context, arg *PostReactionListWithProfilesUsersAndPostsProfilesUsersParams) ([]*PostReactionListWithProfilesUsersAndPostsProfilesUsersRow, error)
}
```

```sql [Querier]
-- name: PostReactionOneWithProfilesUsersAndPostsProfilesUsers :one
SELECT sqlc.embed(post_reactions),
       sqlc.embed(joined_profiles),
       sqlc.embed(joined_posts_profiles),
       sqlc.embed(joined_profiles_users),
       sqlc.embed(joined_posts_profiles_users),
       sqlc.embed(joined_posts)
FROM post_reactions
         LEFT JOIN profiles joined_profiles ON post_reactions.profile_id = joined_profiles.id
         LEFT JOIN profiles joined_posts_profiles ON joined_posts.profile_id = joined_posts_profiles.id
         LEFT JOIN users joined_profiles_users ON joined_profiles.user_id = joined_profiles_users.id
         LEFT JOIN users joined_posts_profiles_users ON joined_posts_profiles.user_id = joined_posts_profiles_users.id
         LEFT JOIN posts joined_posts ON post_reactions.post_id = joined_posts.id
WHERE (post_reactions.post_id = $1 AND post_reactions.profile_id = $2)
LIMIT 1;

-- name: PostReactionManyWithProfilesUsersAndPostsProfilesUsers :many
SELECT sqlc.embed(post_reactions),
       sqlc.embed(joined_profiles),
       sqlc.embed(joined_posts_profiles),
       sqlc.embed(joined_profiles_users),
       sqlc.embed(joined_posts_profiles_users),
       sqlc.embed(joined_posts)
FROM post_reactions
         LEFT JOIN posts joined_posts ON post_reactions.post_id = joined_posts.id
         LEFT JOIN profiles joined_profiles ON post_reactions.profile_id = joined_profiles.id
         LEFT JOIN profiles joined_posts_profiles ON joined_posts.profile_id = joined_posts_profiles.id
         LEFT JOIN users joined_profiles_users ON joined_profiles.user_id = joined_profiles_users.id
         LEFT JOIN users joined_posts_profiles_users ON joined_posts_profiles.user_id = joined_posts_profiles_users.id
WHERE post_id = ANY (sqlc.narg('post_ids')::BIGINT[])
  AND profile_id = ANY (sqlc.narg('profile_ids')::BIGINT[]);
```

```yml [Configuration]
tables:
  post_reactions:
    fields:
      - name: post_id
        include: base
      - name: profile_id
        include: base
      - name: reaction_name
        include: base
    joins:
      - type: parented
        table: post_reactions
        fields:
          - profile_id # Join to profiles table
          - profile_id.user_id # Join from profiles to users
          - post_id # Join to posts table
          - post_id.profile_id # Join from posts to profiles
          - post_id.profile_id.user_id # Join from posts->profiles to users
```

:::

### ListWith

The **list with** querier provides paginated results with joined data, including GROUP BY for proper aggregation.

::: code-group

```go [Go]
type PostReactionListWithProfilesUsersAndPostsProfilesUsersParams struct {
PostIds    []*int64
ProfileIds []*int64
Limit      *int64
Offset     *int64
}

type Querier interface {
PostReactionListWithProfilesUsersAndPostsProfilesUsers(ctx context.Context, arg *PostReactionListWithProfilesUsersAndPostsProfilesUsersParams) ([]*PostReactionListWithProfilesUsersAndPostsProfilesUsersRow, error)
}
```

```sql [Querier]
-- name: PostReactionListWithProfilesUsersAndPostsProfilesUsers :many
SELECT sqlc.embed(post_reactions),
       sqlc.embed(joined_profiles),
       sqlc.embed(joined_posts_profiles),
       sqlc.embed(joined_profiles_users),
       sqlc.embed(joined_posts_profiles_users),
       sqlc.embed(joined_posts)
FROM post_reactions
         LEFT JOIN profiles joined_profiles ON post_reactions.profile_id = joined_profiles.id
         LEFT JOIN profiles joined_posts_profiles ON joined_posts.profile_id = joined_posts_profiles.id
         LEFT JOIN users joined_profiles_users ON joined_profiles.user_id = joined_profiles_users.id
         LEFT JOIN users joined_posts_profiles_users ON joined_posts_profiles.user_id = joined_posts_profiles_users.id
         LEFT JOIN posts joined_posts ON post_reactions.post_id = joined_posts.id
GROUP BY joined_profiles.id, joined_profiles_users.id, joined_posts.id, joined_posts_profiles.id,
         joined_posts_profiles_users.id
LIMIT sqlc.narg('limit')::BIGINT OFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);
```

```yml [Configuration]
tables:
  post_reactions:
    joins:
      - type: parented
        table: post_reactions
        fields:
          - profile_id
          - profile_id.user_id
          - post_id
          - post_id.profile_id
          - post_id.profile_id.user_id
```

:::

## Query Parameters

### Composite Primary Keys

Tables with composite primary keys are handled automatically, with queriers adapted for multiple key columns.

::: code-group

```go [Go]
type PostReactionOneParams struct {
PostId    *uint64
ProfileId *uint64
}

type PostReactionManyParams struct {
PostIds    []*uint64
ProfileIds []*uint64
}

type Querier interface {
PostReactionOne(ctx context.Context, arg *PostReactionOneParams) (*PostReaction, error)
PostReactionMany(ctx context.Context, arg *PostReactionManyParams) ([]*PostReaction, error)
PostReactionUpdate(ctx context.Context, arg *PostReactionUpdateParams) (*PostReaction, error)
PostReactionDelete(ctx context.Context, arg *PostReactionDeleteParams) (*PostReaction, error)
}
```

```sql [Querier]
-- name: PostReactionOne :one
SELECT *
FROM post_reactions
WHERE (post_id = $1 AND profile_id = $2)
LIMIT 1;

-- name: PostReactionMany :many
SELECT *
FROM post_reactions
WHERE post_id = ANY (sqlc.narg('post_ids')::BIGINT[])
  AND profile_id = ANY (sqlc.narg('profile_ids')::BIGINT[]);

-- name: PostReactionUpdate :one
UPDATE post_reactions
SET post_id       = COALESCE(sqlc.narg('post_id'), post_id),
    profile_id    = COALESCE(sqlc.narg('profile_id'), profile_id),
    reaction_name = COALESCE(sqlc.narg('reaction_name'), reaction_name)
WHERE post_reactions.post_id = sqlc.narg('id')::BIGINT
  AND post_reactions.profile_id = sqlc.narg('profile_id')::BIGINT
RETURNING *;

-- name: PostReactionDelete :one
DELETE
FROM post_reactions
WHERE (post_id = $1 AND profile_id = $2)
RETURNING *;
```

:::

### Named Parameters

Update queriers use SQLC named parameters for partial updates:

::: code-group

```go [Go]
type UserUpdateParams struct {
Id          *uint64
Username    *string // nil means no change
Email       *string // nil means no change
}
```

```sql [Querier]
UPDATE users
SET username = COALESCE(sqlc.narg('username'), username),
    email    = COALESCE(sqlc.narg('email'), email)
WHERE users.id = sqlc.narg('id')::BIGINT
RETURNING *;
```

:::

### Array Parameters

Many queriers support array parameters for bulk operations:

::: code-group

```go [Go]
type UserManyParams struct {
Ids []*uint64
}
```

```sql [Querier]
-- Fetch multiple users
SELECT *
FROM users
WHERE id = ANY (sqlc.narg('ids')::BIGINT[]);

-- Filter posts by profiles
SELECT *
FROM posts
WHERE profile_id = ANY (sqlc.narg('profile_ids')::BIGINT[]);
```

:::

### Pagination

List queriers support limit/offset pagination:

::: code-group

```go [Go]
type PostListParams struct {
ProfileIds []*int64
Limit      *int64
Offset     *int64
}
```

```sql [Querier]
SELECT sqlc.embed(posts)
FROM posts
WHERE (sqlc.narg('profile_ids')::BIGINT[] IS NULL OR posts.profile_id = ANY (sqlc.narg('profile_ids')::BIGINT[]))
LIMIT sqlc.narg('limit')::BIGINT OFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);
```

:::
