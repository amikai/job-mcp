//go:generate go tool github.com/ogen-go/ogen/cmd/ogen --target . -package oracle --clean openapi.yaml

// Package oracle discovers Oracle Recruiting Cloud Candidate Experience sites
// and provides generated and site-bound clients for anonymous job search and
// detail resources.
package oracle
