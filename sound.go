package luciobot

import (
	"encoding/binary"
	"io"
	"os/exec"
	"strconv"

	"layeh.com/gopus"

	"github.com/bwmarrin/discordgo"
)

const (
	BITRATE    = 64
	CHANNELS   = 2
	FRAME_RATE = 48000
	FRAME_SIZE = 960
	MAX_BYTES  = CHANNELS * FRAME_SIZE * 2
)

type Sound struct {
	loaded bool
	buffer [][]byte
}

// Plays this sound over the specified VoiceConnection
func (s *Sound) Play(vc *discordgo.VoiceConnection) {
	vc.Speaking(true)
	defer vc.Speaking(false)

	for _, buff := range s.buffer {
		vc.OpusSend <- buff
	}
}

func (s *Sound) Load(file string) error {
	// load sound and convert to opus
	ffmpeg := exec.Command("ffmpeg", "-i", file, "-f", "s16le", "-ar", strconv.Itoa(FRAME_RATE), "-ac", strconv.Itoa(CHANNELS), "pipe:1")
	stdout, err := ffmpeg.StdoutPipe()
	if err != nil {
		return err
	}
	err = ffmpeg.Start()
	if err != nil {
		return err
	}

	encChan := make(chan []int16)
	done := make(chan bool)
	encoder, err := gopus.NewEncoder(FRAME_RATE, CHANNELS, gopus.Audio)
	if err != nil {
		return err
	}
	encoder.SetBitrate(BITRATE * 1000)
	encoder.SetApplication(gopus.Audio)
	go func() {
		for pcm := range encChan {
			frame, _ := encoder.Encode(pcm, FRAME_SIZE, MAX_BYTES)
			s.buffer = append(s.buffer, frame)
		}
		done <- true
	}()

	for {
		// read data from ffmpeg stdout
		buf := make([]int16, FRAME_SIZE*CHANNELS)
		err = binary.Read(stdout, binary.LittleEndian, &buf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return err
		}

		// write pcm data to the EncodeChan
		encChan <- buf
	}

	close(encChan)
	<-done
	s.loaded = true
	return nil
}
