//go:build w5_cutover_gate

package docx_v2_test

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestPostFlipSoakLogComplete runs under the w5_cutover_gate build tag and
// is the CI prerequisite for Task 3 (destructive migration 0113) and
// Task 6 (code destruction).
//
// Time-enforcement: the gate parses the flip timestamp from
// the log header and asserts
//   (a) all 5 Day dates are unique, in strict chronological order,
//   (b) every Day date is on-or-after the flip date,
//   (c) every Day date is not in the future relative to time.Now().UTC(),
//   (d) the span from Day 1 to Day 5 is at least 5 business days (Mon–Fri),
//       which means the earliest possible GO is 4 business days after flip.
// This prevents same-day backfill of all 5 entries.
func TestPostFlipSoakLogComplete(t *testing.T) {
	raw, err := os.ReadFile("../../docs/superpowers/evidence/docx-v2-w5-post-flip-soak.md")
	if err != nil {
		t.Fatalf("post-flip soak log missing: %v", err)
	}
	content := string(raw)

	flipRe := regexp.MustCompile(`(?m)^Flag flip applied \(UTC\):\s+(\d{4}-\d{2}-\d{2})T`)
	flipMatch := flipRe.FindStringSubmatch(content)
	if flipMatch == nil {
		t.Fatalf("soak log missing or malformed 'Flag flip applied (UTC):' header")
	}
	flipDate, err := time.Parse("2006-01-02", flipMatch[1])
	if err != nil {
		t.Fatalf("unparsable flip date %q: %v", flipMatch[1], err)
	}

	headerRe := regexp.MustCompile(`(?m)^### Day [1-5] — (\d{4}-\d{2}-\d{2})$`)
	headers := headerRe.FindAllStringSubmatch(content, -1)
	if len(headers) < 5 {
		t.Fatalf("soak log must contain 5 dated Day sections; found %d", len(headers))
	}
	for _, h := range headers {
		if h[1] == "YYYY-MM-DD" {
			t.Fatalf("soak log Day header uses placeholder date: %q", h[0])
		}
	}

	dayDates := make([]time.Time, 0, len(headers))
	seenDates := map[string]bool{}
	nowUTC := time.Now().UTC().Truncate(24 * time.Hour)
	for i, h := range headers {
		if seenDates[h[1]] {
			t.Fatalf("duplicate Day date %q at position %d", h[1], i+1)
		}
		seenDates[h[1]] = true
		d, err := time.Parse("2006-01-02", h[1])
		if err != nil {
			t.Fatalf("unparsable Day %d date %q: %v", i+1, h[1], err)
		}
		if d.Before(flipDate) {
			t.Fatalf("Day %d date %s is BEFORE flip date %s — soak cannot start before the flip",
				i+1, d.Format("2006-01-02"), flipDate.Format("2006-01-02"))
		}
		if d.After(nowUTC) {
			t.Fatalf("Day %d date %s is in the future (now=%s)",
				i+1, d.Format("2006-01-02"), nowUTC.Format("2006-01-02"))
		}
		if i > 0 && !d.After(dayDates[i-1]) {
			t.Fatalf("Day %d date %s is not strictly after Day %d date %s",
				i+1, d.Format("2006-01-02"), i, dayDates[i-1].Format("2006-01-02"))
		}
		dayDates = append(dayDates, d)
	}

	bd := businessDaysInclusive(dayDates[0], dayDates[len(dayDates)-1])
	if bd < 5 {
		t.Fatalf("soak span Day 1 (%s) → Day 5 (%s) is %d business days; require >= 5",
			dayDates[0].Format("2006-01-02"), dayDates[len(dayDates)-1].Format("2006-01-02"), bd)
	}

	placeholderTokens := []string{
		"__.__%", "__ms", "__%", "__/day", "__/__",
		"YYYY-MM-DD",
		"GO / NO-GO",
	}
	ellipsisLineRe := regexp.MustCompile(`(?m)^\s*\.\.\.\s*$`)
	if ellipsisLineRe.FindString(content) != "" {
		t.Fatalf("soak log contains an unfilled '...' line")
	}
	for _, tok := range placeholderTokens {
		if strings.Contains(content, tok) {
			t.Fatalf("soak log contains unfilled placeholder token %q", tok)
		}
	}

	dayBlocks := splitCutoverDayBlocks(content)
	if len(dayBlocks) < 5 {
		t.Fatalf("could not isolate 5 day blocks; got %d", len(dayBlocks))
	}

	requiredRows := []struct {
		name string
		re   *regexp.Regexp
	}{
		{"availability", regexp.MustCompile(`(?m)^- /api/v2/documents availability:\s+(\d{2,3}\.\d{1,2})\s*%`)},
		{"p95", regexp.MustCompile(`(?m)^- /api/v2/export/pdf p95:\s+(\d+)\s*ms\b`)},
		{"cached ratio", regexp.MustCompile(`(?m)^- cached=false ratio:\s+(\d{1,3})\s*%`)},
		{"429 max", regexp.MustCompile(`(?m)^- 429 rate per user \(max\):\s+(\d+)/day\b`)},
		{"OOM events", regexp.MustCompile(`(?m)^- Gotenberg OOM events:\s+(\d+)\b`)},
		{"tenants exercising", regexp.MustCompile(`(?m)^- Tenants exercising /api/v2:\s+(\d+)/(\d+)\b`)},
		{"Incidents", regexp.MustCompile(`(?m)^- Incidents:\s+(none|P0|P1|P2)\b`)},
		{"Sign-off", regexp.MustCompile(`(?m)^- Sign-off:\s+@\S+\s+@\S+\s*$`)},
	}

	for i, block := range dayBlocks {
		dayNum := i + 1
		for _, row := range requiredRows {
			if row.re.FindString(block) == "" {
				t.Fatalf("Day %d: missing or malformed %q row", dayNum, row.name)
			}
		}
	}

	exerciseRe := regexp.MustCompile(`(?m)^- Tenants exercising /api/v2:\s+(\d+)/(\d+)\b`)
	anyHigh := false
	for _, m := range exerciseRe.FindAllStringSubmatch(content, -1) {
		a, errA := strconv.Atoi(m[1])
		b, errB := strconv.Atoi(m[2])
		if errA != nil || errB != nil || b == 0 {
			continue
		}
		if float64(a)/float64(b) >= 0.80 {
			anyHigh = true
			break
		}
	}
	if !anyHigh {
		t.Fatalf("no day with >=80%% tenants exercising /api/v2 — dark-launch proof requirement")
	}

	for _, label := range []string{"Admin sign-off:", "Product manager sign-off:", "SRE sign-off:"} {
		re := regexp.MustCompile(regexp.QuoteMeta(label) + `\s+@\S+\s+—\s+\d{4}-\d{2}-\d{2}`)
		if re.FindString(content) == "" {
			t.Fatalf("soak log missing filled %q line", label)
		}
	}

	goRe := regexp.MustCompile(`(?m)^\*\*Decision:\*\*\s+GO\b`)
	if goRe.FindString(content) == "" {
		t.Fatalf("soak log missing explicit **Decision:** GO line")
	}
}

func businessDaysInclusive(from, to time.Time) int {
	if to.Before(from) {
		return 0
	}
	count := 0
	d := from
	for !d.After(to) {
		wd := d.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			count++
		}
		d = d.AddDate(0, 0, 1)
	}
	return count
}

func splitCutoverDayBlocks(content string) []string {
	blocks := []string{}
	headerRe := regexp.MustCompile(`(?m)^### Day [1-5] — \d{4}-\d{2}-\d{2}\s*$`)
	idxs := headerRe.FindAllStringIndex(content, -1)
	for i, start := range idxs {
		bodyStart := start[1]
		var bodyEnd int
		if i+1 < len(idxs) {
			bodyEnd = idxs[i+1][0]
		} else {
			rest := content[bodyStart:]
			nextSec := regexp.MustCompile(`(?m)^##[^#]`).FindStringIndex(rest)
			if nextSec != nil {
				bodyEnd = bodyStart + nextSec[0]
			} else {
				bodyEnd = len(content)
			}
		}
		blocks = append(blocks, content[bodyStart:bodyEnd])
	}
	return blocks
}
