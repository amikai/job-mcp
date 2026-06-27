.PHONY: ut validate-openapi

OPENAPI_SPECS := \
	internal/cake/openapi.yaml \
	internal/google/openapi.yaml \
	internal/synopsys/openapi.yaml \
	internal/tsmc/openapi.yaml

ut:
	go test $(shell go list ./... | grep -vE '/cmd($|/)')

validate-openapi: $(OPENAPI_SPECS)
	@for spec in $^; do \
		echo "Validating $$spec..."; \
		docker run --rm -v "$(PWD)/$$spec:/openapi.yaml" pythonopenapi/openapi-spec-validator /openapi.yaml; \
	done
