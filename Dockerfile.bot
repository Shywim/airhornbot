FROM golang:1.13.4-alpine3.10

VOLUME ["/etc/airhornbot", "/data", "/etc/airhornbot/plugins"]

WORKDIR /root

# go plugin package requires CGO
RUN apk add --no-cache git gcc musl-dev ffmpeg
RUN git clone https://github.com/magefile/mage \
	&& cd mage \
	&& go run bootstrap.go \
	&& cd .. && rm -rf mage

COPY . .

RUN mage AirhornBot

CMD ["./airhornbot"]

