# API tests

Run commands from `api/` with the Go version in `go.mod`.

```sh
go vet ./...
go test ./...                         # offline, no database or .env.test
go test -tags db ./...                # offline fixtures plus Postgres tests
go test -tags live ./src/sources/...  # real-site change checks
```

The first dependency download requires network access; test execution only uses
local `httptest` servers. Source fixtures live in each source's `testdata/`.
They are small synthetic HTML/JSON examples of the response structures consumed
by the parsers, not downloaded snapshots. `{{BASE}}` is replaced by the local
server URL. MangaPlus's saved protobuf JSON is encoded to protobuf by its test
server so the real binary response decoder is exercised. Exact expectations
belong in fixture tests; live checks assert metadata, URLs and decodable covers
without pinning titles, chapter numbers, dates or cover URLs.

For database tests, start a disposable Postgres and export:

```sh
export POSTGRES_HOST=localhost POSTGRES_PORT=5432
export POSTGRES_DB=mantium_test POSTGRES_USER=postgres POSTGRES_PASSWORD=postgres
go test -tags db ./...
```

The role needs `CREATEDB`. Each database test creates a uniquely named database,
initializes the schema with the application's migrations, and drops it on cleanup.
No `.env.test` is needed. Do not run these tests in parallel within a package:
they temporarily change the process's database environment and connection pool.

Live source tests do not require Postgres or `.env.test`. They intentionally fail
on connectivity, anti-bot, removed-title or site-format changes. Review such
failures before updating fixtures. Browser-based custom selector checks can be
run with `go test -tags live ./src/manga -run TestCustomMangaLastReleasedChapter`
and require Chromium. The existing external-service integration checks are also
behind `live`: `go test -tags live ./src/integrations/...`. Those load the root
`.env.test`, require the configured Kaizoku/Tranga/Suwayomi/ntfy services, and may
add downloads or send notifications. They are not part of the source-only command
or CI.

GitHub Actions runs vet, offline tests, and the database tier against Postgres 16
on pushes and pull requests. The image publish workflow is separate.
