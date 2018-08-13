# Golang v1.10
FROM golang:1.10.2-alpine
WORKDIR /go/src/github.com/maurodelazeri/winter

ARG LIBRDKAFKA_NAME="librdkafka"
ARG LIBRDKAFKA_VER="0.11.4"

# Install librdkafka
RUN apk add --no-cache --virtual .fetch-deps \
    ca-certificates \
    libressl \
    wget \ 
    build-base \
    curl \
    tar && \
    \
    BUILD_DIR="$(mktemp -d)" && \
    \
    curl -sLo "$BUILD_DIR/$LIBRDKAFKA_NAME.tar.gz" "https://github.com/edenhill/librdkafka/archive/v$LIBRDKAFKA_VER.tar.gz" && \
    mkdir -p $BUILD_DIR/$LIBRDKAFKA_NAME-$LIBRDKAFKA_VER && \
    tar \
    --extract \
    --file "$BUILD_DIR/$LIBRDKAFKA_NAME.tar.gz" \
    --directory "$BUILD_DIR/$LIBRDKAFKA_NAME-$LIBRDKAFKA_VER" \
    --strip-components 1 && \
    \
    apk add --no-cache --virtual .build-deps \
    bash \
    g++ \
    libressl-dev \
    make \
    musl-dev \
    zlib-dev && \
    \
    cd "$BUILD_DIR/$LIBRDKAFKA_NAME-$LIBRDKAFKA_VER" && \
    ./configure \
    --prefix=/usr && \
    make -j "$(getconf _NPROCESSORS_ONLN)" && \
    make install && \
    \
    runDeps="$( \
    scanelf --needed --nobanner --recursive /usr/local \
    | awk '{ gsub(/,/, "\nso:", $2); print "so:" $2 }' \
    | sort -u \
    | xargs -r apk info --installed \
    | sort -u \
    )" && \
    apk add --no-cache --virtual .librdkafka-rundeps \
    $runDeps && \
    \
    cd / && \
    apk del .fetch-deps .build-deps && \
    rm -rf $BUILD_DIR

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh g++

RUN go get -u github.com/golang/dep/cmd/dep
RUN go get -u github.com/confluentinc/confluent-kafka-go/kafka

# SSH Acess to the private repo
RUN mkdir /root/.ssh/
ADD id_rsa /root/.ssh/id_rsa
RUN ssh-keyscan bitbucket.org >> /root/.ssh/known_hosts
RUN chmod 700 /root/.ssh/id_rsa

RUN git clone --depth 1 git@bitbucket.org:maurodelazeri/winter.git data
RUN mv data/* .
COPY . .

RUN dep ensure
RUN go build -o  /app/winter
