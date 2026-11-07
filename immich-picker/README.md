# immich-picker

Builds a local catalog of the photos of one person in [Immich](https://immich.app), then shortlists the good ones. Results are saved per photo, so nothing is processed twice.

**It never writes to Immich, XMP files or the originals.** All output goes to a local data folder.

## Setup

Create an API key in Immich (Account settings → API keys) with only these permissions:

- `asset.read`
- `asset.view`
- `person.read`

Then save it in the data folder, or set `IMMICH_API_KEY`. The data folder holds your photos, so keep it out of any repo:

```fish
mkdir -p ~/.local/share/immich-picker
chmod 700 ~/.local/share/immich-picker
install -m 600 /dev/null ~/.local/share/immich-picker/api-key
read -s -P "Immich API key: " key
echo $key > ~/.local/share/immich-picker/api-key
set -e key
```

Set the server with `IMMICH_URL` or `-url`. Use the server's root URL, without a path.

## Usage

```bash
go run ./immich-picker collect -person "Name" -since 2023-01-01 \
    -query "hiking in the mountains" -query "smiling portrait" -query "cooking"
go run ./immich-picker analyze
go run ./immich-picker shortlist -limit 150
go run ./immich-picker sheets
go run ./immich-picker import-labels labels.json
go run ./immich-picker status
```

`-person` takes a name, an ID, or a link to the person's page in Immich. `-data-dir` and `-url` go before the command.

Only `collect` and `analyze` use the network. `shortlist` works from saved results, so its filters can be changed and re-run freely.

| Step | What it does |
|------|--------------|
| `collect` | Saves each photo's Immich metadata and which `-query` themes it matched |
| `analyze` | Downloads previews and measures face size, other faces, face sharpness and a similarity hash |
| `shortlist` | Rejects unsuitable photos, keeps the best of each duplicate group and outing, and takes turns between themes |
| `sheets` | Draws the shortlist into numbered grids, for reviewing many photos as a few images |
| `import-labels` | Validates labels from an LLM or a person and merges them into the catalog |

Similar photos are grouped in three ways, all from saved results:

- Immich's own duplicate groups, when its duplicate detection has run (`duplicateId`)
- A similarity hash, for near-identical frames Immich didn't group (`-duplicate-bits`)
- Bursts: one photo from any moment shorter than `-burst-window`
- Outings: photos with no gap longer than `-outing-gap` between them, capped by `-max-per-outing`

Ranking works against phone selfies winning by default:

- Sharpness is ranked against photos from the same camera, and cameras take turns filling the list
- Faces earn no extra credit beyond `-face-area-enough`, and small faces are never scaled up for measuring

Search themes always return their closest matches, even poor ones, so a theme means "among the closest", not "shows this". Keep `-query-limit` low.

## Data folder

Defaults to `~/.local/share/immich-picker`; change it with `-data-dir`.

```
api-key               the API key, unless IMMICH_API_KEY is set
run.json              the person, date range and photo list of the last collect
assets/<id>.json      everything known about one photo
previews/             downloaded previews
shortlist.json        the ranked result
rejections.json       why each other photo was left out
shortlist/            copies of the shortlisted previews, named by rank
sheets/               contact sheets; each number is a rank in shortlist.json
```

Each `assets/<id>.json` has these sections:

| Section | Source |
|---------|--------|
| `immich` | Immich's response, including EXIF and face boxes |
| `queries` | Themes matched in the last `collect` |
| `local` | Computed metrics, tagged with `metricsVersion` |
| `llm` | Imported labels, with the model and prompt version |

`local` and `llm` are kept across runs. They are discarded only when the image's checksum changes. Bump `metricsVersion` in `analyze.go` to recompute metrics without downloading previews again.

## Labels file

A JSON list. The whole file is rejected if any entry is invalid.

```json
[
  {
    "asset_id": "…",
    "model": "model-name",
    "prompt_version": "1",
    "keep": true,
    "category": "activity",
    "description": "Hiking on a ridge, looking at the camera",
    "issues": ["slightly backlit"],
    "edit_needed": true,
    "edit_notes": "Lift the shadows on the face from the RAW",
    "score": 7,
    "notes": "optional"
  }
]
```

`score` and `notes` are optional. Set `edit_needed` on a photo that is a good fit only after editing; `edit_notes` is then required. A photo with the category `exclude`, such as an identity document, is never shortlisted again.

## Why it is safe to run

- `allowedRequests` in `immich.go` lists every request the tool can make. `guardTransport` refuses anything else, or any other host, before it is sent.
- Immich implements search as `POST`, so the two search endpoints are the only `POST`s allowed.
- Redirects are never followed, so the API key can't be forwarded to another host.
- The API key's permissions enforce the same limit on the server.

## Tests

```bash
go test ./immich-picker/
```
