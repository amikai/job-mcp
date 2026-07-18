// Package amazon accesses Amazon's public careers search API.
//
// Amazon Jobs exposes one undocumented JSON endpoint, /en/search.json. Search
// hits already include full descriptions and qualifications. JobDetail
// performs an exact lookup through that endpoint because the site has no
// separate JSON detail operation.
package amazon
