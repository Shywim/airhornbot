# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/). 

## [Unreleased]
### Added 
 - **bot** Can now send gifs along playing sounds
 - **bot** New help command
 
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
 - **bot** Fix for discordgo change on `GuildCreate.Guild.Unavaible`
 - **bot** Fix for discord api auth change (prepend `Bot ` to token)
 - **bot** Fixed play queue that was not thread safe
 - **bot** Fixed EventSource for nginx

 [Unreleased]: https://github.com/hammerandchisel/airhornbot/compare/master...Shywim:master
 