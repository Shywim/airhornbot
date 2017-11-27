Note: this is a fork of the original [Airhorn bot](https://github.com/hammerandchisel/airhornbot) with fixes
and new features.

# Airhorn Bot
Airhorn is an example implementation of the [Discord API](https://discordapp.com/developers/docs/intro).
Airhorn bot utilizes the [discordgo](https://github.com/bwmarrin/discordgo) library, a free and open source
library. Airhorn Bot requires Go 1.8 or higher.

## Usage

If you want to get the Airhorn bot to your server, simply go to the [website](https://airhorn.shywim.fr) then
click *Add to Discord*!

### Commands

Mention the bot with 'help' as message for a list of commands! (e.g.: `@Airhorn help`)

## Self host

Airhorn Bot has two components, a bot client that handles the playing of loyal airhorns,
and a web server to connect the bot to a discord server and manage custom clips.

### Using Docker

 - **The bot**

     docker run -d --name airhornbot \
     	-v /etc/airhornbot:/etc/airhornbot \
	-v /etc/airhornbot/plugins:/etc/airhornbot/plugins \
	-v airhornbot-data:/data \
	--link airhornbot-redis:redis \
	--link airhornbot-db:db \
	registry.gitlab.com/Shywim/airhornbot/bot:latest

 - **Web application**

     docker run -d --name airhornweb -p 14000:14000 \
     	-v /etc/airhornbot:/etc/airhornbot \
	-v airhornbot-data:/data \
	--link airhornbot-redis:redis \
	--link airhornbot-db:db \
	registry.gitlab.com/Shywim/airhornbot/web:latest

### Get the bot

	// TODO

