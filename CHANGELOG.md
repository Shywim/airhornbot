# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/). 

## [Unreleased]
### Added 
 - **bot** Can now send gifs along playing sounds
 - **bot** New help command (using `@Airhorn help`)
 - **web-app** Stats now displayed on mobile
 
### Changed
 - **web-app** Ditched React and the bloat coming with it and rewrote the front page in simple TypeScript
 - **web-app** Removed Stylus preprocessor in favor of standard css (might change for another preprocessor or css-next in the future)
 - **web-app** Changed url for this fork
 - **web-app** Enabled video for mobile
 - **repo** Updated licence and readme
 
### Removed
 - **bot** Removed message spam on channel join
 - **web-app** Removed Google Analytics
 - **web-app** Removed references to an unknown secret count
 - **build** Removed Gulp
 
### Fixed
 - **bot** Fix for discordgo change on `GuildCreate.Guild.Unavaible` [hammerandchisel/airhornbot][origin35]
 - **bot** Fix for discord api auth change [hammerandchisel/discord-api-docs#119][dad119]
 - **bot** Fixed play queue that was not thread safe
 - **bot** Fixed EventSource for nginx

 [Unreleased]: https://github.com/hammerandchisel/airhornbot/compare/master...Shywim:master

 [dad119]: https://github.com/hammerandchisel/discord-api-docs/issues/119
 [origin35]: https://github.com/hammerandchisel/airhornbot/issues/35
