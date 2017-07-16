package main

import (
	"flag"
	"os"
	"os/signal"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
)

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Info("Received READY payload")
	err := s.UpdateStatus(0, "Why are you so angry?")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Could not set status.")
	}
}

func onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		// send to #general
		if channel.ID == event.Guild.ID {
			_, err := s.ChannelMessageSend(channel.ID, "Lucio, coming at you!")
			if err != nil {
				log.WithFields(log.Fields{
					"channel": channel.Name,
					"error":   err,
				}).Warn("Could not send message to channel.")
			}
			return
		}
	}
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(m.Content) <= 0 || (m.Content[0] != '!') {
		return
	}

	// msg := strings.Replace(m.ContentWithMentionsReplaced(), s.State.Ready.User.Username, "username", 1)
	// parts := strings.Split(strings.ToLower(msg), " ")

	channel, _ := s.State.Channel(m.ChannelID)
	if channel == nil {
		log.WithFields(log.Fields{
			"channel": m.ChannelID,
			"message": m.ID,
		}).Warning("Failed to grab channel")
		return
	}

	guild, _ := s.State.Guild(channel.GuildID)
	if guild == nil {
		log.WithFields(log.Fields{
			"guild":   channel.GuildID,
			"channel": channel,
			"message": m.ID,
		}).Warning("Failed to grab guild")
		return
	}

	_, err := s.ChannelMessageSend(m.ChannelID, "I'm not hearing that noise!")
	if err != nil {
		log.WithFields(log.Fields{
			"channel": channel.Name,
			"error":   err,
		}).Warn("Could not send message to channel.")
	}
	// // Find the collection for the command we got
	// for _, coll := range COLLECTIONS {
	// 	if scontains(parts[0], coll.Commands...) {

	// 		// If they passed a specific sound effect, find and select that (otherwise play nothing)
	// 		var sound *Sound
	// 		if len(parts) > 1 {
	// 			for _, s := range coll.Sounds {
	// 				if parts[1] == s.Name {
	// 					sound = s
	// 				}
	// 			}

	// 			if sound == nil {
	// 				return
	// 			}
	// 		}

	// 		go enqueuePlay(m.Author, guild, coll, sound)
	// 		return
	// 	}
	// }
}

func main() {
	var (
		Token = flag.String("t", "", "Discord Authentication Token")
	)
	flag.Parse()

	discord, err := discordgo.New("Bot " + *Token)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord session")
		return
	}

	discord.AddHandler(onReady)
	discord.AddHandler(onGuildCreate)
	discord.AddHandler(onMessageCreate)

	err = discord.Open()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord websocket connection")
		return
	}

	log.Info("Bot ready")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
