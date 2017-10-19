/*
 * Copyright (c) 2017 Ethan Wells
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dwarvesf/glod"
	"github.com/dwarvesf/glod/chiasenhac"
	"github.com/dwarvesf/glod/facebook"
	"github.com/dwarvesf/glod/soundcloud"
	"github.com/dwarvesf/glod/vimeo"
	"github.com/dwarvesf/glod/youtube"
	"github.com/dwarvesf/glod/zing"
	"github.com/jonas747/dca"
	"github.com/rylio/ytdl"
)

const (
	initZingMp3    string = "zing"
	initYoutube    string = "youtube"
	initSoundCloud string = "soundcloud"
	initChiaSeNhac string = "chiasenhac"
	initFacebook   string = "facebook"
	initVimeo      string = "vimeo"
)

var blocker chan bool = make(chan bool, 1)

type ObjectResponse struct {
	Resp *http.Response
	Name string
}

func NStream(videoURL, guildID, channelID string, s *discordgo.Session) error {
	// Change these accordingly
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	videoInfo, err := ytdl.GetVideoInfo(videoURL)
	if err != nil {
		return err
	}

	format := videoInfo.Formats.Extremes(ytdl.FormatAudioBitrateKey, true)[0]
	downloadURL, err := videoInfo.GetDownloadURL(format)
	if err != nil {
		return err
	}

	encodingSession, err := dca.EncodeFile(downloadURL.String(), options)
	if err != nil {
		return err
	}
	defer encodingSession.Cleanup()

	defer func() {
		<-blocker
	}()
	blocker <- true
	var vc *discordgo.VoiceConnection
	vc, err = s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}
	defer func() {
		vc.Speaking(false)
		vc.Disconnect()
	}()

	vc.Speaking(true)
	done := make(chan error)
	dca.NewStream(encodingSession, vc, done)
	err = <-done
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func Stream(link string, stop *bool, err *error, s *discordgo.Session, guildID, channelID string) {
	defer func() {
		*stop = false
	}()
	var ggl glod.Source
	if ggl = func() glod.Source {
		switch {
		case strings.Contains(link, initZingMp3):
			return &zing.Zing{}
		case strings.Contains(link, initYoutube):
			return &youtube.Youtube{}
		case strings.Contains(link, initSoundCloud):
			return &soundcloud.SoundCloud{}
		case strings.Contains(link, initChiaSeNhac):
			return &chiasenhac.ChiaSeNhac{}
		case strings.Contains(link, initFacebook):
			return &facebook.Facebook{}
		case strings.Contains(link, initVimeo):
			return &vimeo.Vimeo{}
		}
		return nil
	}(); ggl == nil {
		*err = errors.New("source link read problem")
		return
	}

	var objs []ObjectResponse
	listResponse, e := ggl.GetDirectLink(link)
	if e != nil {
		*err = e
		return
	}
	for _, r := range listResponse {
		temp := r.StreamURL
		if strings.Contains(link, initYoutube) || strings.Contains(link, initZingMp3) || strings.Contains(link, initVimeo) {
			splitUrl := strings.Split(temp, "~")
			temp = splitUrl[0]
		}

		resp, e := http.Get(temp)
		if e != nil {
			*err = errors.New("failed to get response from stream")
			return
		}

		fullName := fmt.Sprintf("%s%s", r.Title, ".mp3")

		fullName = strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, fullName)

		objs = append(objs, ObjectResponse{
			resp,
			fullName,
		})
	}

	var (
		vc *discordgo.VoiceConnection
	)

	vc.Speaking(true)
	for _, o := range objs {
		func() {
			defer o.Resp.Body.Close()
			opts := dca.StdEncodeOptions
			opts.RawOutput = true
			opts.Bitrate = 120

			encoder, e := dca.EncodeMem(o.Resp.Body, opts)
			if e != nil {
				*err = errors.New("encoding bork")
				return
			}

			for !*stop {
				frame, e := encoder.OpusFrame()
				if e != nil {
					if e != io.EOF {
						*err = errors.New("connection bork 1")
						return
					}

					break
				}

				select {
				case vc.OpusSend <- frame:
				case <-time.After(time.Second):
					*err = errors.New("connection bork 2")
					return
				}
			}
		}()
	}
	*err = errors.New("Done")
}
