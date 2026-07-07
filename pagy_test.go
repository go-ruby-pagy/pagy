// Copyright (c) the go-ruby-pagy/pagy authors
//
// SPDX-License-Identifier: BSD-3-Clause

package pagy

import (
	"errors"
	"net/url"
	"reflect"
	"strconv"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

// TestNewDefaults exercises the default-application and arithmetic of New for
// the canonical middle-of-the-set page.
func TestNewDefaults(t *testing.T) {
	p, err := New(Vars{Count: 720, Page: 6})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Items != DefaultItems {
		t.Errorf("Items = %d, want %d", p.Items, DefaultItems)
	}
	if p.Limit() != DefaultItems {
		t.Errorf("Limit = %d, want %d", p.Limit(), DefaultItems)
	}
	if p.Last != 36 || p.Pages() != 36 {
		t.Errorf("Last/Pages = %d/%d, want 36/36", p.Last, p.Pages())
	}
	if p.Offset != 100 {
		t.Errorf("Offset = %d, want 100", p.Offset)
	}
	if p.From != 101 || p.To != 120 || p.In != 20 {
		t.Errorf("from/to/in = %d/%d/%d, want 101/120/20", p.From, p.To, p.In)
	}
	if p.Prev != 5 || p.Next != 7 {
		t.Errorf("prev/next = %d/%d, want 5/7", p.Prev, p.Next)
	}
	if p.Vars.PageParam != DefaultPageParam {
		t.Errorf("PageParam = %q, want %q", p.Vars.PageParam, DefaultPageParam)
	}
	if p.Vars.Size != DefaultSize {
		t.Errorf("Size = %d, want %d", p.Vars.Size, DefaultSize)
	}
	if p.Vars.Ends == nil || *p.Vars.Ends != true {
		t.Errorf("Ends default = %v, want true", p.Vars.Ends)
	}
}

// TestNewClampsAndNormalizes covers the negative/zero clamps: count, page,
// items, outset, size.
func TestNewClampsAndNormalizes(t *testing.T) {
	p, err := New(Vars{Count: -5, Page: -3, Items: -1, Outset: -2, Size: -7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Count != 0 {
		t.Errorf("Count = %d, want 0", p.Count)
	}
	if p.Page != 1 {
		t.Errorf("Page = %d, want 1", p.Page)
	}
	if p.Items != DefaultItems {
		t.Errorf("Items = %d, want %d", p.Items, DefaultItems)
	}
	if p.Outset != 0 {
		t.Errorf("Outset = %d, want 0", p.Outset)
	}
	if p.Vars.Size != DefaultSize {
		t.Errorf("Size = %d, want %d", p.Vars.Size, DefaultSize)
	}
}

// TestNewCountZero: count 0 yields a single empty page.
func TestNewCountZero(t *testing.T) {
	p, err := New(Vars{Count: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Last != 1 {
		t.Errorf("Last = %d, want 1", p.Last)
	}
	if p.From != 0 || p.To != 0 || p.In != 0 {
		t.Errorf("from/to/in = %d/%d/%d, want 0/0/0", p.From, p.To, p.In)
	}
	if p.Prev != 0 {
		t.Errorf("Prev = %d, want 0", p.Prev)
	}
	if p.Next != 0 {
		t.Errorf("Next = %d, want 0 (no cycle)", p.Next)
	}
}

// TestNewSinglePage: count fits in one page, page 1 == last.
func TestNewSinglePage(t *testing.T) {
	p, err := New(Vars{Count: 15, Items: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Last != 1 {
		t.Errorf("Last = %d, want 1", p.Last)
	}
	if p.From != 1 || p.To != 15 || p.In != 15 {
		t.Errorf("from/to/in = %d/%d/%d, want 1/15/15", p.From, p.To, p.In)
	}
}

// TestNewOutset: outset shifts the offset and record window.
func TestNewOutset(t *testing.T) {
	p, err := New(Vars{Count: 100, Page: 2, Items: 10, Outset: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Offset != 15 { // items*(page-1)+outset = 10*1+5
		t.Errorf("Offset = %d, want 15", p.Offset)
	}
	if p.From != 11 || p.To != 20 { // offset-outset+1 .. offset-outset+items
		t.Errorf("from/to = %d/%d, want 11/20", p.From, p.To)
	}
}

// TestNewCycle: on the last page with cycle, Next wraps to 1.
func TestNewCycle(t *testing.T) {
	p, err := New(Vars{Count: 40, Page: 2, Items: 20, Cycle: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Next != 1 {
		t.Errorf("Next = %d, want 1 (cycle wrap)", p.Next)
	}
	if p.Prev != 1 {
		t.Errorf("Prev = %d, want 1", p.Prev)
	}
}

// TestNewMaxPages: max_pages caps the last page (and can trigger overflow).
func TestNewMaxPages(t *testing.T) {
	p, err := New(Vars{Count: 1000, Page: 3, Items: 20, MaxPages: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Last != 3 {
		t.Errorf("Last = %d, want 3 (capped)", p.Last)
	}
	// max_pages larger than the natural last page leaves it untouched.
	p2, err := New(Vars{Count: 40, Page: 1, Items: 20, MaxPages: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p2.Last != 2 {
		t.Errorf("Last = %d, want 2 (uncapped)", p2.Last)
	}
}

// TestNewOverflow: a page past the last page returns an *OverflowError.
func TestNewOverflow(t *testing.T) {
	p, err := New(Vars{Count: 720, Page: 40, Items: 20})
	if p != nil {
		t.Fatalf("expected nil Pagy on overflow, got %+v", p)
	}
	var oe *OverflowError
	if !errors.As(err, &oe) {
		t.Fatalf("expected *OverflowError, got %T (%v)", err, err)
	}
	if oe.Page != 40 || oe.Last != 36 {
		t.Errorf("overflow = page %d last %d, want 40/36", oe.Page, oe.Last)
	}
	const want = "expected :page in 1..36; got 40"
	if oe.Error() != want {
		t.Errorf("Error() = %q, want %q", oe.Error(), want)
	}
}

func seriesEqual(t *testing.T, got []any, want []any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("series =\n  %#v\nwant\n  %#v", got, want)
	}
}

// TestSeriesMiddleWithGaps reproduces the canonical README shape and covers all
// four gap-setting "true" branches (start > 1, not an end page).
func TestSeriesMiddleWithGaps(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20}) // last 36
	seriesEqual(t, mustSeries(t, p, 9), []any{1, Gap, 4, 5, "6", 7, 8, Gap, 36})
}

// TestSeriesBeginning covers the branches where series[0]==1 and series[1]==2
// (no left gap) while the right side gets a gap.
func TestSeriesBeginning(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 1, Items: 20}) // last 36
	seriesEqual(t, mustSeries(t, p, 9), []any{"1", 2, 3, 4, 5, 6, 7, Gap, 36})
}

// TestSeriesEnd covers the branches where series[-2]==last-1 and
// series[-1]==last (no right gap) while the left side gets a gap.
func TestSeriesEnd(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 36, Items: 20}) // last 36
	seriesEqual(t, mustSeries(t, p, 9), []any{1, Gap, 30, 31, 32, 33, 34, 35, "36"})
}

// TestSeriesDefaultSize7 uses the default size via Series().
func TestSeriesDefaultSize7(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20}) // last 36
	seriesEqual(t, p.Series(), []any{1, Gap, 5, "6", 7, Gap, 36})
}

// TestSeriesSmallNoGaps covers the size < 7 branch: a bare central window.
func TestSeriesSmallNoGaps(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20}) // last 36
	seriesEqual(t, mustSeries(t, p, 5), []any{4, 5, "6", 7, 8})
}

// TestSeriesEndsDisabled covers the *Ends == false branch: size >= 7 but no
// forced first/last/gaps.
func TestSeriesEndsDisabled(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20, Ends: boolPtr(false)}) // last 36
	seriesEqual(t, mustSeries(t, p, 9), []any{2, 3, 4, 5, "6", 7, 8, 9, 10})
}

// TestSeriesBeginningClampWindow covers the "start = 1" branch when the page is
// within the left half for a small size (page <= left).
func TestSeriesBeginningClampWindow(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 1, Items: 20}) // last 36
	seriesEqual(t, mustSeries(t, p, 5), []any{"1", 2, 3, 4, 5})
}

// TestSeriesEndClampWindow covers the "start = last-size+1" branch for a small
// size (page in the last half).
func TestSeriesEndClampWindow(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 36, Items: 20}) // last 36
	seriesEqual(t, mustSeries(t, p, 5), []any{32, 33, 34, 35, "36"})
}

// TestSeriesSizeGELast covers the "size >= last" branch: the full range.
func TestSeriesSizeGELast(t *testing.T) {
	p, _ := New(Vars{Count: 60, Page: 2, Items: 20}) // last 3
	seriesEqual(t, mustSeries(t, p, 7), []any{1, "2", 3})
}

// TestSeriesZero covers the size == 0 branch.
func TestSeriesZero(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20})
	got, err := p.SeriesSize(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seriesEqual(t, got, []any{})
}

// TestSeriesNegative covers the size < 0 error branch.
func TestSeriesNegative(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20})
	got, err := p.SeriesSize(-1)
	if got != nil {
		t.Errorf("expected nil series, got %v", got)
	}
	var ve *VariableError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VariableError, got %T (%v)", err, err)
	}
	const want = "expected :size to be an Integer >= 0; got -1"
	if ve.Error() != want {
		t.Errorf("Error() = %q, want %q", ve.Error(), want)
	}
}

// TestSeriesCurrentIsString asserts the current page is a string, not an int.
func TestSeriesCurrentIsString(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20})
	for _, v := range mustSeries(t, p, 9) {
		if s, ok := v.(string); ok {
			if s != "6" {
				t.Errorf("current page string = %q, want %q", s, "6")
			}
			return
		}
	}
	t.Error("no string element (current page) found in series")
}

func mustSeries(t *testing.T, p *Pagy, size int) []any {
	t.Helper()
	s, err := p.SeriesSize(size)
	if err != nil {
		t.Fatalf("SeriesSize(%d): %v", size, err)
	}
	return s
}

// TestGapString covers SeriesGap.String.
func TestGapString(t *testing.T) {
	if Gap.String() != "gap" {
		t.Errorf("Gap.String() = %q, want %q", Gap.String(), "gap")
	}
}

// TestURLDefaultBuilder covers the pure query-string builder with params and a
// fragment.
func TestURLDefaultBuilder(t *testing.T) {
	p, _ := New(Vars{
		Count:       720,
		Page:        6,
		Items:       20,
		RequestPath: "/posts",
		Params:      url.Values{"q": {"go"}},
		Fragment:    "#list",
	})
	got := p.URL(3, nil)
	const want = "/posts?page=3&q=go#list"
	if got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}

// TestURLNoParamsNoFragment covers the builder with empty params/fragment and a
// custom page param.
func TestURLNoParamsNoFragment(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20, PageParam: "p"})
	got := p.URL(2, nil)
	const want = "?p=2"
	if got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}

// TestURLFormatterSeam covers the URLFormatter override branch.
func TestURLFormatterSeam(t *testing.T) {
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20})
	got := p.URL(9, func(page int) string {
		return "/custom/" + strconv.Itoa(page)
	})
	if got != "/custom/9" {
		t.Errorf("URL = %q, want %q", got, "/custom/9")
	}
}

// TestPrevNextURL covers both branches of Prev/NextURL (present and absent).
func TestPrevNextURL(t *testing.T) {
	// Middle page: both present.
	p, _ := New(Vars{Count: 720, Page: 6, Items: 20})
	if got := p.PrevURL(nil); got != "?page=5" {
		t.Errorf("PrevURL = %q, want %q", got, "?page=5")
	}
	if got := p.NextURL(nil); got != "?page=7" {
		t.Errorf("NextURL = %q, want %q", got, "?page=7")
	}
	// Single page: neither present.
	one, _ := New(Vars{Count: 5, Items: 20})
	if got := one.PrevURL(nil); got != "" {
		t.Errorf("PrevURL = %q, want empty", got)
	}
	if got := one.NextURL(nil); got != "" {
		t.Errorf("NextURL = %q, want empty", got)
	}
}
