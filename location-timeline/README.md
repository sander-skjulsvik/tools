# location-timeline

Works out where you have been from the location metadata of your photos in [Immich](https://immich.app). Use it to answer questions like "where was I on 14 March", "which countries have I been to in the last 6 months", or to give a finance review or diary the context of where you were.

**It never writes to Immich.** It makes one kind of request, a metadata search, and keeps everything else in a local data folder.

Photos only show where you were when you took one. A period without photos is shown as a gap. It is never filled in from the photos on either side.

## Setup

Create an API key in Immich (Account settings → API keys) with only the `asset.read` permission. Save it in the data folder, or set `IMMICH_API_KEY`. The data folder holds your location history, so keep it out of any repo:

```fish
mkdir -p ~/.local/share/location-timeline
chmod 700 ~/.local/share/location-timeline
install -m 600 /dev/null ~/.local/share/location-timeline/api-key
read -s -P "Immich API key: " key
echo $key > ~/.local/share/location-timeline/api-key
set -e key
```

Set the server with `IMMICH_URL` or `-url`. Use the server's root URL, without a path.

Then fetch your photos' locations, and list the cameras in them:

```bash
go run ./location-timeline sync immich
go run ./location-timeline cameras
```

## Sources and config

`config.json` in the data folder has one section per source, and only the sources it lists are used. Their points are merged in time order before any report. Immich is the only source so far; Google location history is planned.

### Immich: which photos count

Not every geotagged photo in your library shows where you were: forwarded and shared photos carry someone else's location. A photo counts only if it was taken with one of your cameras, or if you are in it:

```json
{
  "immich": {
    "cameras": [
      {"make": "Apple", "model": "iPhone 15 Pro"},
      {"make": "FUJIFILM"}
    ],
    "people": ["Your Name"]
  }
}
```

- A camera without a `model` matches every model of that make. Use the names exactly as `cameras` prints them; case does not matter.
- `people` takes your name or person ID in Immich. A name that more than one person in Immich has is refused, so use the ID then.
- Changing the config needs no new sync: `sync` keeps every geotagged photo.

## Usage

```bash
go run ./location-timeline stays -since 2025-01-01
go run ./location-timeline stays -level country -since 1y
go run ./location-timeline stays -on 2025-03-14
go run ./location-timeline days -since 2025-10-01 -until 2025-10-31
go run ./location-timeline countries -since 6m
go run ./location-timeline points -since 30d > points.json
```

| Command | What it does |
|---------|--------------|
| `sync immich` | Fetches the location, time, camera and people of every geotagged photo. The only command using the network |
| `cameras` | Lists the cameras in your geotagged Immich photos and which ones the config counts as yours |
| `stays` | Groups photos into stays in one place, with the gaps between them |
| `days` | Lists the places seen on each day, in order |
| `countries` | Lists the countries seen in a period, with how many days have photos at all |
| `points` | Prints every counted point as JSON, for other tools |

Dates take `YYYY-MM-DD` or a time ago: `30d`, `6w`, `6m`, `1y`. `-on` is a single day. Every command but `points` prints a table, or JSON with `-json`.

`stays` starts a new stay when a photo is more than `-radius` km from the current stay's centre, or with `-level country` when the country changes. Any time longer than `-max-gap` without photos becomes a gap, also between two stays in the same place.

`countries` can only list countries that have photos. It says how many days of the period have photos, and the longest run of days without, so you know how much to trust it.

Dates are local to where the photo was taken, so a photo at 00:30 in Tokyo is on that Tokyo date.

## Using the output elsewhere

The `-json` output and `points` are meant for other tools and skills. Every point has a `source` (only `immich` today), the time in UTC, the local time, coordinates, place names, and `evidence`, which says why the source counted it: for Immich, `camera`, `person` or both. Sources that know how accurate a location is add `accuracy_m`; photos leave it out. In `stays` output, entries are of `kind` `stay` or `gap`. A gap is unknown, not "still there".

## Adding a source

A source implements `Source` in `source.go`: a name, and `Points()`, which reads what the source saved in the data folder and returns only the points its config counts. Add its config section to `Config` and list it in `Sources`. How the data gets into the data folder is up to the source, since sources differ: Immich has an API to sync from, while Google Timeline is a file exported from the phone.

## Data folder

Defaults to `~/.local/share/location-timeline`; change it with `-data-dir`, before the command.

```
api-key       the API key, unless IMMICH_API_KEY is set
config.json   the sources to use, and which of their data is yours
immich.json   every geotagged photo from the last sync
```
