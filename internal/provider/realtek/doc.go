// Package realtek provides a client for the JSON endpoints backing
// Realtek's recruitment site (https://recruit.realtek.com/Job/Search), a
// Kendo UI grid fed by jQuery AJAX calls. Recon done 2026-07-21; see
// openapi.yaml's info.description for the full list of surface quirks
// (the Pass/Data envelope, the 200-row-capped dump vs. form-urlencoded
// filtered list with minimum-experience xp and soft non-keyword filters,
// JobOppId-keyed detail with its all-null not-found shape, and the
// string-typed numeric fields). Realtek is a single site, not a roster of
// companies.
package realtek
