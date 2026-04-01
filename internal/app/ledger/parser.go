package ledger

import (
	"bufio"
	"regexp"
	"strings"
	"time"
)

var (
	rePeriod = regexp.MustCompile(`(\d{2}/\d{2}/\d{4}).+(\d{2}/\d{2}/\d{4})`)
	reHeader = regexp.MustCompile(`^ID#`)
	reDate   = regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)
	reAccount = regexp.MustCompile(`^\d+(?:-\d+)+$`)
)

func ParseSnapshot(reader *bufio.Reader) (*Snapshot, error) {
	snapshot := &Snapshot{
		Parties: map[string]*SnapshotParty{},
	}

	scanner := bufio.NewScanner(reader)

	var (
		currentParty string
		headerPassed bool
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if match := rePeriod.FindStringSubmatch(line); len(match) == 3 {
			snapshot.PeriodStart = match[1]
			snapshot.PeriodEnd = match[2]
			continue
		}

		if reHeader.MatchString(line) {
			headerPassed = true
			continue
		}
		if !headerPassed {
			continue
		}

		if strings.Contains(line, "Total") || strings.HasPrefix(line, "Grand") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		if len(parts) >= 7 && reDate.MatchString(parts[1]) {
			if currentParty == "" {
				continue
			}

			dt, err := time.Parse("02/01/2006", parts[1])
			if err != nil {
				continue
			}

			amountIdx := len(parts) - 3
			amount, ok := parseSnapshotAmount(parts[amountIdx])
			if !ok {
				continue
			}

			accountIdx := -1
			for idx := 2; idx < amountIdx; idx++ {
				if isSnapshotAccountToken(parts[idx]) {
					accountIdx = idx
					break
				}
			}

			account := ""
			refNo := ""
			if accountIdx >= 0 {
				account = parts[accountIdx]
				if accountIdx+1 < amountIdx {
					refNo = strings.Join(parts[accountIdx+1:amountIdx], " ")
				}
			} else if 2 < amountIdx {
				refNo = strings.Join(parts[2:amountIdx], " ")
			}

			item := SnapshotItem{
				ID:      parts[0],
				Date:    dt,
				Account: account,
				RefNo:   strings.TrimSpace(refNo),
				Amount:  amount,
				Tax:     parts[amountIdx+1],
				Status:  parts[amountIdx+2],
			}

			snapshot.Parties[currentParty].Items = append(snapshot.Parties[currentParty].Items, item)
			continue
		}

		party := normalizePartyName(line)
		if party == "" {
			continue
		}
		currentParty = party
		if _, exists := snapshot.Parties[currentParty]; !exists {
			snapshot.Parties[currentParty] = &SnapshotParty{
				Party: currentParty,
				Items: []SnapshotItem{},
			}
		}
	}

	return snapshot, scanner.Err()
}

func isSnapshotAccountToken(token string) bool {
	value := strings.TrimSpace(token)
	if value == "" {
		return false
	}
	return reAccount.MatchString(value)
}
