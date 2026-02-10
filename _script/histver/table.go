package main

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

type tableRow struct {
	Version string
	Runtime string
	Date    string
}

func (t tableRow) merge(other tableRow) tableRow {
	return tableRow{
		Version: cmp.Or(other.Version, t.Version),
		Runtime: cmp.Or(other.Runtime, t.Runtime),
		Date:    cmp.Or(other.Date, t.Date),
	}
}

func (r *tableRow) fromStrings(ss []string) error {
	if len(ss) < 3 {
		return fmt.Errorf("not enough fields")
	}
	r.Version = strings.Trim(ss[0], "*")
	r.Runtime = strings.Trim(ss[1], "*")
	r.Date = strings.Trim(ss[2], "*")
	return nil
}

func (r *tableRow) fromVersion(ver string) error {
	// syncthing v1.23.1-rc.1 "Fermium Flea" (go1.19.5 darwin-arm64) teamcity@build.syncthing.net 2023-01-12 03:30:17 UTC [stnoupgrade]
	exp := regexp.MustCompile(`syncthing (v\d+\.\d+\.\d+).*(go\d+\.\d+(?:\.\d+)?).*(\d{4}-\d{2}-\d{2}) `)
	m := exp.FindStringSubmatch(ver)
	if len(m) < 3 {
		return fmt.Errorf("failed to parse version")
	}
	r.Version = m[1]
	r.Runtime = m[2]
	r.Date = m[3]
	return nil
}

func (r tableRow) toStrings() []string {
	return []string{r.Version, r.Runtime, r.Date}
}

var tableHeader = []string{"Version", "Runtime", "Date"}

func writeTable(w io.Writer, rows []tableRow) error {
	sort.Slice(rows, func(a, b int) bool {
		if rows[a].Date == rows[b].Date {
			return rows[a].Version > rows[b].Version
		}
		return rows[a].Date > rows[b].Date
	})

	prevRunMinor := ""
	prevSynMinor := ""
	for i := len(rows) - 1; i >= 0; i-- {
		r := &rows[i]
		// Bold major/minor runtime releases
		var runMinor string
		if strings.Count(r.Runtime, ".") == 1 {
			// old style "go1.2" type release number
			runMinor = r.Runtime
		} else {
			// modern style "go1.25.0" to release number
			runMinor = r.Runtime[:strings.LastIndex(r.Runtime, ".")]
		}
		if runMinor != prevRunMinor {
			prevRunMinor = runMinor
			r.Runtime = fmt.Sprintf("**%s**", r.Runtime)
		}
		// Bold major/minor Syncthing releases
		synMinor := r.Version[:strings.LastIndex(r.Version, ".")]
		if synMinor != prevSynMinor {
			prevSynMinor = synMinor
			r.Version = fmt.Sprintf("**%s**", r.Version)
		}
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(tableHeader); err != nil {
		return err
	}
	for _, r := range rows {
		if err := cw.Write(r.toStrings()); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func readTable(r io.Reader) ([]tableRow, error) {
	cr := csv.NewReader(r)
	var rows []tableRow
	for {
		ss, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(ss) == 0 {
			continue
		}
		if ss[0] == tableHeader[0] {
			continue
		}
		var row tableRow
		if err := row.fromStrings(ss); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}
