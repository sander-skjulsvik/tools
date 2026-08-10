#!/usr/bin/env python3
"""
Export a Capture One catalog (.cocatalogdb) to JSON / CSV, or write XMP sidecars
that digiKam (or anything else that reads XMP) can pick up.

The catalog is opened read-only. Nothing in the catalog directory is written
unless you use the `xmp` subcommand, which only ever writes *.xmp next to the
originals.

Usage:
    co_export.py json    CATALOG.cocatalogdb [-o out.json]
    co_export.py csv     CATALOG.cocatalogdb [-o out.csv]
    co_export.py xmp     CATALOG.cocatalogdb [--apply]      # dry-run by default
    co_export.py missing CATALOG.cocatalogdb                # broken references
"""
import argparse
import csv
import datetime as dt
import json
import os
import re
import sqlite3
import sys
from xml.sax.saxutils import escape

# ZCOLOR_TAG_INDEX -> XMP label. 1-4 and 6-7 were confirmed against the XMP
# sidecars Capture One itself wrote; 5 is inferred from Capture One's tag order
# (every sidecar with tag 5 in this catalog predates the tag and has no label).
COLOR_TAGS = {0: None, 1: "Red", 2: "Orange", 3: "Yellow",
              4: "Green", 5: "Blue", 6: "Pink", 7: "Purple"}

# Windows roots as stored in ZPATHLOCATION.ZWINROOT -> where they are mounted here.
DEFAULT_ROOT_MAP = {
    r"\\TRUENAS\photos": "/mnt/truenas/photos",
    "P:\\": "/mnt/p",
    "C:\\": "/mnt/c",
}


def parse_gps(value):
    """'59,56.7543N' (deg,decimal-minutes + hemisphere) -> signed decimal degrees."""
    if not value:
        return None
    m = re.match(r"^\s*(-?\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*([NSEW])\s*$", value)
    if not m:
        return None
    deg, minutes, hemi = float(m.group(1)), float(m.group(2)), m.group(3)
    dec = abs(deg) + minutes / 60.0
    return -dec if hemi in ("S", "W") else dec


def parse_keywords(value):
    """'marta||0,oslo||1' -> ['marta', 'oslo'], ordered by the trailing index."""
    if not value:
        return []
    items = []
    for i, part in enumerate(value.split(",")):
        name, sep, idx = part.rpartition("||")
        if not sep:            # no '||' marker; take the chunk verbatim
            name, idx = part, i
        try:
            idx = int(idx)
        except ValueError:
            idx = i
        if name:
            items.append((idx, name))
    return [name for _, name in sorted(items)]


def to_iso(epoch):
    """Capture One stores plain Unix epoch seconds (not the Core Data 2001 epoch)."""
    if epoch is None:
        return None
    try:
        return dt.datetime.fromtimestamp(epoch, dt.timezone.utc).isoformat()
    except (OverflowError, OSError, ValueError):
        return None


def collection_paths(con):
    """Map collection Z_PK -> human readable path, e.g. '[Collection]/2022 pride'."""
    rows = con.execute(
        "SELECT Z_PK, ZPARENT, ZNAME, Z_ENT FROM ZCOLLECTION").fetchall()
    parent = {pk: par for pk, par, _, _ in rows}
    name = {pk: nm for pk, _, nm, _ in rows}
    ent = {pk: e for pk, _, _, e in rows}
    out = {}
    for pk in name:
        parts, cur, guard = [], pk, 0
        while cur in name and guard < 64:
            parts.append(name[cur] or f"<{ent[cur]}:{cur}>")
            cur = parent.get(cur)
            if cur is None or cur < 0:
                break
            guard += 1
        out[pk] = "/".join(reversed(parts))
    return out


def load(con, root_map, win_catalog_dir, local_catalog_dir):
    colls = collection_paths(con)

    # album membership, via variants and (for not-yet-variant'ed imports) images
    var_colls, img_colls = {}, {}
    for vpk, cpk in con.execute(
            "SELECT ZVARIANT, ZCOLLECTION FROM ZVARIANTINCOLLECTION"):
        var_colls.setdefault(vpk, []).append(colls.get(cpk, str(cpk)))
    for ipk, cpk in con.execute(
            "SELECT ZIMAGE, ZCOLLECTION FROM ZIMAGEINCOLLECTION"):
        img_colls.setdefault(ipk, []).append(colls.get(cpk, str(cpk)))

    q = """
    SELECT i.Z_PK, i.ZIMAGEUUID, i.ZIMAGEFILENAME, i.ZDISPLAYNAME,
           i.ZISINSIDECATALOG, i.ZISTRASHED, i.ZEXP_DATE, i.ZIMPORTDATE,
           i.ZCAMERA_MAKE, i.ZCAMERA_MODEL, i.ZCAMERA_LENS, i.ZISO,
           i.ZEXP_APERTURE, i.ZEXP_SHUTTERSPEED, i.ZEXP_FOCALLENGTH,
           i.ZWIDTH, i.ZHEIGHT, i.ZFILE_SIZE,
           i.ZGPSLATITUDE, i.ZGPSLONGITUDE, i.ZRAWFILEQUICKHASH,
           p.ZWINROOT, p.ZRELATIVEPATH, p.ZISRELATIVE,
           v.Z_PK, v.ZVARIANTUUID, v.ZINDEX,
           m.ZBASIC_RATING, m.ZCOLOR_TAG_INDEX, m.ZCONTENT_KEYWORDS,
           m.ZSTATUS_TITLE, m.ZCONTENT_DESCRIPTION, m.ZCONTENT_HEADLINE,
           m.ZCONTACT_CREATOR, m.ZSTATUS_COPYRIGHTNOTICE,
           m.ZIMAGE_CITY, m.ZIMAGE_COUNTRY, m.ZIMAGE_LOCATION
    FROM ZIMAGE i
    LEFT JOIN ZPATHLOCATION   p ON p.Z_PK  = i.ZIMAGELOCATION
    LEFT JOIN ZVARIANT        v ON v.ZIMAGE = i.Z_PK
    LEFT JOIN ZVARIANTMETADATA m ON m.ZLAYER = v.ZCOMBINEDSETTINGS
    ORDER BY i.Z_PK, v.ZINDEX
    """
    for r in con.execute(q):
        (ipk, iuuid, fname, dname, inside, trashed, exp_date, imp_date,
         make, model, lens, iso, aperture, shutter, focal,
         w, h, size, lat, lon, qhash,
         winroot, relpath, isrel,
         vpk, vuuid, vindex,
         rating, ctag, kw, title, desc, headline, creator, copyright_,
         city, country, location) = r

        rel = (relpath or "").replace("\\", "/")
        if isrel:
            # ZISRELATIVE=1 means "relative to the catalog directory itself"
            win_path = f"{win_catalog_dir}\\{relpath or ''}\\{fname}"
            abs_path = os.path.join(local_catalog_dir, rel, fname or "")
        else:
            sep = "" if (winroot or "").endswith("\\") else "\\"
            win_path = f"{winroot or ''}{sep}{relpath or ''}\\{fname}"
            base = root_map.get(winroot)
            abs_path = os.path.join(base, rel, fname or "") if base else None

        yield {
            "image_pk": ipk, "image_uuid": iuuid, "variant_pk": vpk,
            "variant_uuid": vuuid, "variant_index": vindex,
            "filename": fname, "display_name": dname,
            "path": abs_path, "windows_path": win_path,
            "inside_catalog": bool(inside), "trashed": bool(trashed),
            "capture_date": to_iso(exp_date), "import_date": to_iso(imp_date),
            "camera_make": make, "camera_model": model, "lens": lens,
            "iso": iso, "aperture": aperture, "shutter_speed": shutter,
            "focal_length": focal, "width": w, "height": h, "file_size": size,
            "gps_lat": parse_gps(lat), "gps_lon": parse_gps(lon),
            "quick_hash": qhash,
            "rating": rating or 0,
            "color_label": COLOR_TAGS.get(ctag),
            "keywords": parse_keywords(kw),
            "title": title, "description": desc, "headline": headline,
            "creator": creator, "copyright": copyright_,
            "city": city, "country": country, "location": location,
            "collections": sorted(set(
                var_colls.get(vpk, []) + img_colls.get(ipk, []))),
        }


XMP_TEMPLATE = """<x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="co_export">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about=""
    xmlns:xmp="http://ns.adobe.com/xap/1.0/"
    xmlns:dc="http://purl.org/dc/elements/1.1/"
    xmlns:lightroom="http://ns.adobe.com/lightroom/1.0/">
{body} </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>
"""


def build_xmp(rec):
    body = [f"   <xmp:Rating>{rec['rating']}</xmp:Rating>"]
    if rec["color_label"]:
        body.append(f"   <xmp:Label>{escape(rec['color_label'])}</xmp:Label>")
    if rec["title"]:
        body.append("   <dc:title>\n    <rdf:Alt>\n"
                    f"     <rdf:li xml:lang=\"x-default\">{escape(rec['title'])}</rdf:li>\n"
                    "    </rdf:Alt>\n   </dc:title>")
    if rec["description"]:
        body.append("   <dc:description>\n    <rdf:Alt>\n"
                    f"     <rdf:li xml:lang=\"x-default\">{escape(rec['description'])}</rdf:li>\n"
                    "    </rdf:Alt>\n   </dc:description>")
    if rec["keywords"]:
        lis = "\n".join(f"     <rdf:li>{escape(k)}</rdf:li>" for k in rec["keywords"])
        body.append(f"   <dc:subject>\n    <rdf:Bag>\n{lis}\n    </rdf:Bag>\n   </dc:subject>")
        body.append("   <lightroom:hierarchicalSubject>\n    <rdf:Bag>\n"
                    f"{lis}\n    </rdf:Bag>\n   </lightroom:hierarchicalSubject>")
    return XMP_TEMPLATE.format(body="\n".join(body) + "\n")


CSV_FIELDS = ["image_pk", "variant_pk", "filename", "path", "windows_path",
              "inside_catalog", "trashed", "capture_date", "camera_make",
              "camera_model", "lens", "iso", "aperture", "shutter_speed",
              "focal_length", "width", "height", "gps_lat", "gps_lon",
              "rating", "color_label", "keywords", "title", "description",
              "collections"]


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("mode", choices=["json", "csv", "xmp", "missing"])
    ap.add_argument("catalog")
    ap.add_argument("-o", "--out")
    ap.add_argument("--win-catalog-dir",
                    help="Windows path of the catalog dir, used only to build "
                         "the windows_path column for catalog-internal images")
    ap.add_argument("--local-catalog-dir",
                    help="Where the catalog directory is mounted on this machine. "
                         "Defaults to the directory holding the .cocatalogdb — set "
                         "it explicitly when working on a copy of the database.")
    ap.add_argument("--apply", action="store_true",
                    help="xmp mode: actually write the sidecars (default: dry run)")
    args = ap.parse_args()

    local_dir = args.local_catalog_dir or os.path.dirname(os.path.abspath(args.catalog))
    win_dir = args.win_catalog_dir or local_dir
    con = sqlite3.connect(f"file:{args.catalog}?mode=ro", uri=True)
    records = load(con, DEFAULT_ROOT_MAP, win_dir, local_dir)

    if args.mode == "json":
        recs = list(records)
        out = open(args.out, "w", encoding="utf-8") if args.out else sys.stdout
        json.dump(recs, out, indent=1, ensure_ascii=False)
        if args.out:
            out.close()
        print(f"{len(recs)} records", file=sys.stderr)

    elif args.mode == "csv":
        out = open(args.out, "w", newline="", encoding="utf-8") if args.out else sys.stdout
        w = csv.DictWriter(out, fieldnames=CSV_FIELDS, extrasaction="ignore")
        w.writeheader()
        n = 0
        for rec in records:
            row = dict(rec)
            row["keywords"] = "; ".join(rec["keywords"])
            row["collections"] = "; ".join(rec["collections"])
            w.writerow(row)
            n += 1
        if args.out:
            out.close()
        print(f"{n} rows", file=sys.stderr)

    elif args.mode == "missing":
        n = bad = 0
        for rec in records:
            n += 1
            if rec["path"] is None or not os.path.exists(rec["path"]):
                bad += 1
                print(f"{rec['image_pk']}\t{rec['windows_path']}")
        print(f"{bad} of {n} unreachable", file=sys.stderr)

    else:  # xmp
        written = skipped = 0
        for rec in records:
            if not rec["path"] or rec["trashed"] or not os.path.exists(rec["path"]):
                skipped += 1
                continue
            # Only the first variant owns the sidecar; extra variants have no
            # file of their own to sit next to.
            if rec["variant_index"] not in (None, 0):
                skipped += 1
                continue
            xmp_path = os.path.splitext(rec["path"])[0] + ".xmp"
            if args.apply:
                with open(xmp_path, "w", encoding="utf-8") as fh:
                    fh.write(build_xmp(rec))
            written += 1
        verb = "wrote" if args.apply else "would write"
        print(f"{verb} {written} sidecars, skipped {skipped}", file=sys.stderr)


if __name__ == "__main__":
    main()
