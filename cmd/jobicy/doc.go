// Command jobicy is a debug CLI for the Jobicy remote-jobs feed.
//
//	go run ./cmd/jobicy search --tag golang --geo usa --industry dev --count 5
//	go run ./cmd/jobicy locations
//	go run ./cmd/jobicy industries
//
// The feed has no detail endpoint: every search row already carries the
// complete HTML description, which --format json includes as
// jobDescription. Text output shows the plain-text excerpt instead.
package main
