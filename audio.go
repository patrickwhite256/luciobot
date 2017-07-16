package luciobot

import (
	"fmt"
	"os"

	"github.com/rylio/ytdl"
)

func GetVideoInfo(videoUrl string) *ytdl.VideoInfo {
	fmt.Println(videoUrl)
	info, err := ytdl.GetVideoInfo(videoUrl)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return info
}

func DownloadVideo(info *ytdl.VideoInfo) error {
	fo, err := os.OpenFile(fmt.Sprintf("%s.mp4", info.ID), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil
	}

	defer fo.Close()
	err = info.Download(info.Formats.Best(ytdl.FormatAudioEncodingKey)[0], fo)
	if err != nil {
		return nil
	}

	return nil
}
