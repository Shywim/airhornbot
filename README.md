Note: this is a fork of the original [Airhorn bot](https://github.com/hammerandchisel/airhornbot) with fixes
and new features.

# Airhorn Bot
Airhorn is an example implementation of the [Discord API](https://discordapp.com/developers/docs/intro).
Airhorn bot utilizes the [discordgo](https://github.com/bwmarrin/discordgo) library, a free and open source
library. Airhorn Bot requires Go 1.4 or higher.

## Usage

If you want to get the Airhorn bot to your server, simply go to the [website](https://airhorn.shywim.fr) then
click *Add to Discord*!

### Commands

Mention the bot with 'help' as message for a list of commands! (e.g.: `@Airhorn help`)

## Features coming

- Enable users to upload their sounds to use in their respective servers
- Update dependencies and code cleanup

## Building

Airhorn Bot has two components, a bot client that handles the playing of loyal airhorns, and a web server
that implements OAuth2 and stats.

### Get the bot

**Using `go get`:**

```
go get github.com/shywim/airhornbot
go install github.com/shywim/airhornbot
```

**Using `git`:**

`git clone https://github.com/shywim/airhornbot`

You can also download the archive from here.

### Make

Run `make bot` to obtain the bot server build.  
Run `make web` to obtain the web server build.  
Run `make all` to obtain all the builds. Simple, right?

This should spawn a `bot` executable file and/or a `web` executable file in the sources' root.

## Running the Bot

```
./bot -r "localhost:6379" -t "BOT_USER_TOKEN" -o "OWNER_ID"
```

**-r [OPTIONAL]:**

Address of your redis instance, to store stats about your bot usage.

**-t:**

Your bot account token founds in your developer account.

**-o [OPTIONAL]:**

The owner id (generally yours). To obtain your ID, enable developers settings in discord then right click on
your name then click on "Copy ID". Or you can also type `\@YOUR_NAME` in chat.

## Running the Web Server

```
./airhornweb -r "localhost:6379" -i "APP_CLIENT_ID" -s "APP_CLIENT_SECRET"
```

**-r [OPTIONAL]:**

Address of your redis instance, to store stats about your bot usage.

**-i:**

Your app's client id found in your developer account.

**-s:**

Your app's secret token found in your developer account.
