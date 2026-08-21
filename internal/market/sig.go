package market

import (
	"fmt"
	"strings"
)

func (q *Quote) MarketSig() string {
	if q == nil {
		return ""
	}
	var b strings.Builder
	for _, bk := range q.Books {
		if bk.Current == nil {
			continue
		}
		fmt.Fprintf(&b, "e%d:%.2f/%.2f/%.2f,", bk.CompanyID, bk.Current.H, bk.Current.D, bk.Current.A)
	}
	writeMove := func(tag string, m *LineMove) {
		if m == nil {
			return
		}
		fmt.Fprintf(&b, "%s:%s/%.2f/%.2f,", tag, m.CurrentLine, m.CurrentLeft, m.CurrentRight)
	}
	writeMove("ahm", q.AsianMove)
	writeMove("oum", q.OUMove)
	for _, m := range q.AsianBooks {
		fmt.Fprintf(&b, "ah%d:%s/%.2f/%.2f,", m.CompanyID, m.CurrentLine, m.CurrentLeft, m.CurrentRight)
	}
	for _, m := range q.OUBooks {
		fmt.Fprintf(&b, "ou%d:%s/%.2f/%.2f,", m.CompanyID, m.CurrentLine, m.CurrentLeft, m.CurrentRight)
	}
	return b.String()
}
