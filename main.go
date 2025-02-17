//    Copyright (C) 2025 Martin Spiering
//
//    This program is free software: you can redistribute it and/or modify
//    it under the terms of the GNU General Public License as published by
//    the Free Software Foundation, either version 3 of the License, or
//    (at your option) any later version.
//
//    This program is distributed in the hope that it will be useful,
//    but WITHOUT ANY WARRANTY; without even the implied warranty of
//    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//    GNU General Public License for more details.
//
//    You should have received a copy of the GNU General Public License
//    along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

var (
	Token  = os.Getenv("DISCORD_TOKEN")
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	loc    *time.Location
)

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		logger.Error("Error creating Discord session,", "error", err)
		os.Exit(1)
	}

	// Get the local time zone
	loc, err = time.LoadLocation("Local")
	if err != nil {
		logger.Error("Error loading local time zone", "error", err)
		os.Exit(1)
	}

	// Add a handler for the "ready" event to confirm the bot is online.
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logger.Info("Bot is up!")
	})

	// Add a handler for the command interaction
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			if i.ApplicationCommandData().Name == "sendlater" {
				options := i.ApplicationCommandData().Options
				message := ""
				sendTime := ""
				attachment := ""
				date := ""
				var channel *discordgo.Channel

				// we get the options set by the user
				for _, option := range options {
					if option.Name == "message" {
						message = option.StringValue()
					} else if option.Name == "time" {
						sendTime = option.StringValue()
					} else if option.Name == "date" {
						date = option.StringValue()
					} else if option.Name == "channel" {
						channel = option.ChannelValue(s)
					} else if option.Name == "attachment" {
						// we get the attachment url and then we download it
						attachmentID := option.Value.(string)
						if attachmentID == "" {
							continue
						}
						attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL
						resp, err := http.Get(attachmentUrl)
						if err != nil {
							slog.Error("Could not get attachment", "error", err, "url", attachmentUrl)
							s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseChannelMessageWithSource,
								Data: &discordgo.InteractionResponseData{
									Content: "Could not get attachment: " + err.Error(),
								},
							})
							return
						}
						if strings.Contains(resp.Header.Get("Content-type"), "plain/text") {
							slog.Error("Attachment is not text", "content-type", resp.Header.Get("Content-type"), "url", attachmentUrl)
							s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseChannelMessageWithSource,
								Data: &discordgo.InteractionResponseData{
									Content: "Could not get attachment, attachment is not text but " + resp.Header.Get("Content-type"),
								},
							})
							return
						}
						attachmentBytes, err := io.ReadAll(resp.Body)
						if err != nil {
							slog.Error("Could not get attachment", "error", err, "url", attachmentUrl)
							s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseChannelMessageWithSource,
								Data: &discordgo.InteractionResponseData{
									Content: "Could not get attachment: " + err.Error(),
								},
							})
							return
						}
						attachment = string(attachmentBytes)
					}
				}

				// if the channel wasn't set by the user, we get the current channel
				if channel == nil {
					channel, err = s.Channel(i.ChannelID)
					if err != nil {
						logger.Error("Error scheduling message: ", "error", err)
						s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Content: "Error scheduling message: " + err.Error(),
							},
						})
						return
					}
				}

				// if the date wasn't set by the user, we get the current date
				if date == "" {
					date = time.Now().Format("02/01/2006")
				}

				// we check that at least message or attachment is set but not both
				if message == "" && attachment == "" {
					logger.Error("Error scheduling message: ", "error", "message and attachment cannot be empty")
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Error scheduling message: message and attachment cannot be empty",
						},
					})
					return
				}

				if message != "" && attachment != "" {
					logger.Error("Error scheduling message: ", "error", "message and attachment cannot be both set")
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Error scheduling message: message and attachment cannot be both set",
						},
					})
					return
				}

				// we schedule the message
				err := scheduleMessage(s, message, attachment, sendTime, date, channel)
				if err != nil {
					logger.Error("Error scheduling message: ", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Error scheduling message: " + err.Error(),
						},
					})
					return
				}
				logger.Info("Message scheduled\n", "message", message+attachment, "date", date, "sendTime", sendTime, "channel", channel.Name)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Message scheduled!",
					},
				})
				return
			}
		}
	})

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		logger.Error("Error opening Discord session,", "error", err)
		os.Exit(1)
	}

	// Register the command
	cmd, err := registerCommand(dg, "sendlater")
	if err != nil {
		logger.Error("Error registering command,", "error", err, "command", cmd.Name)
		os.Exit(1)
	}

	// watch for interruption and gracefully shut down
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	logger.Info("Press Ctrl+C to exit")
	<-stop
	logger.Info("Removing commands...")

	err = dg.ApplicationCommandDelete(dg.State.User.ID, "", cmd.ID)
	if err != nil {
		logger.Error("Cannot delete command", "error", err, "command", cmd.Name)
	}

	logger.Info("Gracefully shutting down.")
}

func registerCommand(s *discordgo.Session, commandName string) (*discordgo.ApplicationCommand, error) {
	// Create a new command
	command := &discordgo.ApplicationCommand{
		Name:        commandName,
		Description: "Schedules a message (one line) or an attachment (several lines) to be sent at a later time. If time is set in the past, the message will be sent after one minute",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "time",
				Description: "The time to send the message (HH:MM)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message",
				Description: "The message to send (one line)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        "attachment",
				Description: "The message to send (several lines)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "date",
				Description: "[Optionnal] The date to send the message (dd/mm/yyyy). Default: today",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "channel",
				Description: "[Optionnal] Channel to send the message. Default: current channel",
				Required:    false,
			},
		},
	}

	// Register the command
	cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", command)
	if err != nil {
		logger.Error("Error creating command,", "error", err)

	} else {
		logger.Info("Command registered successfully!")
	}
	return cmd, nil
}

func scheduleMessage(s *discordgo.Session, message string, attachment string, sendTime string, date string, channel *discordgo.Channel) error {
	// Define the fixed time when the message should be sent.
	toSend := ""
	fixedTime, err := time.ParseInLocation("02/01/2006 15:04", date+" "+sendTime, loc)
	if err != nil {
		return errors.New("Error parsing fixed time: " + err.Error())
	}
	logger.Info("Time parsed", "time", fixedTime)
	if message != "" {
		toSend = message
	} else {
		toSend = attachment
	}
	go func() {
		// Use a ticker to periodically check the current time.
		ticker := time.NewTicker(time.Minute)
		//ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				if fixedTime.Before(now) {
					// Send a message to the specified channel.
					logger.Info("Sending message", "message", toSend, "channel", channel.Name)
					_, err := s.ChannelMessageSend(channel.ID, toSend)
					if err != nil {
						logger.Error("Error sending message,", "error", err)
					}
					return
				}
			}
		}
	}()
	return nil
}
