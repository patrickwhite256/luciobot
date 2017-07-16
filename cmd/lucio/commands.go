package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/patrickwhite256/luciobot"
)

var handlers = map[string]func(*discordgo.User, *discordgo.Channel, *discordgo.Guild, *string){
	"stop":   handleStop,
	"play":   handlePlay,
	"volume": handleVolume,
	"skip":   handleSkip,
}

func handleStop(user *discordgo.User, channel *discordgo.Channel, guild *discordgo.Guild, arg *string) {
	sendMessage("Skipping all video!", channel)
	// drain queue
	more := true
	for more {
		select {
		case <-queues[guild.ID]:
		default:
			more = false
		}

	}
	skips[guild.ID] <- true
}

func handlePlay(user *discordgo.User, channel *discordgo.Channel, guild *discordgo.Guild, arg *string) {
	if arg == nil {
		sendMessage("You gotta tell me a video to play!", channel)
		return
	}

	info := luciobot.GetVideoInfo(*arg)
	if info == nil {
		sendMessage("Couldn't find any video from that!", channel)
		return
	} else {
		sendMessage(fmt.Sprintf("Queuing `%s`!", info.Title), channel)
	}
	err := luciobot.DownloadVideo(info)
	if err != nil {
		sendMessage(fmt.Sprintf("Ran into an error downloading `%s`!", info.Title), channel)
	}

	var s luciobot.Sound
	err = s.Load(fmt.Sprintf("%s.mp4", info.ID))
	if err != nil {
		sendMessage(fmt.Sprintf("Ran into an error converting `%s` to audio!", info.Title), channel)
	}

	go enqueuePlay(user, guild, &s)
}

func handleVolume(user *discordgo.User, channel *discordgo.Channel, guild *discordgo.Guild, arg *string) {
	// TODO
}

func handleSkip(user *discordgo.User, channel *discordgo.Channel, guild *discordgo.Guild, arg *string) {
	sendMessage("Skipping this video!", channel)
	skips[guild.ID] <- true
}
