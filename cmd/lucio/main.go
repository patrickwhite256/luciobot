package main

import (
	"flag"
	"os"
	"os/signal"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/patrickwhite256/luciobot"
)

var (
	demoSound *luciobot.Sound

	// discordgo session
	discord *discordgo.Session

	// Map of Guild id's to *Play channels, used for queuing and rate-limiting guilds
	queues map[string]chan *Play = make(map[string]chan *Play)

	// Sound encoding settings
	BITRATE        = 128
	MAX_QUEUE_SIZE = 6
)

// Play represents an individual play
type Play struct {
	GuildID   string
	ChannelID string
	UserID    string
	Sound     *luciobot.Sound

	// The next play to occur after this
	Next *Play
}

// Attempts to find the current users voice channel inside a given guild
func getCurrentVoiceChannel(user *discordgo.User, guild *discordgo.Guild) *discordgo.Channel {
	for _, vs := range guild.VoiceStates {
		if vs.UserID == user.ID {
			channel, _ := discord.State.Channel(vs.ChannelID)
			return channel
		}
	}
	return nil
}

// Prepares a play
func createPlay(user *discordgo.User, guild *discordgo.Guild, sound *luciobot.Sound) *Play {
	// Grab the users voice channel
	channel := getCurrentVoiceChannel(user, guild)
	if channel == nil {
		log.WithFields(log.Fields{
			"user":  user.ID,
			"guild": guild.ID,
		}).Warning("Failed to find channel to play sound in")
		return nil
	}

	// Create the play
	play := &Play{
		GuildID:   guild.ID,
		ChannelID: channel.ID,
		UserID:    user.ID,
		Sound:     sound,
	}

	return play
}

// Prepares and enqueues a play into the ratelimit/buffer guild queue
func enqueuePlay(user *discordgo.User, guild *discordgo.Guild, sound *luciobot.Sound) {
	play := createPlay(user, guild, sound)
	if play == nil {
		return
	}

	// Check if we already have a connection to this guild
	//   yes, this isn't threadsafe, but its "OK" 99% of the time
	_, exists := queues[guild.ID]

	if exists {
		if len(queues[guild.ID]) < MAX_QUEUE_SIZE {
			queues[guild.ID] <- play
		}
	} else {
		queues[guild.ID] = make(chan *Play, MAX_QUEUE_SIZE)
		playSound(play, nil)
	}
}

func playSound(play *Play, vc *discordgo.VoiceConnection) (err error) {
	log.WithFields(log.Fields{
		"play": play,
	}).Info("Playing sound")

	if vc == nil {
		vc, err = discord.ChannelVoiceJoin(play.GuildID, play.ChannelID, false, false)
		// vc.Receive = false
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Failed to play sound")
			delete(queues, play.GuildID)
			return err
		}
	}

	// If we need to change channels, do that now
	if vc.ChannelID != play.ChannelID {
		vc.ChangeChannel(play.ChannelID, false, false)
		time.Sleep(time.Millisecond * 125)
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(time.Millisecond * 32)

	// Play the sound
	play.Sound.Play(vc)

	// If this is chained, play the chained sound
	if play.Next != nil {
		playSound(play.Next, vc)
	}

	// If there is another song in the queue, recurse and play that
	if len(queues[play.GuildID]) > 0 {
		play := <-queues[play.GuildID]
		playSound(play, vc)
		return nil
	}

	// If the queue is empty, delete it
	time.Sleep(time.Millisecond * time.Duration(250))
	delete(queues, play.GuildID)
	vc.Disconnect()
	return nil
}

//////////////////////
// discord handlers //
//////////////////////

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

	go enqueuePlay(m.Author, guild, demoSound)
}

func main() {
	var (
		Token = flag.String("t", "", "Discord Authentication Token")
		err   error
	)
	flag.Parse()

	var s luciobot.Sound
	err = s.Load("audio/atlas.mp4")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create sound")
		return
	}

	demoSound = &s

	discord, err = discordgo.New("Bot " + *Token)
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
