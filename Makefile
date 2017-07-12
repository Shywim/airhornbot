BOT_BINARY=bot
WEB_BINARY=web

JS_FILES = $(shell find static/src/ -type f -name '*.ts')

.PHONY: all
all: bot web

bot: cmd/bot/bot.go
	go build -o ${BOT_BINARY} cmd/bot/bot.go

web: cmd/webserver/web.go tsbuild
	go build -o ${WEB_BINARY} cmd/webserver/web.go

npm: web-app/package.json
	cd web-app && npm install .

tsbuild: $(JS_FILES) npm
	cd web-app && npm build

.PHONY: clean
clean:
	rm -r ${BOT_BINARY} ${WEB_BINARY} web-app/public/js
