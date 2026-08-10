# Capture One catalog tools

Reading and rewriting a Capture One catalog (`*.cocatalogdb`) directly.

Derived by inspecting `\\TRUENAS\photos\CaptureOne\MainCaputreOneCatalog\MainCaputreOneCatalog.cocatalogdb`
(Capture One 16.5, 90 234 images). Everything below was verified against that
catalog and the files on disk — see "How this was verified" at the bottom.

## The two tools

```bash
# Export everything to CSV / JSON, with paths resolved and metadata flattened
python3 co_export.py csv  CATALOG.cocatalogdb -o export.csv
python3 co_export.py json CATALOG.cocatalogdb -o export.json

# Write XMP sidecars for digiKam to pick up (dry run unless --apply)
python3 co_export.py xmp CATALOG.cocatalogdb --apply

# List images the catalog points at but that are no longer on disk
python3 co_export.py missing CATALOG.cocatalogdb

# Move files and keep the catalog pointing at them (dry run unless -apply)
./comove -catalog CATALOG.cocatalogdb \
         -from 'Originals\2022\10\9\0' \
         -to-root '\\TRUENAS\photos' -to-path 'sorted\2022-10' \
         -to-dir /mnt/truenas/photos/sorted/2022-10
```

When running against a *copy* of the database, pass `--local-catalog-dir` /
`-catalog-dir` so relative paths still resolve to the real catalog directory.

---

## Schema

It is a **Core Data** store — hence the `Z` prefixes, `Z_PK` primary keys and
`Z_ENT` entity discriminators. `ZENTITIES` maps `Z_ENT` numbers to names
(`12 = Image`, `15 = Variant`, `38 = PathLocation`, `2 = AlbumCollection`, …),
which is the key to reading the rest.

### The core chain

```
ZPATHLOCATION ──< ZIMAGE ──< ZVARIANT ──< ZVARIANTLAYER ──< ZVARIANTMETADATA
   (folders)      (files)   (an edit of    (3 per variant)   (rating, keywords,
                             a file)                          labels, IPTC)
```

- **`ZIMAGE`** — one row per file on disk. Filename, camera/EXIF fields, GPS,
  dimensions, hashes, `ZISTRASHED`, `ZISINSIDECATALOG`.
- **`ZVARIANT`** — one row per *editable version*. Almost always 1:1 with an
  image (in this catalog 90 209 images have one variant, 25 have 2–4).
  `ZVARIANT.ZIMAGE` → `ZIMAGE.Z_PK`.
- **`ZVARIANTLAYER`** — the adjustment stack, ~250 columns of sliders. Exactly
  three rows per variant, distinguished by `Z_ENT`:

  | `Z_ENT` | Entity | Pointed at by | Meaning |
  |---|---|---|---|
  | 18 | `VariantDefaultLayer` | `ZVARIANT.ZDEFAULTLAYER` | baseline from the file / XMP |
  | 17 | `VariantAdjustmentLayer` | `ZVARIANT.ZADJUSTMENTLAYER` | your edits |
  | 20 | `VariantCombinedLayer` | `ZVARIANT.ZCOMBINEDSETTINGS` | **the effective result** |

  (`Z_ENT 23`, `LocalAdjustmentsLayer`, appears for the few images with local
  adjustment masks.)
- **`ZVARIANTMETADATA`** — rating, colour tag, keywords, IPTC. Joined via
  `ZVARIANTMETADATA.ZLAYER = ZVARIANTLAYER.Z_PK`.

**To read what Capture One actually shows, always go through the combined layer:**

```sql
SELECT i.ZIMAGEFILENAME, m.ZBASIC_RATING, m.ZCOLOR_TAG_INDEX, m.ZCONTENT_KEYWORDS
FROM ZIMAGE i
JOIN ZVARIANT         v ON v.ZIMAGE = i.Z_PK
JOIN ZVARIANTMETADATA m ON m.ZLAYER = v.ZCOMBINEDSETTINGS;
```

Joining the default or adjustment layer instead gives you a partial answer —
in this catalog the combined layer has 16 620 keyworded rows where the
adjustment layer has only 3 529.

### Where the files are

`ZIMAGE.ZIMAGELOCATION` → `ZPATHLOCATION.Z_PK`. The folder is stored once and
shared by every image in it (553 rows for 90 k images).

| `ZISRELATIVE` | Resolution |
|---|---|
| `1` | `<catalog directory>` + `ZRELATIVEPATH` — a file stored *inside* the catalog |
| `0` | `ZWINROOT` + `ZRELATIVEPATH` — a *referenced* file elsewhere |

`ZRELATIVEPATH` uses backslashes and never includes the filename; append
`ZIMAGE.ZIMAGEFILENAME`. `ZWINROOT` is a Windows root as typed —
`\\TRUENAS\photos`, `P:\`, `C:\` — so mapping it to a Linux mount point is on you.
`ZMACROOT`, `ZVOLUME` and `ZWINATTRIBUTE` are empty throughout this catalog.

### `ZSIDECARPATH` is not a path to the file

This one matters, and it is the thing most likely to bite you.

`ZIMAGE.ZSIDECARPATH` looks like a path (`2022/10/9/61/DSCF1013.JPG`, forward
slashes) and for 89 662 of 89 724 internal images it happens to equal
`ZRELATIVEPATH` minus the leading `Originals\`. But it is **a stable identity
key, not a location**. It indexes the parallel trees:

```
Adjustments/LAM/<sidecarpath dir>/<filename>.comask   local adjustment masks
Adjustments/{ICC,LCC}/…                               profiles / lens casts
Cache/Previews/<sidecarpath dir>/…                    previews
Cache/Thumbnails/<zero-padded ZVARIANT.Z_PK>.cot      thumbnails
```

Proof that it is an identity and not a location: the 62 images where the two
disagree are all files that were *moved* — `ZRELATIVEPATH` followed them to the
new folder while `ZSIDECARPATH` stayed on the old value. Referenced images show
it even more plainly: files sitting in `\\TRUENAS\photos\to be inported` carry
`ZSIDECARPATH = 2024/3/4/506/_DSF5855.JPG`, a directory they have never been in.

**So: when you move a file, update `ZIMAGELOCATION` and leave `ZSIDECARPATH`
alone.** Rewriting it orphans that image's masks and previews. `comove` does
exactly this.

Note that thumbnails are keyed by **variant** PK, not image PK — so they need no
attention on a move at all.

### Collections

`ZCOLLECTION` is a tree via `ZPARENT`, with **`-1` as the root sentinel, not
`NULL`** (a recursive CTE starting at `ZPARENT IS NULL` returns nothing).
`Z_ENT` says what kind: `2` album, `3`/`36` folder, `5` smart, `7` project,
`8` virtual folder, `11` trash, `40` all-images.

Membership lives in two tables and you generally need both:

- `ZVARIANTINCOLLECTION` (21 116 rows) — the normal case
- `ZIMAGEINCOLLECTION` (29 960 rows) — used by the auto-generated
  "Recent Imports" albums

Smart collections store their rules as XML in `ZSMARTCRITERIA` / `ZSEARCHCRITERIA`
and have no membership rows — you would have to evaluate them yourself.

Folder collections (`Z_ENT = 36`) have a NULL `ZNAME`; their name comes from
`ZFOLDERLOCATION` → `ZPATHLOCATION`.

### Field encodings

| Field | Encoding |
|---|---|
| All dates (`ZEXP_DATE`, `ZIMPORTDATE`, `ZDATECREATED`, …) | **Unix epoch seconds**, *not* the Core Data 2001 epoch |
| `ZGPSLATITUDE` / `ZGPSLONGITUDE` | `"59,56.7543N"` = degrees, decimal minutes, hemisphere |
| `ZCONTENT_KEYWORDS` | `"marta\|\|0,oslo\|\|1"` — comma-separated `name||order` |
| `ZBASIC_RATING` | 0–5; NULL and 0 both mean unrated |
| `ZEXP_SHUTTERSPEED` | seconds as a float (0.02 = 1/50) |
| `ZCOLOR_TAG_INDEX` | see below |

Colour tags — 1–4 and 6–7 confirmed against the XMP sidecars Capture One wrote
itself; **5 is inferred** from Capture One's tag order, because all 20 images
tagged 5 in this catalog have sidecars predating the tag and carry no label:

| 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 |
|---|---|---|---|---|---|---|---|
| none | Red | Orange | Yellow | Green | *Blue (inferred)* | Pink | Purple |

### Tables you can ignore

`ZCAPTUREPILOT`, `ZENABLEDOUTPUTRECIPE`, `ZSELECTEDVARIANTS`,
`ZCOMPAREVARIANTS`, `ZPROCESSHISTORY`, `ZDOCUMENTSETTING` — UI state, tethering
config and export history. `ZSIDECAR` is empty here. `ZKEYWORD` is the keyword
*library* (a nested set, `ZLEFT`/`ZRIGHT`) and is **not** how keywords attach to
photos; that is the `ZCONTENT_KEYWORDS` string on the metadata row.

---

## Getting to digiKam

The good news: Capture One already writes XMP sidecars next to the originals
containing `xmp:Rating`, `xmp:Label`, `dc:subject` and
`lightroom:hierarchicalSubject` — exactly what digiKam reads. Point digiKam at
the folders, enable sidecar reading, and ratings/labels/keywords come across.

The catch: **those sidecars go stale.** In this catalog every image tagged Blue
has a sidecar written before the tag, with no `<xmp:Label>` at all. The database
is authoritative; the sidecars are a snapshot from whenever Capture One last
synced. So regenerate them first:

```bash
python3 co_export.py xmp CATALOG.cocatalogdb          # dry run: how many
python3 co_export.py xmp CATALOG.cocatalogdb --apply  # overwrite from the DB
```

This writes rating, colour label, keywords, title and description, matching
byte-for-byte what Capture One produces for the same image. It only writes next
to files that exist, skips trashed images, and only for the first variant (extra
variants have no file of their own to sit beside).

What does **not** survive the trip: albums, and adjustments. Albums are a
catalog concept with no XMP equivalent — export them from the `collections`
column of the CSV and rebuild them as digiKam tags or albums. Adjustments are
Capture One's own slider values in `ZVARIANTLAYER` and mean nothing to digiKam;
if you want the edited look you have to render out processed files.

---

## Writing to the catalog safely

1. **Capture One must be closed, and the catalog must have shut down cleanly.**
   A `*.cocatalogdb-journal` file next to the database means it did not. Writing
   then is how you corrupt it. `comove` refuses to run with `-apply` if a
   journal file is present.
2. **Back up first.** `cp` the `.cocatalogdb`; it is self-contained.
3. **Do all related writes in one transaction.**
4. **Do not renumber `Z_PK`s** and do not delete `ZPATHLOCATION` rows that other
   images still reference.
5. Set `Z_ENT` correctly on any row you insert — Core Data uses it to decide
   which class to instantiate, and a wrong value confuses Capture One rather
   than erroring. `ZPATHLOCATION` rows need `Z_ENT = 38`.
6. `PRAGMA integrity_check` afterwards, then open it in Capture One and look.

A move needs exactly three things: a `ZPATHLOCATION` row for the destination
(reuse an existing one if it matches), `ZIMAGE.ZIMAGELOCATION` repointed at it
(plus `ZISINSIDECATALOG`, and `ZIMAGEFILENAME` if you also rename), and the file
itself moved along with its `.xmp`.

---

## How this was verified

- Path resolution: all 400 sampled catalog-internal images resolved to files
  that exist. The 337 sampled misses were all *referenced* images, and the
  folders were confirmed by hand to no longer contain those files — genuine
  broken links in the catalog, not a flaw in the rule.
- Colour tags and ratings: read back out of Capture One's own XMP sidecars.
- `ZSIDECARPATH` semantics: from the 62 moved-file discrepancies and the
  referenced images in `to be inported`.
- `comove`: exercised end-to-end on a copy of the real catalog with 178 dummy
  files — dry run, then `-apply`, then confirmed the new `ZPATHLOCATION` row,
  178 repointed images, `ZSIDECARPATH` untouched, files plus sidecars moved,
  `integrity_check` clean, and the exporter reading the new location back.
- Not verified: writing to the *live* catalog and reopening it in Capture One.
  Do that once on a copy before trusting it with the real thing.
