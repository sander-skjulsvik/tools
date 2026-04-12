# isodate-nest

Moves ISO-date-prefixed files into nested `YYYY/MM/DD/` subdirectories.

A file named `2026-04-12-notes.txt` in the target folder will be moved to `2026/04/12/notes.txt` inside that same folder.

## Usage

```
isodate-nest [-dry-run] <folder>
```

## Example

Given a folder with these files:

```
photos/
  2026-04-12-holiday.jpg
  2026-03-01-birthday.jpg
  readme.txt
```

Preview the moves without changing anything:

```
$ isodate-nest -dry-run photos/
photos/2026-04-12-holiday.jpg -> photos/2026/04/12-holiday.jpg
photos/2026-03-01-birthday.jpg -> photos/2026/03/01-birthday.jpg
```

Files without a valid ISO date prefix (e.g. `readme.txt`) are skipped.

Run without `-dry-run` to apply the moves:

```
$ isodate-nest photos/
photos/2026-04-12-holiday.jpg -> photos/2026/04/12-holiday.jpg
photos/2026-03-01-birthday.jpg -> photos/2026/03/01-birthday.jpg
```

Result:

```
photos/
  readme.txt
  2026/
    03/
      01-birthday.jpg
    04/
      12-holiday.jpg
```

## Build

```
go build -o isodate-nest .
```
