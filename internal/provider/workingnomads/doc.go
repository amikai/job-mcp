// Package workingnomads reads public job listings from Working Nomads's
// exposed_jobs API at https://www.workingnomads.com/api/exposed_jobs/.
// There is no official developer API or OpenAPI spec; this package's
// contract was reverse-engineered directly against the live endpoint
// (2026-07-19). The canonical host is workingnomads.com — workingnomads.co
// 301-redirects to it.
//
// # One endpoint, full dump, no server-side search
//
// The endpoint takes no path or query parameters that do anything: a
// plain GET returns the whole board (34 jobs at capture time) with full
// HTML descriptions inline, and probing plausible query params
// (?category=, ?tag=, and an unrecognized one) all returned a
// byte-identical body to the unfiltered request — confirmed live,
// 2026-07-19 (see jobs_noop_query_req.hurl). There is also no
// /api/categories, /api/tags, or similar discovery endpoint (each 404s),
// so [FilterOptions.Category] has no fixed enum to validate against; it
// matches whatever free-text category_name value a job happens to carry.
// No User-Agent spoofing or auth is needed to fetch it, unlike some other
// hand-scraped providers in this repo.
//
// # Job identity and detail
//
// Each entry's "url" field is not a Working Nomads-hosted detail page —
// it is an outbound apply-tracking redirect (HTTP 302) straight to the
// employer's own posting or ATS, e.g.
// https://www.workingnomads.com/job/go/1734670/ redirected to a Lemon.io
// listing at capture time. Appending that same numeric id back onto the
// exposed_jobs path 404s (see detail_not_found_req.hurl) — there is no
// per-job endpoint. [Client.Detail] instead resolves an id against a
// fresh [Client.Jobs] fetch, the same "no detail endpoint, resolve from
// the dump" shape used by the Remotive and We Work Remotely providers.
//
// [Job.ID] is that numeric segment (e.g. "1734670"), not the array index
// the entry happens to occupy in a given response — a numeric index would
// silently reassign identity to a different job the moment the dump's
// ordering shifts between fetches.
//
// # Fields
//
// "location" is a single free-text field with no fixed shape — observed
// values include a single word ("Global"), a comma-joined region list
// ("Europe, North America, Latin America, APAC"), a "<country> - Remote"
// pair, and a US-state pair ("Texas, Oklahoma") — so
// [FilterOptions.Location] is a plain substring match, not a structured
// city/state/country split. "tags" is a comma-joined free-text list,
// parsed into [Job.Tags] but also folded into [FilterOptions.Keyword]
// matching, same as skills in the We Work Remotely provider.
package workingnomads
