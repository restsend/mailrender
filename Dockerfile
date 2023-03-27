FROM golang:1.19-alpine AS build-env

#RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
#RUN set -ex && \
#    apk upgrade --no-cache --available && \
#    apk add --no-cache build-base

WORKDIR /mailrender

COPY go.mod go.sum ./

COPY . ./
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn
RUN go mod download
RUN go build .

FROM alpine:3.17
LABEL maintainer="mailrender@restsend.com"
LABEL org.opencontainers.image.source=https://github.com/restsend/mailrender
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN set -ex && \
    apk upgrade --no-cache --available && \
    apk --no-cache add ca-certificates tzdata chromium 
    
RUN apk --no-cache add font-liberation \
    font-noto-cjk \
    font-noto-emoji \
    font-noto-thai font-noto-arabic \
    font-freefont fontconfig font-roboto font-ubuntu-nerd

RUN ln -fs /usr/share/zoneinfo/America/New_York /etc/localtime

COPY fonts /usr/share/fonts/win
COPY conf/local.conf /etc/fonts/local.conf

RUN fc-cache -f && rm -rf /var/cache/*

WORKDIR /app
COPY --from=build-env /mailrender/mailrender /app/
ADD entrypoint.sh /app/
ADD html /app/html

EXPOSE 8000
CMD ["/app/entrypoint.sh"]