// Copyright (c) the go-ruby-pagy/pagy authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package pagy is a pure-Go (CGO-free) reimplementation of the deterministic
// pagination core of Ruby's [pagy] gem. It reproduces, byte-for-byte with the
// MRI gem (v8.6.3 semantics), the arithmetic a Pagy instance performs at
// construction — offset, limit, page count, the from/to record window, the
// in-page count, the prev/next links — and the navigation series algorithm
// (the [1, :gap, 4, 5, "6", 7, 8, :gap, 36] array, with the current page as a
// String and elided ranges as :gap).
//
// It is the pagination library for
// [go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a
// standalone, reusable module — a sibling of the other go-ruby-* gems.
//
// # What it is — and isn't
//
// Everything pagy computes about a page set is pure arithmetic and needs no
// interpreter, so it lives here as pure Go: clamping the page into range,
// deriving the offset and record window from count/items/outset, capping the
// last page, and building the series with its gaps. The overflow condition
// (a page past the last one) surfaces as an [*OverflowError], mirroring the
// gem's Pagy::OverflowError.
//
// Rendering a URL for a page is the only piece that touches application state
// in the gem (it reaches into the Rack request). Here it is a pure template
// seam: [Pagy.URL] builds a query-string URL from the vars stored on the
// instance (request path, params, page param, fragment), or delegates to an
// optional [URLFormatter] you supply — no routing, no request object.
//
// # Flow
//
//	p, err := pagy.New(pagy.Vars{Count: 1000, Page: 6, Items: 20})
//	if err != nil {
//		// *OverflowError when Page is past the last page
//	}
//	p.Offset      // 100
//	p.Limit()     // 20   (alias of p.Items)
//	p.Pages()     // 50   (alias of p.Last)
//	p.From, p.To  // 101, 120
//	p.Prev, p.Next // 5, 7 (0 means "no such page")
//	p.Series()    // [1 :gap 4 5 "6" 7 8 :gap 50]  (default size 7... etc.)
//
// # Value model
//
// A [Pagy] carries the decoded page-set fields. A series is a []any whose
// elements are int (a page link), string (the current page), or [Gap] (an
// elided range, the gem's :gap). Construction errors are [*OverflowError];
// an out-of-range series size is a [*VariableError]. A host
// (go-embedded-ruby / rbgo) maps its Ruby Pagy object and Pagy::OverflowError
// onto these shapes: Pagy.new(count:, page:, items:) and
// .offset/.limit/.pages/.series/.prev/.next.
//
// [pagy]: https://github.com/ddnexus/pagy
package pagy
