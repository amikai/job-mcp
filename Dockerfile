# Built by GoReleaser (dockers_v2): the build context contains prebuilt
# binaries laid out as <os>/<arch>/openings-mcp, one per target platform.
FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/openings-mcp /usr/bin/openings-mcp
ENTRYPOINT ["/usr/bin/openings-mcp"]
