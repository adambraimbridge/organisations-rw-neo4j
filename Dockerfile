FROM alpine:3.3
ADD *.go .git /public-organisations-api/
ADD organisations/*.go /public-organisations-api/organisations/
RUN apk add --update bash \
  && apk --update add git go \
  && cd public-organisations-api \
  && git fetch origin 'refs/tags/*:refs/tags/*' \
  && BUILDINFO_PACKAGE="github.com/Financial-Times/service-status-go/buildinfo." \
  && VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && LDFLAGS="-X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && cd .. \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/public-organisations-api" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && cp -r public-organisations-api/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get ./... \
  && cd $GOPATH/src/${REPO_PATH} \
  && echo ${LDFLAGS} \
  && go build -ldflags="${LDFLAGS}" \
  && mv public-organisations-api /app \
  && apk del go git \
  && rm -rf $GOPATH /var/cache/apk/*
CMD exec /app --neo-url=$NEO_URL --port=$APP_PORT --batchSize=$BATCH_SIZE --graphiteTCPAddress=$GRAPHITE_ADDRESS --graphitePrefix=$GRAPHITE_PREFIX --logMetrics=false --env=local

