# send-later-discord-bot

## Description

This is a Discord bot that allows you to send messages to a channel at a later time. It is written in go and uses the discordgo library.

## Installation

To install the bot, you will need to have go 1.23.6 installed on your computer. You will also need to have a Discord account and a server to invite the bot to.

1. Clone the repository to your computer.
2. Run `go build -o sendlater` in the root directory of the project.
3. Set the `DISCORD_TOKEN` environment variable to your bot's token.
4. Invite the bot to your server using the following URL: `https://discord.com/oauth2/authorize?client_id=YOUR_BOT_ID&scope=bot&permissions=2147483648`
5. Run `./sendlater` in the root directory of the project to start the bot.

## Usage

To use the bot, you will need to send a message to the bot in the following format:

```
/sendlater <channel> <date> <time> <message> <attachment>
```

Where `<channel>` is the name of the channel you want to send the message to, `<time>` is the time you want to send the message at in the format `HH:MM`, `<date>` is the date you want to send the message at in the format `dd/mm/yyyy` and `<message>` is the message you want to send. You can also choose to send an `<attachment>` instead of a `<message>`

- `<time>` is mandatory
- Exactly one of `<message>` or `<attachment>` is mandatory
- `<date>` is optional, if not provided, the message will be sent at the specified time on the current date.
- `<channel>` is optional, if not provided, the message will be sent to the channel the command was sent in.

For example, to send the message "Hello, world!" to the channel `#general` at 12:00 PM, you would send the following message to the bot:

```
/sendlater #general 12:00 "Hello, world!"
```

## License

This project is licensed under the GPLv3 License. See the LICENSE file for more information.
