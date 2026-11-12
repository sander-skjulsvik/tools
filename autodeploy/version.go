package main

import (
	"fmt"
	"strconv"
	"strings"
)

// version is a release tag of the form vMAJOR.MINOR.PATCH.
type version struct{ major, minor, patch int }

// parseVersion accepts only plain release tags such as v1.2.3. Everything
// else (latest, sha-…, pre-releases like v1.2.0-rc.1, leading zeros) is not a
// release and returns false.
func parseVersion(tag string) (version, bool) {
	rest, ok := strings.CutPrefix(tag, "v")
	if !ok {
		return version{}, false
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 3 {
		return version{}, false
	}
	var n [3]int
	for i, p := range parts {
		if p == "" || (len(p) > 1 && p[0] == '0') || strings.Trim(p, "0123456789") != "" {
			return version{}, false
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return version{}, false
		}
		n[i] = v
	}
	return version{n[0], n[1], n[2]}, true
}

func (v version) less(o version) bool {
	if v.major != o.major {
		return v.major < o.major
	}
	if v.minor != o.minor {
		return v.minor < o.minor
	}
	return v.patch < o.patch
}

func (v version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.major, v.minor, v.patch)
}

// newest returns the highest release among tags, ignoring tags that aren't releases.
func newest(tags []string) (version, bool) {
	var best version
	found := false
	for _, t := range tags {
		v, ok := parseVersion(t)
		if ok && (!found || best.less(v)) {
			best, found = v, true
		}
	}
	return best, found
}
