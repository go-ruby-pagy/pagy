// Copyright (c) the go-ruby-pagy/pagy authors
//
// SPDX-License-Identifier: BSD-3-Clause

package pagy

import (
	"fmt"
	"net/url"
	"strconv"
)

// Default variable values, mirroring Ruby pagy's DEFAULT hash.
const (
	// DefaultItems is the default number of items per page (pagy: items 20).
	DefaultItems = 20
	// DefaultSize is the default series size (pagy: size 7).
	DefaultSize = 7
	// DefaultPageParam is the default query-string parameter for the page
	// number (pagy: page_param :page).
	DefaultPageParam = "page"
)

// SeriesGap is the type of the [Gap] sentinel that marks an elided range of
// pages inside a series, mirroring Ruby pagy's :gap symbol.
type SeriesGap struct{}

// String renders the gap as "gap".
func (SeriesGap) String() string { return "gap" }

// Gap is the sentinel placed in a [Pagy.Series] where a contiguous range of
// page numbers has been elided (Ruby :gap).
var Gap = SeriesGap{}

// Vars are the input variables for [New], mirroring the pagy gem's option hash.
// The zero value of each field means "use the pagy default": Page 1, Items 20,
// Size 7, PageParam "page", Ends true, no page cap.
type Vars struct {
	// Count is the total number of items to paginate. Negative values are
	// treated as 0.
	Count int
	// Page is the requested 1-based page. Values <= 0 clamp to 1. A value
	// past the last page makes [New] return an [*OverflowError].
	Page int
	// Items is the number of items per page. Values <= 0 use [DefaultItems].
	Items int
	// Outset is the number of items skipped before pagination begins
	// (pagy: outset). Negative values are treated as 0.
	Outset int
	// Size is the default series size used by [Pagy.Series]. Values <= 0 use
	// [DefaultSize].
	Size int
	// Ends, when nil, defaults to true: the series forces the first and last
	// page (with gaps) for sizes >= 7. Set to a pointer to false to disable.
	Ends *bool
	// Cycle, when true, makes Next wrap from the last page back to 1
	// (pagy: cycle).
	Cycle bool
	// MaxPages caps the last page when > 0 (pagy: max_pages).
	MaxPages int
	// PageParam is the query-string key carrying the page number in URLs.
	// Empty uses [DefaultPageParam].
	PageParam string
	// RequestPath is the URL path used by [Pagy.URL] (pagy: request_path).
	RequestPath string
	// Params are extra query parameters merged into URLs by [Pagy.URL].
	Params url.Values
	// Fragment is appended to URLs built by [Pagy.URL] (e.g. "#items").
	Fragment string
}

// Pagy is a computed page set: the pure result of pagy's initializer.
type Pagy struct {
	// Count is the total item count (clamped to >= 0).
	Count int
	// Page is the resolved current page, in 1..Last.
	Page int
	// Items is the resolved number of items per page (>= 1).
	Items int
	// Outset is the resolved outset (>= 0).
	Outset int
	// Last is the number of the last page (>= 1). Also available via Pages.
	Last int
	// Offset is the zero-based offset of the first item on the current page.
	Offset int
	// From is the 1-based index of the first item shown (0 when the page is
	// empty).
	From int
	// To is the 1-based index of the last item shown (0 when empty).
	To int
	// In is the number of items shown on the current page.
	In int
	// Prev is the previous page number, or 0 when there is none.
	Prev int
	// Next is the next page number, or 0 when there is none.
	Next int
	// Vars are the normalized input variables (defaults applied).
	Vars Vars
}

// OverflowError reports that the requested page is past the last page,
// mirroring Ruby pagy's Pagy::OverflowError.
type OverflowError struct {
	// Page is the requested (overflowing) page.
	Page int
	// Last is the last valid page.
	Last int
}

// Error renders the message exactly as the gem does, e.g.
// "expected :page in 1..36; got 40".
func (e *OverflowError) Error() string {
	return fmt.Sprintf("expected :page in 1..%d; got %d", e.Last, e.Page)
}

// VariableError reports an out-of-range variable, mirroring Ruby pagy's
// Pagy::VariableError.
type VariableError struct {
	// Variable is the offending variable name (e.g. "size").
	Variable string
	// Description states the expectation (e.g. "to be an Integer >= 0").
	Description string
	// Value is the value that was supplied.
	Value int
}

// Error renders the message exactly as the gem does, e.g.
// "expected :size to be an Integer >= 0; got -1".
func (e *VariableError) Error() string {
	return fmt.Sprintf("expected :%s %s; got %d", e.Variable, e.Description, e.Value)
}

// New computes a [Pagy] from v, applying pagy's defaults and arithmetic. It
// returns an [*OverflowError] when the requested page is past the last page.
func New(v Vars) (*Pagy, error) {
	// normalize_vars + setup_vars: apply defaults and clamp to valid ranges.
	count := v.Count
	if count < 0 {
		count = 0
	}
	page := v.Page
	if page <= 0 {
		page = 1
	}
	items := v.Items
	if items <= 0 {
		items = DefaultItems
	}
	outset := v.Outset
	if outset < 0 {
		outset = 0
	}
	if v.Size <= 0 {
		v.Size = DefaultSize
	}
	if v.PageParam == "" {
		v.PageParam = DefaultPageParam
	}
	if v.Ends == nil {
		ends := true
		v.Ends = &ends
	}

	// setup_last_var: last = max(ceil(count/items), 1), capped by max_pages.
	last := 1
	if count > 0 {
		last = (count + items - 1) / items
	}
	if v.MaxPages > 0 && last > v.MaxPages {
		last = v.MaxPages
	}

	if page > last {
		return nil, &OverflowError{Page: page, Last: last}
	}

	offset := items*(page-1) + outset
	p := &Pagy{
		Count:  count,
		Page:   page,
		Items:  items,
		Outset: outset,
		Last:   last,
		Offset: offset,
		From:   min(offset-outset+1, count),
		To:     min(offset-outset+items, count),
		Vars:   v,
	}
	p.In = min(p.To-p.From+1, count)
	if page != 1 {
		p.Prev = page - 1
	}
	switch {
	case page != last:
		p.Next = page + 1
	case v.Cycle:
		p.Next = 1
	}
	return p, nil
}

// Limit reports the number of items per page (pagy alias of Items / limit).
func (p *Pagy) Limit() int { return p.Items }

// Pages reports the number of pages (pagy alias of Last).
func (p *Pagy) Pages() int { return p.Last }

// Series returns the navigation series using the default size stored in Vars.
// The result never carries an error because the stored size is always valid.
func (p *Pagy) Series() []any {
	s, _ := p.SeriesSize(p.Vars.Size)
	return s
}

// SeriesSize returns the navigation series for the given size: a []any of int
// page links, the current page as a string, and [Gap] for elided ranges. It
// returns a [*VariableError] when size is negative, and an empty slice when
// size is 0, mirroring pagy's series method.
func (p *Pagy) SeriesSize(size int) ([]any, error) {
	if size < 0 {
		return nil, &VariableError{Variable: "size", Description: "to be an Integer >= 0", Value: size}
	}
	if size == 0 {
		return []any{}, nil
	}

	var series []any
	if size >= p.Last {
		for n := 1; n <= p.Last; n++ {
			series = append(series, n)
		}
	} else {
		left := (size - 1) / 2 // left half is 1 shorter for even size
		var start int
		switch {
		case p.Page <= left: // beginning pages
			start = 1
		case p.Page > p.Last-size+left: // end pages
			start = p.Last - size + 1
		default: // intermediate pages
			start = p.Page - left
		}
		for n := start; n < start+size; n++ {
			series = append(series, n)
		}
		// Force first/last pages plus gaps when requested and size allows.
		if *p.Vars.Ends && size >= 7 {
			if series[0] != 1 {
				series[0] = 1
			}
			if series[1] != 2 {
				series[1] = Gap
			}
			if series[len(series)-2] != p.Last-1 {
				series[len(series)-2] = Gap
			}
			if series[len(series)-1] != p.Last {
				series[len(series)-1] = p.Last
			}
		}
	}
	// Render the current page as a string.
	for i, v := range series {
		if n, ok := v.(int); ok && n == p.Page {
			series[i] = strconv.Itoa(p.Page)
			break
		}
	}
	return series, nil
}

// URLFormatter formats an absolute-or-relative URL for a page number. It is the
// only seam in the package: supply one to [Pagy.URL] to override the built-in
// query-string builder with your own routing.
type URLFormatter func(page int) string

// URL builds the URL for the given page. When f is non-nil it delegates to f;
// otherwise it builds a query-string URL from the instance vars — RequestPath,
// Params (merged), the PageParam set to page, and Fragment appended — mirroring
// pagy_url_for as a pure template with no request object.
func (p *Pagy) URL(page int, f URLFormatter) string {
	if f != nil {
		return f(page)
	}
	q := url.Values{}
	for k, vs := range p.Vars.Params {
		q[k] = append([]string(nil), vs...)
	}
	q.Set(p.Vars.PageParam, strconv.Itoa(page))
	return p.Vars.RequestPath + "?" + q.Encode() + p.Vars.Fragment
}

// PrevURL builds the URL for the previous page, or returns "" when there is no
// previous page. See [Pagy.URL] for the meaning of f.
func (p *Pagy) PrevURL(f URLFormatter) string {
	if p.Prev == 0 {
		return ""
	}
	return p.URL(p.Prev, f)
}

// NextURL builds the URL for the next page, or returns "" when there is no next
// page. See [Pagy.URL] for the meaning of f.
func (p *Pagy) NextURL(f URLFormatter) string {
	if p.Next == 0 {
		return ""
	}
	return p.URL(p.Next, f)
}
