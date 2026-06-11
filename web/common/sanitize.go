package common

import "github.com/microcosm-cc/bluemonday"

// descPolicy allows the basic formatting markup Google Books descriptions use
// (paragraphs, bold/italics, line breaks, lists) and strips everything else —
// notably scripts, event handlers, and embedded media.
var descPolicy = bluemonday.UGCPolicy()

// SafeDescription sanitizes externally-sourced HTML (Google Books
// descriptions, librarian-edited metadata) for use with templ.Raw.
func SafeDescription(s string) string {
	return descPolicy.Sanitize(s)
}
