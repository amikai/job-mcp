// Package bamboohr provides a public hosted-careers API client and curated
// company roster.
//
// Each customer board lives at https://{subdomain}.bamboohr.com/careers.
// The public surface is a full dump of sparse summaries at /careers/list plus
// a per-job /careers/{id}/detail; there is no server-side search. Quirks
// (302 on unknown tenants, locationType codes, sparse list fields) are
// documented on openapi.yaml.
package bamboohr
