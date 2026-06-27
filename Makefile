.PHONY: ut
ut:
	go test $(shell go list ./... | grep -vE '/cmd($|/)')
