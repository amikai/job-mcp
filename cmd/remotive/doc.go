// Command remotive is a debug CLI for Remotive's public remote-jobs API.
//
//	go run ./cmd/remotive search --keyword golang --category software-development
//	go run ./cmd/remotive detail --id 2091069
//	go run ./cmd/remotive categories
//
// Every invocation fetches the full dump once and filters it client-side —
// the API's documented query params are no-ops (see the deviation notes in
// internal/provider/remotive/openapi.yaml). Upstream blocks >2 requests
// per minute and asks for at most ~4 fetches a day, so keep invocations
// sparse.
package main
