BOT_BINARY=bot
WEB_BINARY=web

JS_FILES = $(shell find web-app/src/ -type f -name '*.ts')

.PHONY: all
all: bot web

bot: cmd/bot/bot.go
	go build -o ${BOT_BINARY} cmd/bot/bot.go

web: cmd/webserver/web.go npm
	go build -o ${WEB_BINARY} cmd/webserver/web.go

npm: web-app/package.json
	cd web-app && npm install .

.PHONY: clean
clean:
	rm -r ${BOT_BINARY} ${WEB_BINARY} web-app/public/js
