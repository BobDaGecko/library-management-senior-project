.PHONY: templ test-db test-fetch seed
run: templ
	go run main.go

# Wipe + repopulate senior-library.db with demo data. First run fetches book
# metadata from Google Books (cached to cmd/seed/cache); later runs are offline.
seed:
	rm -f senior-library.db
	go run ./cmd/seed -gapi-token "$(shell grep -oP '(?<=")[^"]+(?=")' .env)"

templ:
	templ generate -path .

test: templ test-db test-fetch

test-db:
	go test ./db

test-fetch:
	go test ./fetch
