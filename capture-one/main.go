// comove relocates images that a Capture One catalog points at, and updates the
// catalog to match, so the photos stay linked and keep their adjustments.
//
// What a move actually touches:
//
//	ZIMAGE.ZIMAGELOCATION  -> a ZPATHLOCATION row for the destination folder
//	ZIMAGE.ZIMAGEFILENAME  -> only when the file is also renamed
//	the file on disk, plus its .xmp sidecar if one exists
//
// What it deliberately does NOT touch:
//
//	ZIMAGE.ZSIDECARPATH — this is not the file's location. It is the stable key
//	into Adjustments/{ICC,LAM,LCC}/ and Cache/Previews/, and Capture One keeps it
//	fixed when a file moves. Rewriting it orphans local adjustment masks and
//	previews.
//
// Run without -apply first; that prints the plan and changes nothing.
package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type config struct {
	catalog   string // path to the .cocatalogdb
	catalogFS string // where the catalog *directory* is mounted locally
	from      string // ZPATHLOCATION match: images currently under this folder
	toWinRoot string // destination ZWINROOT, e.g. \\TRUENAS\photos  ("" = catalog-relative)
	toRelPath string // destination ZRELATIVEPATH, backslash separated
	toFS      string // where the destination folder is mounted locally
	apply     bool
}

type image struct {
	pk       int64
	filename string
	locPK    int64
	inside   bool
	winRoot  string
	relPath  string
	isRel    bool
}

func main() {
	var c config
	flag.StringVar(&c.catalog, "catalog", "", "path to the .cocatalogdb file (required)")
	flag.StringVar(&c.catalogFS, "catalog-dir", "", "where the catalog directory is mounted locally (default: dir of -catalog)")
	flag.StringVar(&c.from, "from", "", "move images whose ZPATHLOCATION.ZRELATIVEPATH equals this (required)")
	flag.StringVar(&c.toWinRoot, "to-root", "", `destination ZWINROOT, e.g. \\TRUENAS\photos; empty means catalog-relative`)
	flag.StringVar(&c.toRelPath, "to-path", "", `destination ZRELATIVEPATH, e.g. sorted\2023 (required)`)
	flag.StringVar(&c.toFS, "to-dir", "", "where the destination folder is mounted locally (required)")
	flag.BoolVar(&c.apply, "apply", false, "actually move files and write the catalog")
	flag.Parse()

	if err := run(c); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(c config) error {
	if c.catalog == "" || c.from == "" || c.toRelPath == "" || c.toFS == "" {
		flag.Usage()
		return errors.New("missing required flag")
	}
	if c.catalogFS == "" {
		c.catalogFS = filepath.Dir(c.catalog)
	}

	db, err := sql.Open("sqlite", c.catalog)
	if err != nil {
		return err
	}
	defer db.Close()

	// A catalog left open by Capture One, or closed uncleanly, has a -journal
	// beside it. Writing then is how you corrupt it.
	if _, err := os.Stat(c.catalog + "-journal"); err == nil && c.apply {
		return errors.New("a -journal file exists next to the catalog: close Capture One " +
			"(and let it finish) before writing, or work on a copy")
	}

	images, err := findImages(db, c.from)
	if err != nil {
		return err
	}
	if len(images) == 0 {
		return fmt.Errorf("no images found with ZRELATIVEPATH = %q", c.from)
	}

	// Plan every move before doing any of it, so a conflict aborts cleanly.
	type move struct{ src, dst string }
	moves := make([]move, 0, len(images))
	for _, img := range images {
		src := filepath.Join(img.localDir(c.catalogFS), img.filename)
		dst := filepath.Join(c.toFS, img.filename)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("source missing, refusing to continue: %s", src)
		}
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("destination already exists: %s", dst)
		}
		moves = append(moves, move{src, dst})
	}

	fmt.Printf("%d images: %s  ->  %s%s%s\n", len(images),
		c.from, c.toWinRoot, sep(c.toWinRoot), c.toRelPath)
	for i, m := range moves {
		if i < 5 || i == len(moves)-1 {
			fmt.Printf("  %s -> %s\n", m.src, m.dst)
		} else if i == 5 {
			fmt.Printf("  ... %d more\n", len(moves)-6)
		}
	}
	if !c.apply {
		fmt.Println("\ndry run — nothing changed. Re-run with -apply.")
		return nil
	}

	// Database first, inside one transaction: if the file moves fail we can roll
	// the catalog back, but a half-written catalog cannot be un-written.
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	locPK, err := findOrCreateLocation(tx, c.toWinRoot, c.toRelPath)
	if err != nil {
		return err
	}
	insideCatalog := c.toWinRoot == ""
	for _, img := range images {
		if _, err := tx.Exec(
			`UPDATE ZIMAGE SET ZIMAGELOCATION = ?, ZISINSIDECATALOG = ? WHERE Z_PK = ?`,
			locPK, boolToInt(insideCatalog), img.pk); err != nil {
			return fmt.Errorf("updating image %d: %w", img.pk, err)
		}
	}

	if err := os.MkdirAll(c.toFS, 0o755); err != nil {
		return err
	}
	moved := 0
	for _, m := range moves {
		if err := moveFile(m.src, m.dst); err != nil {
			return fmt.Errorf("after moving %d/%d files: %w (catalog not committed; "+
				"already-moved files must be put back by hand)", moved, len(moves), err)
		}
		// The XMP sidecar rides along; it is matched by basename, not recorded in the DB.
		srcXMP := strings.TrimSuffix(m.src, filepath.Ext(m.src)) + ".xmp"
		if _, err := os.Stat(srcXMP); err == nil {
			dstXMP := strings.TrimSuffix(m.dst, filepath.Ext(m.dst)) + ".xmp"
			if err := moveFile(srcXMP, dstXMP); err != nil {
				return fmt.Errorf("moving sidecar %s: %w", srcXMP, err)
			}
		}
		moved++
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("moved %d files and updated the catalog\n", moved)
	return nil
}

// localDir returns where this image's folder lives on this machine.
func (i image) localDir(catalogFS string) string {
	rel := strings.ReplaceAll(i.relPath, `\`, string(filepath.Separator))
	if i.isRel {
		return filepath.Join(catalogFS, rel)
	}
	// An absolute location; the caller is responsible for the root being mounted.
	return filepath.Join(mountFor(i.winRoot), rel)
}

// mountFor maps the Windows roots this catalog uses onto local mount points.
func mountFor(winRoot string) string {
	switch winRoot {
	case `\\TRUENAS\photos`:
		return "/mnt/truenas/photos"
	case `P:\`:
		return "/mnt/p"
	case `C:\`:
		return "/mnt/c"
	}
	return winRoot
}

func findImages(db *sql.DB, from string) ([]image, error) {
	rows, err := db.Query(`
		SELECT i.Z_PK, i.ZIMAGEFILENAME, i.ZIMAGELOCATION, i.ZISINSIDECATALOG,
		       COALESCE(p.ZWINROOT,''), COALESCE(p.ZRELATIVEPATH,''), COALESCE(p.ZISRELATIVE,0)
		FROM ZIMAGE i JOIN ZPATHLOCATION p ON p.Z_PK = i.ZIMAGELOCATION
		WHERE p.ZRELATIVEPATH = ?`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []image
	for rows.Next() {
		var img image
		var inside, isRel int
		if err := rows.Scan(&img.pk, &img.filename, &img.locPK, &inside,
			&img.winRoot, &img.relPath, &isRel); err != nil {
			return nil, err
		}
		img.inside, img.isRel = inside != 0, isRel != 0
		out = append(out, img)
	}
	return out, rows.Err()
}

// findOrCreateLocation returns the Z_PK of a ZPATHLOCATION row for this folder,
// creating it if the catalog does not already describe it.
func findOrCreateLocation(tx *sql.Tx, winRoot, relPath string) (int64, error) {
	isRel := boolToInt(winRoot == "")
	var pk int64
	err := tx.QueryRow(`
		SELECT Z_PK FROM ZPATHLOCATION
		WHERE COALESCE(ZWINROOT,'') = ? AND COALESCE(ZRELATIVEPATH,'') = ?
		  AND COALESCE(ZISRELATIVE,0) = ?`, winRoot, relPath, isRel).Scan(&pk)
	if err == nil {
		return pk, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	// Z_ENT 38 is the PathLocation entity (see ZENTITIES).
	res, err := tx.Exec(`
		INSERT INTO ZPATHLOCATION (Z_ENT, ZWINROOT, ZMACROOT, ZRELATIVEPATH, ZISRELATIVE, ZVOLUME, ZWINATTRIBUTE)
		VALUES (38, ?, '', ?, ?, '', '')`, winRoot, relPath, isRel)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// moveFile renames, falling back to copy+delete across filesystems.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}
	in.Close()
	return os.Remove(src)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func sep(root string) string {
	if root == "" || strings.HasSuffix(root, `\`) {
		return ""
	}
	return `\`
}
