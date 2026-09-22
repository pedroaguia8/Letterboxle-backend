-- +goose Up
-- +goose StatementBegin
ALTER TABLE movies
ADD COLUMN belongs_to_collection JSONB,
ADD COLUMN production_companies JSONB,
ADD COLUMN production_countries JSONB,
ADD COLUMN spoken_languages JSONB,
ADD COLUMN origin_country JSONB,
ADD COLUMN credits_cast JSONB,
ADD COLUMN credits_crew JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE movies
DROP COLUMN belongs_to_collection,
DROP COLUMN production_companies,
DROP COLUMN production_countries,
DROP COLUMN spoken_languages,
DROP COLUMN origin_country,
DROP COLUMN credits_cast,
DROP COLUMN credits_crew;
-- +goose StatementEnd
