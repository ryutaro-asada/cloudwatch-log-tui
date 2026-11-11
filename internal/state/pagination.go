// Package state manages the application state for the CloudWatch Log TUI.
package state

import "sync"

// Pagination tracks the page tokens of a paginated Describe* listing.
// The zero value is ready to use.
type Pagination struct {
	mu          sync.Mutex
	currentPage int
	hasNext     bool
	// pageTokens[n] is the token that fetches page n. Page 1 needs no token.
	pageTokens map[int]*string
}

// Target returns the page to fetch when moving in direction d and the token
// that fetches it. ok is false when there is no page in that direction.
func (p *Pagination) Target(d Direction) (page int, token *string, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch d {
	case Home:
		return 1, nil, true
	case Next:
		if !p.hasNext {
			return 0, nil, false
		}
		page = p.currentPage + 1
	case Prev:
		if p.currentPage <= 1 {
			return 0, nil, false
		}
		page = p.currentPage - 1
	}
	if page == 1 {
		return 1, nil, true
	}
	token, ok = p.pageTokens[page]
	return page, token, ok
}

// Apply records the result of fetching page: the number of items it held and
// the token for the following page. It returns false when page is a later
// page that turned out to be empty; the current page then stays current and
// is marked as the last one.
func (p *Pagination) Apply(page, count int, nextToken *string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if page <= 1 || p.pageTokens == nil {
		p.pageTokens = make(map[int]*string)
	}
	if page > 1 && count == 0 {
		delete(p.pageTokens, page)
		p.hasNext = false
		return false
	}

	p.currentPage = page
	p.hasNext = nextToken != nil && *nextToken != ""
	if p.hasNext {
		p.pageTokens[page+1] = nextToken
	}
	return true
}

// Reset forgets all pages, for use when the listing's parameters change.
func (p *Pagination) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.currentPage = 0
	p.hasNext = false
	p.pageTokens = nil
}

// HasPrev returns true if there is a previous page available.
func (p *Pagination) HasPrev() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.currentPage > 1
}

// HasNext returns true if there is a next page available.
func (p *Pagination) HasNext() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.hasNext
}
