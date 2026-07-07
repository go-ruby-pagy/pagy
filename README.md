<p align="center"><img src="https://go-ruby-pagy.github.io/logo.png" alt="go-ruby-pagy/pagy" width="720"></p>

# pagy — go-ruby-pagy

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-pagy.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the deterministic pagination core of
Ruby's [`pagy`](https://github.com/ddnexus/pagy) gem.** It reproduces, with the
same arithmetic the MRI gem performs (v8.6.3 semantics), the values a `Pagy`
instance computes at construction — `offset`, `limit`, page count (`pages` /
`last`), the `from`/`to` record window, the in-page count (`in`), the
`prev`/`next` links — and the navigation **series** (the
`[1, :gap, 4, 5, "6", 7, 8, :gap, 36]` array, with the current page as a
`String` and elided ranges as `:gap`) — **without any Ruby runtime.**

It is the pagination library for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module — a sibling of the other `go-ruby-*` gems.

> **What it is — and isn't.** Everything `pagy` computes about a page set is pure
> arithmetic and needs **no interpreter**, so it lives here as pure Go: clamping
> the page into range, deriving the `offset` and record window from
> `count`/`items`/`outset`, capping the last page (`max_pages`), and building the
> series with its gaps. The **only application-coupled piece of the gem** —
> rendering a page URL out of the Rack request — is here a **pure template
> seam**: [`Pagy.URL`](pagy.go) builds a query-string URL from the vars on the
> instance, or delegates to an optional `URLFormatter` you supply. No routing, no
> request object.

## Features

Faithful port of the `pagy` gem's calculator:

- **`New(Vars)`** — applies the gem's defaults (`items 20`, `size 7`,
  `page_param :page`, `ends true`, `outset 0`) and arithmetic, returning a
  `*Pagy` or an `*OverflowError` when the requested page is past the last one.
- **Page set** — `Offset`, `Items` (`Limit()`), `Last` (`Pages()`), `From`,
  `To`, `In`, `Prev`, `Next` (0 means "no such page"), with faithful edge
  handling: `page` clamped up from `<= 0`, `count 0` → one empty page, single
  page, `outset`, `cycle` (wrap `next` to 1), `max_pages` cap.
- **`Series()` / `SeriesSize(size)`** — the navigation array as a `[]any` of
  `int` page links, the current page as a `string`, and `Gap` for elided ranges
  (the gem's `:gap`). Reproduces the gem's start-window selection, the
  first/last-plus-gap forcing for `size >= 7`, and the `ends: false` and small
  (`size < 7`) branches. `size 0` yields `[]`; a negative size is a
  `*VariableError`.
- **URL helpers** — `URL(page, formatter)`, `PrevURL`, `NextURL`: a pure
  `request_path?page_param=…&params…#fragment` builder, or your own
  `URLFormatter` seam.

## Usage

```go
p, err := pagy.New(pagy.Vars{Count: 1000, Page: 6, Items: 20})
if err != nil {
	// *OverflowError when Page is past the last page
}
p.Offset        // 100
p.Limit()       // 20   (alias of p.Items)
p.Pages()       // 50   (alias of p.Last)
p.From, p.To    // 101, 120
p.Prev, p.Next  // 5, 7

p.Series()      // [1 :gap 5 "6" 7 :gap 50]      (default size 7)
p.SeriesSize(9) // [1 :gap 4 5 "6" 7 8 :gap 50]  (window of 5 + first/last)

p.URL(3, nil)   // "?page=3"   (or supply a URLFormatter)
```

## Ruby surface

A host (`go-embedded-ruby` / `rbgo`) maps its Ruby `Pagy` object onto these
shapes so `require "pagy"` works CGO-free:

```ruby
pagy = Pagy.new(count: 1000, page: 6, items: 20)
pagy.offset   # 100
pagy.limit    # 20
pagy.pages    # 50
pagy.series   # [1, :gap, 5, "6", 7, :gap, 50]
pagy.prev     # 5
pagy.next     # 7
```

`Pagy::OverflowError` maps to `*OverflowError`.

## Tests & coverage

Pure computation, deterministic, no network and no filesystem: every branch —
the overflow error, `count 0`, single page, `outset`, the gap / no-gap series
branches, `ends: false`, `max_pages`, `cycle`, and the URL seam — is exercised,
holding statement coverage at **100%**. The suite runs under `-race`, on the six
supported 64-bit targets (`amd64`/`arm64`/`riscv64`/`loong64`/`ppc64le`/`s390x`,
the emulated lanes under qemu-user), and builds for `js/wasm` and `wasip1/wasm`.

```sh
go test -race ./...
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the `go-ruby-pagy/pagy` authors.
