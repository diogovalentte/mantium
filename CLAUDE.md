# Mantium — context and pending work

This file is the working spec for modernizing this repo. It's not a description
of how Mantium works (the [README](./README.md) and
[integrations.md](./integrations.md) cover that) — it's the list of what has
aged badly, why, and in which order to touch it.

Written on 2026-08-09, after building
[valentium](https://github.com/diogovalentte/valentium) on the same stack (Go +
gin, Python + Streamlit) with the current versions of everything. Most items
below are things that were solved there and can be brought over.

## Bug audit, 2026-09-16

Separate from the modernization plan below: a pass looking for things that are
already broken, rather than things that have aged. Everything here was
reproduced before being written down, and everything in the first list is fixed
on `fix/auditoria-bugs`.

The two headline ones, verified against the published v6.3.3 binary running on
a throwaway Postgres:

```
UPDATE_MANGAS_JOB_PARALLEL_JOBS=4, 1 manga:
  PATCH /v1/mangas/metadata -> 500, panic recovered: slice bounds out of range [2:1]
GET /v1/mangas/iframe?limit=abc&api_url=...
  -> status 400 with 8832 bytes of iframe HTML appended to it
```

### Fixed

| | What was wrong |
| --- | --- |
| Update job panics | `chunkSize := (len+n-1)/n` then `items[start:end]` with no `start <= end` check. Any `UPDATE_MANGAS_JOB_PARALLEL_JOBS` above the item count panicked on every run; `0` divided by zero. Now `util.ChunkSlice`, with tests |
| Sources share mutable state | `Sources` holds one `*rawkuma.Source` for the whole process and it kept a `*colly.Collector`. Two parallel updates of the same source swap it between reset and visit, so one manga's page fires the other's callbacks — the wrong cover and chapter get written, silently. All seven sources are `struct{}` now |
| Every cover downloaded 3× | `GetImageFromURL`'s retry loop never broke on success. All 11 call sites pass `retries=3`. A success followed by a later failure was also returned as an error |
| `newMetadata` data race | Written from every worker goroutine, unsynchronised. Now carried on the result struct |
| iframe double response | Missing `return` after the 400 for a bad `limit` |
| Kaizoku errors swallowed | `addMangaToKaizoku` ended in `return nil` for everything except one message. That hid two more bugs: fallback sources came from a randomised map iteration, and `manga.Source` was mutated on the caller's manga — which is what Tranga and Suwayomi read afterwards |
| No outbound timeouts | All 9 HTTP clients and all 3 colly collectors. A hung source wedged the periodic update goroutine permanently and silently |
| Goroutine leak | `waitUntilEmptyCheckFixOutOfSyncChaptersQueues` raced a poller against `time.After` on an unbuffered channel; on timeout it leaked, kept hammering Kaizoku, then blocked on the send forever |
| Postgres password in logs | `OpenConn` put the whole connection string in the Ping error, which `init()` panics with |
| `rows.Err()` never checked | 7 loops returned a partial library as if it were complete |
| No FK index, plus an N+1 | `mangas(multimanga_id)` was unindexed (Postgres only indexes the referenced side), and `getMultiMangasWithMangasDB` queried per multimanga from inside the outer result set |
| 33 connection pools | Every call site did `OpenConn()` + `defer Close()`, and `OpenConn` did `sql.Open` + `Ping` each time |
| `SourcesList` duplicated | A second hand-maintained copy of the `Sources` keys in `config`. Now published by `sources` at init, pinned by a test. `ALLOWED_SOURCES` is trimmed too — `mangadex, mangahub` used to refuse to boot |
| Background warning off by one | `<=` meant the default of 5 showed on the 6th consecutive error |
| Silent startup failure | `router.Run`'s error was dropped, so a busy port exited with status 0 and no output |
| TLD update | Built with `Sprintf` instead of bind params, and rewrote every row on every boot even when nothing changed |
| Dashboard had no timeouts | 24 `requests` calls with no `timeout=`. A non-answering API froze the Streamlit worker thread, including the 5s polling fragment |
| Dashboard rerun loop | The global handler set a flag and called `st.rerun()`, but the code that rendered the error lived inside `show()`, which the failing path never reached |
| Dashboard tests never ran | `src/util/` had no `__init__.py`, so collection aborted with `ImportError`. `test_manga_api.py` is two majors stale and now carries an `integration` marker |
| Source fallback | `update_manga.py` excluded the loop's manga from the next attempt instead of the candidate the API actually returned |

The TLD regex `(https?://[^/]+?)\.[a-z]+` is **correct**, despite the lazy
quantifier looking wrong — Postgres's POSIX engine is leftmost-longest, not
Perl-style. Checked against a real database, including `www.` hosts and ports.

### Known and not fixed

- **SSRF.** `cover_img_url` on the cover routes, and the custom manga URL fed to
  colly and to rod, are fetched by the server with no check on the target. This
  is item 4 below, and the hole exists today, not only in the future proxy route.
- **Dependencies.** Per OSV: `pillow==10.4.0` has 17 advisories, `urllib3==2.5.0`
  has 4, `requests==2.32.4` has 1. `streamlit`, `certifi` and `Jinja2` are clean.
- **`urlToSource` matches with `strings.Contains`**, so `mangadex.attacker.com`
  resolves as mangadex.
- **`gofmt`**: `src/integrations/tranga/models.go` and `src/sources/mangaplus/api.go`.
- **A NULL `preferred_group` breaks `getCustomMangasFromDB`** with a scan error.
  The app always writes `''`, so this only bites a hand-edited row.
- **`go test ./...` still cannot run in CI**: it needs `.env.test`, a live
  Postgres and the internet. Items 8 and 9 below.

## State of the repo, in numbers

Measured on 2026-08-09, so re-check before trusting any of it.

| | |
| --- | --- |
| API | ~19,300 lines of Go, tests per source, a source registry that works |
| Dashboard | ~4,300 lines of Python, of which **1028** are `01_📖_Dashboard.py` |
| Streamlit | 1.54.0, with ~80 pinned packages in `requirements.txt` |
| CI | one workflow, publishes images on a tag. Nothing runs the tests |

The split matters: **the API is the mature half.** The work below is almost
entirely dashboard and infrastructure. Don't let "modernize Mantium" turn into
rewriting Go that isn't hurting anyone.

## Dashboard

### 1. The structure is two Streamlit generations old

`01_📖_Dashboard.py` is 1028 lines and sits at the repo root with the title and
icon encoded in the filename (`pages/01_📖_Name.py`, the pre-`st.Page`
convention). `src/pages/` exists and is **empty** — just `__init__.py`.
`src/util/update_manga.py` (1051 lines) and `src/util/add_manga.py` (750) are
UI, not utilities.

The target is `st.navigation` + `st.Page` with **pages as functions**: title,
icon and URL declared explicitly, and importing a page module doesn't execute
Streamlit code. See `dashboard/README.md` in valentium for the shape.

This is the expensive item. Do it last, on a codebase that's already clean.

### 2. `streamlit-extras` is what's really pinning the version

The requirements pull in `faker`, `matplotlib`, `altair`, `pyarrow`,
`streamlit-card`, `streamlit-toggle-switch`, `streamlit-vertical-slider`,
`streamlit-camera-input-live`, `streamlit-embedcode`,
`streamlit-image-coordinates`, `streamlit-keyup`, `streamlit-javascript`,
`streamlit-browser-engine` — a tree that exists to support **two** call sites:

- `stylable_container` (3 imports)
- `browser_detection_engine` (1 use)

Every third party component is something that can break on a Streamlit upgrade.
Removing this tree first is what turns the upgrade from a rewrite into a bump.

### 3. The CSS hacks have native equivalents now

29 `unsafe_allow_html` blocks across the dashboard (17 in the entrypoint alone).
The ones that are pure obsolescence:

| Hack | Replacement |
| --- | --- |
| CSS hiding `stHeaderActionElements` on headers | `anchor=False` — a real parameter now |
| `tagger()` in `util.py`, ~30 lines of hand written HTML | `st.badge` |
| `centered_container()` | `st.container(horizontal_alignment=...)` |

Not all of it is obsolete — the ellipsis on long manga names and the header
decorations have no native equivalent. Keep those, scoped.

### 4. The MangaDex cover workaround belongs in the API

Right now the dashboard injects JavaScript into the **parent** document to force
`<meta name="referrer" content="no-referrer">`, so MangaDex doesn't reject the
cover requests. There's also a `fix_streamlit_index_html()` in `util.py` that
rewrote `index.html` **inside the installed Streamlit package** — dead code, and
its own docstring says so ("DOING IT USING st_javascript INSTEAD").

Fix it where the problem is: **proxy the image through the API**
(`GET /v1/.../image?url=`), which strips the Referer because the request no
longer comes from a browser page. That deletes the JS, the dead function, and
works inside an iframe, where reaching into `window.parent` is a losing game.

Whatever route does this **must refuse non public addresses** — a route that
fetches a URL chosen by the caller is an SSRF primitive. Copy `util.CheckPublicURL`
from valentium.

### 5. `browser_detection_engine` for "is this a phone"

A whole package plus a hidden `st_javascript` iframe (hidden by CSS:
`div.st-key-browser_engine { display: none }`) to learn something the native
responsive layout mostly handles by itself — columns stack below 640px, and
`width="stretch"` covers the rest. Check what `ss["is_mobile"]` actually gates
before removing it; if it's only layout, it can go.

### 6. Almost nothing modern is in use

Counted across the dashboard: **1** `st.fragment`, **0** `st.cache_resource`,
16 `st.dialog` (these are used well), 22 `st.rerun`, 1 `st.badge`.

With one fragment, every interaction reruns all 1028 lines.

And `get_api_client()` is decorated with **`@st.cache_data()`**, which is the
wrong one: `cache_data` is for serializable *return values*, copied per caller;
a shared HTTP client is `cache_resource`. It works today by accident.

While in there, the rule that's easy to get wrong: `cache_data` is for
**idempotent reads only**. A cached call silently doesn't happen the second
time, which is exactly wrong for anything with side effects.

### 7. Dashboard tests

Two files, ~250 lines, and nothing drives the UI. `streamlit.testing.v1.AppTest`
can render a page and assert on what it produced. Two traps found the hard way
in valentium, both written up in its `CLAUDE.md`:

- `AppTest.from_function` re-executes the function source **without its
  imports**, so it can't run a decorated page. Use a small runner module +
  `AppTest.from_file`.
- `AppTest` often can't see `st.expander` (inside a fragment, or when it holds a
  form). Assert on the contents, not the container.

## API

### 8. Source tests hit the real internet with hardcoded expectations

`sources/mangadex/manga_test.go` expects "Death Note", chapter 108 named "End",
and an exact cover URL. So the suite needs network, and it goes red when a site
changes or a new chapter drops — the recent `fix: KLmanga TLD` and
`fix: rawkuma get manga cover` commits are the symptom.

The signal is genuinely valuable (it's how you learn a source changed), so don't
delete it — **split it**:

- parsing against saved HTML fixtures → offline, runs in `make test`
- connectivity against the live site → tagged (`-short` / a build tag), run on
  purpose or on a schedule

### 9. No CI runs the tests

The only workflow publishes images on a tag. Add a job that runs `go test ./...`
on push and PR. This is more valuable here than in valentium, because this repo
is public and takes contributions.

## Infrastructure

### 10. There is no `.dockerignore`

The build context is the repo root and `Dockerfile_dashboard` ends with
`COPY ./dashboard .` — so a local build copies `dashboard/.venv` (**751MB**,
built for the host) straight into the image. CI never hits it because the venv
is git-ignored, which is why it's gone unnoticed.

### 11. The publish workflow

Compared to what valentium ended up with:

- builds `linux/arm64` under QEMU on an amd64 runner, which dominates the wall
  clock. Drop it unless something actually consumes that image
- no build cache (`cache-from/to: type=gha`)
- publishes no `1.2` moving tag. `latest` *is* published — `docker/metadata-action`
  generates it for a semver tag through `flavor.latest=auto`, and the GHCR
  package has both `latest` and `v6.3.3`. An earlier draft of this file said
  otherwise; it was wrong
- requires a PAT in `secrets.GH_TOKEN`. The automatic `GITHUB_TOKEN` plus
  `permissions: packages: write` needs no setup
- the deploy webhook fails the job when its secrets are missing, *after* the
  images are already published

### 12. `docker-compose.yml` still declares `version: "3"`

Obsolete; Compose ignores it and warns.

## Suggested order

1. **`.dockerignore`** and **the workflow** (10, 11, 12) — no application code,
   independent of everything else.
2. **Drop `streamlit-extras` and swap the obsolete hacks for native widgets**
   (2, 3, 5). This is what makes the upgrade cheap.
3. **Upgrade Streamlit**, now that the third party surface is small.
4. **Move the cover fetch into the API** (4) — isolated, and it deletes the
   most fragile code in the dashboard.
5. **`cache_resource`, fragments** (6) — cheap, immediate effect on reruns.
6. **Split the source tests** (8) and **add test CI** (9).
7. **Migrate to `st.navigation` and split the 1028 line file** (1). Last,
   deliberately: it's the expensive one and it's much safer on a clean base.

## What not to do

- **Don't port valentium's plugin architecture to the Go side.** Mantium already
  has its own registry (`sources.Sources`), populated by hand. It isn't hurting
  anything.
- **Don't rewrite the API.** It's the healthy half of this repo.
- Mantium is public, released (`v6.3.1`) and has users. Every item above should
  survive as its own change — none of them needs a big-bang branch.
