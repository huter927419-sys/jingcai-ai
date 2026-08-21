package store

import (
	"testing"
	"time"
)

func TestListAccessCodesPagingAndFilter(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	for _, days := range []int{3, 7, 15, 30} {
		if err := st.EnsureAccessPool(days, 12); err != nil {
			t.Fatal(err)
		}
	}
	all, err := st.ListAccessCodes(0, "", "", 1, 20, now)
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 48 || all.Page != 1 || all.PageSize != 20 || all.Pages != 3 || len(all.Codes) != 20 {
		t.Fatalf("all page1 %+v n=%d", all, len(all.Codes))
	}
	if all.Codes[0].DurationDays != 3 {
		t.Fatalf("expected 3-day first, got %d", all.Codes[0].DurationDays)
	}
	page2, err := st.ListAccessCodes(0, "unused", "", 2, 20, now)
	if err != nil {
		t.Fatal(err)
	}
	if page2.Page != 2 || len(page2.Codes) != 20 {
		t.Fatalf("page2 %+v n=%d", page2, len(page2.Codes))
	}
	only7, err := st.ListAccessCodes(7, "unused", "", 1, 50, now)
	if err != nil {
		t.Fatal(err)
	}
	if only7.Total != 12 || len(only7.Codes) != 12 {
		t.Fatalf("7-day %+v n=%d", only7, len(only7.Codes))
	}
	for _, c := range only7.Codes {
		if c.DurationDays != 7 || c.Status != "unused" {
			t.Fatalf("filter leak %+v", c)
		}
	}
	q := only7.Codes[0].Code
	found, err := st.ListAccessCodes(0, "", q, 1, 50, now)
	if err != nil {
		t.Fatal(err)
	}
	if found.Total != 1 || found.Codes[0].Code != q {
		t.Fatalf("search %+v", found)
	}
}
