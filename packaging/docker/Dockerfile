# syntax=docker/dockerfile:1@sha256:572cebbc2b15e6f84c077742952674889cb63cd5e9b80abdffcbe587b6f19183
# Supply a reviewed immutable base digest from release/container.json.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
ARG TARGETARCH
ARG PRODUCT_VERSION
ARG SOURCE_REPOSITORY=unconfigured
RUN apk --no-cache add ca-certificates tzdata iputils
COPY dist/ztarc-cli_${PRODUCT_VERSION}_linux_${TARGETARCH} /usr/local/bin/ztarc-cli
COPY LICENSE NOTICE THIRD_PARTY_NOTICES /usr/share/licenses/ztarc-cli/
COPY packaging/docker/entrypoint.sh /usr/local/bin/ztarc-entrypoint
RUN chmod 0755 /usr/local/bin/ztarc-cli /usr/local/bin/ztarc-entrypoint && mkdir -m 0700 /etc/ztarc
LABEL org.opencontainers.image.title="ZTARC CLI" org.opencontainers.image.version="${PRODUCT_VERSION}" org.opencontainers.image.source="${SOURCE_REPOSITORY}"
ENV NEWT_SYSTEM_SUBSTRATE=CONTAINER
ENTRYPOINT ["/usr/local/bin/ztarc-entrypoint"]
