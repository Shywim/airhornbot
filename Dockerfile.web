FROM golang:1.13.4-alpine3.10

VOLUME ["/etc/airhornbot", "/data"]

WORKDIR /root

RUN apk add --no-cache git ffmpeg
RUN git clone https://github.com/magefile/mage \
	&& cd mage \
	&& go run bootstrap.go \
	&& cd .. && rm -rf mage

COPY . .

RUN mage AirhornWeb

CMD ["./airhornweb"]

