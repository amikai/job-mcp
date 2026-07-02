# Built by GoReleaser (dockers_v2): the build context contains prebuilt
# binaries laid out as <os>/<arch>/jobmcp, one per target platform.
FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/jobmcp /usr/bin/jobmcp
ENTRYPOINT ["/usr/bin/jobmcp"]
