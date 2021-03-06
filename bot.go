/**
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
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	YoutubeKey = "https://developers.google.com"
	prefix     = ";"
	token      string
	dg         *discordgo.Session
	total      int = 0
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.StringVar(&YoutubeKey, "y", "", "Youtube token")
	flag.Parse()
}

func main() {
	err := LoadCalenders()
	if err != nil {
		fmt.Println(err)
	}

	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			<-t.C
			rand.Seed(time.Now().Unix())
		}
	}()

	if YoutubeKey == "" {
		YoutubeKey = DevKey()
	}

	if token == "" {
		token = Token()
	}

	if token == "" {
		fmt.Println("No token provided. Please set DISCORD_TOKEN to the appropriate token")
		return
	}

	dg, err = discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(guildCreate)
	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// go alertEvents()

	fmt.Println("The bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
	os.Exit(0)
}

func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == "315552571823489024" {
			_, _ = s.ChannelMessageSend(channel.ID, "Connected.")
			return
		}
	}
}

func Token() (token string) {
	token = os.Getenv("DISCORD_TOKEN")
	if (strings.Contains(token, "$") || token == "") && len(os.Args) > 1 {
		token = os.Args[1]
	} else {
		b, err := ioutil.ReadFile("/token.txt")
		if err != nil {
			fmt.Println(err)
		}
		token = strings.TrimSpace(strings.Trim(strings.Split(string(b), ";")[0], "\n"))
	}
	return
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, prefix+"help | @xkcd help")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(m.Message.Content) < 1 {
		return
	}
	if m.Message.ContentWithMentionsReplaced()[:len(prefix)] != prefix &&
		strings.Replace(m.Message.Content, s.State.User.Mention(), "", 1) == m.Message.Content {
		return
	}

	c := strings.Split(strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(m.Message.Content, prefix), s.State.User.Mention())), " ")
	c[0] = strings.ToLower(c[0])

	if c[0] == "ping" {
		s.ChannelMessageSend(m.ChannelID, "pong!")
	} else if c[0] == "help" {
		var e *discordgo.MessageEmbed
		e = &discordgo.MessageEmbed{
			Title:       "Help",
			Description: "How to use this here xkcd bot",
			URL:         "https://xkcd.com/",
			Color:       7506394,
			Type:        "rich",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "help",
					Value: "Display this message",
				},
				{
					Name:  "xkcd <comic number, name, regex or whatever>",
					Value: "Get the designated comic",
				},
				{
					Name:  "latest",
					Value: "The latest and greatest xkcd",
				},
				{
					Name:  "random",
					Value: "A random comic",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "@" + m.Author.String(),
				IconURL: "https://cdn.discordapp.com/avatars/" + m.Author.ID + "/" + m.Author.Avatar + ".png",
			},
		}

		if len(c) > 1 {
			if c[1] == "event" {
				e = &discordgo.MessageEmbed{
					Title:       "Help",
					Description: "How to use `event` commands on this here xkcd bot",
					URL:         "https://xkcd.com/",
					Color:       7506394,
					Type:        "rich",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  TimeFormat,
							Value: "When creating events, be sure to use this format for time",
						},
						{
							Name:  "new <title>; <description>; <participants (mention them)>; <date/time>",
							Value: "Create a new event. Note that you can use spaces in the fields, but the fields are seperated by semicolons",
						},
						{
							Name:  "list",
							Value: "List the events that exist",
						},
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text:    "@" + m.Author.String(),
						IconURL: "https://cdn.discordapp.com/avatars/" + m.Author.ID + "/" + m.Author.Avatar + ".png",
					},
				}
			}
		}

		_, err := s.ChannelMessageSendEmbed(m.ChannelID, e)
		if err != nil {
			fmt.Println(err)
		}
	} else if c[0] == "xkcd" {
		if len(c) < 2 {
			c = append(c, "")
		}
		var (
			xkcd XKCD
			err  error
			r    *regexp.Regexp
		)

		r, err = regexp.Compile("^[0-9]+$")
		if err != nil {
			fmt.Println(err)
			return
		}

		if r.MatchString(c[1]) {
			xkcd, err = GetXkcdNum(c[1])
			if err != nil {
				fmt.Println(err)
			}
		} else {
			xkcd, err = GetXkcdTitle(strings.Join(c[1:], "\\s"))
		}
		if err != nil {
			fmt.Println(err)
			return
		}

		e := &discordgo.MessageEmbed{
			Title:       "xkcd #" + strconv.Itoa(xkcd.Num) + ": " + xkcd.Title,
			Description: xkcd.Alt,
			URL:         "https://xkcd.com/" + strconv.Itoa(xkcd.Num),
			Color:       7506394,
			Type:        "rich",
			Image: &discordgo.MessageEmbedImage{
				URL: xkcd.Img,
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "@" + m.Author.String(),
				IconURL: "https://cdn.discordapp.com/avatars/" + m.Author.ID + "/" + m.Author.Avatar + ".png",
			},
		}

		_, err = s.ChannelMessageSendEmbed(m.ChannelID, e)
		if err != nil {
			fmt.Println(err)
			s.ChannelMessageSend(m.ChannelID, xkcd.Img)
		}
	} else if c[0] == "latest" {
		xkcd, err := GetLatest()
		if err != nil {
			fmt.Println(err)
		}
		total = xkcd.Num

		e := &discordgo.MessageEmbed{
			Title:       "xkcd #" + strconv.Itoa(xkcd.Num) + ": " + xkcd.Title,
			Description: xkcd.Alt,
			URL:         "https://xkcd.com/" + strconv.Itoa(xkcd.Num),
			Color:       7506394,
			Type:        "rich",
			Image: &discordgo.MessageEmbedImage{
				URL: xkcd.Img,
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "@" + m.Author.String(),
				IconURL: "https://cdn.discordapp.com/avatars/" + m.Author.ID + "/" + m.Author.Avatar + ".png",
			},
		}

		_, err = s.ChannelMessageSendEmbed(m.ChannelID, e)
		if err != nil {
			fmt.Println(err)
			s.ChannelMessageSend(m.ChannelID, xkcd.Img)
		}
		return
	} else if c[0] == "random" {
		num := rand.Intn(total)
		xkcd, err := GetXkcdNum(strconv.Itoa(num))

		e := &discordgo.MessageEmbed{
			Title:       "xkcd #" + strconv.Itoa(xkcd.Num) + ": " + xkcd.Title,
			Description: xkcd.Alt,
			URL:         "https://xkcd.com/" + strconv.Itoa(xkcd.Num),
			Color:       7506394,
			Type:        "rich",
			Image: &discordgo.MessageEmbedImage{
				URL: xkcd.Img,
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "@" + m.Author.String(),
				IconURL: "https://cdn.discordapp.com/avatars/" + m.Author.ID + "/" + m.Author.Avatar + ".png",
			},
		}

		_, err = s.ChannelMessageSendEmbed(m.ChannelID, e)
		if err != nil {
			fmt.Println(err)
			s.ChannelMessageSend(m.ChannelID, xkcd.Img)
		}
	} else if c[0] == "play" || c[0] == "search" {
		if strings.Count(m.ContentWithMentionsReplaced(), ";") > 1 {
			spl := strings.Split(m.ContentWithMentionsReplaced(), ";")
			for _, ts := range spl {
				nml := m
				nml.Content = ";" + ts
				go messageCreate(s, nml)
				time.Sleep(time.Second)
			}
			return
		}

		if len(c) < 2 {
			err := s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👎")
			if err != nil {
				fmt.Println(err)
			}
			return
		} else if len(c) > 2 && c[0] == "play" {
			for _, ci := range c[1:] {
				nml := &discordgo.MessageCreate{}
				*nml = *m
				nml.Content = ";play " + ci
				go messageCreate(s, nml)
				time.Sleep(time.Second)
			}
			return
		}

		var (
			err   error
			tch   *discordgo.Channel
			vch   *discordgo.VoiceState
			guild *discordgo.Guild
		)

		o := strings.Join(c[1:], " ")
		if c[0] == "play" {
			o = strings.TrimRight(strings.TrimLeft(o, "<"), ">")
		} else {
			o = strings.Replace(strings.TrimSpace(o), " ", "%20", -1)
			o, err = UrlFromSearch(o)
			if err != nil {
				fmt.Println("Search:", err)
				e := s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👎")
				if e != nil {
					fmt.Println("Reaction:", e)
				}
			}
		}

		fmt.Println("URL:", o)

		tch, err = s.Channel(m.Message.ChannelID)
		if err != nil {
			fmt.Println("Get channel:", err)
			return
		}
		guild, err = s.Guild(tch.GuildID)
		if err != nil {
			fmt.Println("Get guild:", err)
			return
		}

		for _, vs := range guild.VoiceStates {
			if vs.UserID == m.Author.ID {
				vch = vs
				break
			}
		}

		if vch == nil {
			fmt.Println("User not joined channel")
			return
		}

		err = s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👌")
		if err != nil {
			fmt.Println("Reaction:", err)
		}

		Stop[tch.GuildID] = false
		Streams[tch.GuildID] = &Streamer{
			Url:       o,
			GuildID:   tch.GuildID,
			ChannelID: vch.ChannelID,
			S:         s,
		}

		err = Streams[tch.GuildID].Stream()
		if err != nil {
			fmt.Println("Stream:", err)
			err = s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👎")
			if err != nil {
				fmt.Println("Reaction:", err)
			}
		}
	} else if c[0] == "skip" {
		tch, err := s.Channel(m.Message.ChannelID)
		if err != nil {
			fmt.Println(err)
			return
		}
		Streams[tch.GuildID] = nil
		Stop[tch.GuildID] = true
		exitQ = true
		<-blocker
		err = s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👌")
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(time.Second)
		exitQ = false
	} else if c[0] == "exit" {
		tch, err := s.Channel(m.Message.ChannelID)
		if err != nil {
			fmt.Println(err)
			return
		}
		Streams[tch.GuildID] = nil
		Stop[tch.GuildID] = true
		err = s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👌")
		if err != nil {
			fmt.Println(err)
		}
	} else if c[0] == "repeat" {
		tch, err := s.Channel(m.Message.ChannelID)
		if err != nil {
			fmt.Println(err)
			return
		}

		current := Streams[tch.GuildID]
		if current == nil {
			err = s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👎")
			if err != nil {
				fmt.Println(err)
			}
			return
		}

		err = s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👌")
		if err != nil {
			fmt.Println(err)
		}

		for {
			for !Stop[tch.GuildID] {
				time.Sleep(time.Second)
			}
			err = Streams[tch.GuildID].Stream()
			if err != nil {
				fmt.Println(err)
			}
		}
	} else if c[0] == "pause" {
		tch, err := s.Channel(m.Message.ChannelID)
		if err != nil {
			fmt.Println(err)
			return
		}

		pause[tch.GuildID] = true

		err = s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👌")
		if err != nil {
			fmt.Println(err)
		}
	} else if c[0] == "resume" {
		tch, err := s.Channel(m.Message.ChannelID)
		if err != nil {
			fmt.Println(err)
			return
		}

		pause[tch.GuildID] = false

		err = s.MessageReactionAdd(m.Message.ChannelID, m.ID, "👌")
		if err != nil {
			fmt.Println(err)
		}
	} else if c[0] == "event" {
		if len(c) < 2 {
			return
		}

		if c[1] == "new" {
			co := strings.Split(strings.Join(c[2:], " "), ";")
			for i, ci := range co {
				co[i] = strings.TrimSpace(ci)
			}
			if len(co) < 4 {
				return
			}
			f, erro := s.Channel(m.ChannelID)
			if erro != nil {
				fmt.Println(erro)
				return
			}
			if _, err := time.Parse(TimeFormat, co[3]); err != nil {
				co[3] = time.Now().Add(time.Hour * 24).Format(TimeFormat)
			}

			var (
				event *Calender
				err   error
			)
			if len(co) > 4 {
				event, err = NewCalender(co[0], co[1], co[2], co[3], f.GuildID, c[4], m.Author.ID)
			} else {
				event, err = NewCalender(co[0], co[1], co[2], co[3], f.GuildID, f.ID, m.Author.ID)
			}
			if err != nil {
				fmt.Println(err)
				return
			}
			e := &discordgo.MessageEmbed{
				Color: 7506394,
				Type:  "rich",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Event Created: " + event.Title,
						Value:  event.Description,
						Inline: true,
					},
					{
						Name:   "Time",
						Value:  event.Date,
						Inline: true,
					},
					{
						Name:   "Participants",
						Value:  event.Participants,
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text:    "@" + m.Author.String(),
					IconURL: "https://cdn.discordapp.com/avatars/" + m.Author.ID + "/" + m.Author.Avatar + ".png",
				},
			}

			_, err = s.ChannelMessageSendEmbed(m.ChannelID, e)
			if err != nil {
				fmt.Println(err)
			}
		} else if c[1] == "list" {
			var fields []*discordgo.MessageEmbedField
			for _, event := range Events {
				fields = append(fields,
					&discordgo.MessageEmbedField{
						Name:  event.Title,
						Value: event.Date,
					})
			}
			e := &discordgo.MessageEmbed{
				Color:  7506394,
				Type:   "rich",
				Fields: fields,
				Footer: &discordgo.MessageEmbedFooter{
					Text:    "@" + m.Author.String(),
					IconURL: "https://cdn.discordapp.com/avatars/" + m.Author.ID + "/" + m.Author.Avatar + ".png",
				},
			}

			_, err := s.ChannelMessageSendEmbed(m.ChannelID, e)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func alertEvents() {
	t := time.NewTicker(5 * time.Second)
	var e *discordgo.MessageEmbed
	for {
		for i, event := range Events {
			et, err := time.Parse(TimeFormat, event.Date)
			if err != nil {
				fmt.Println(err)
				continue
			}

			author, err := dg.User(event.AuthorID)
			if err != nil {
				fmt.Println(err)
				e = &discordgo.MessageEmbed{
					Color: 7506394,
					Type:  "rich",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   event.Title,
							Value:  event.Description,
							Inline: true,
						},
						{
							Name:   "Time",
							Value:  event.Date,
							Inline: true,
						},
						{
							Name:   "Participants",
							Value:  "@" + event.Participants,
							Inline: true,
						},
					},
				}
			} else if time.Now().After(et) {
				e = &discordgo.MessageEmbed{
					Color: 7506394,
					Type:  "rich",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   event.Title,
							Value:  event.Description,
							Inline: true,
						},
						{
							Name:   "Time",
							Value:  event.Date,
							Inline: true,
						},
						{
							Name:   "Participants",
							Value:  event.Participants,
							Inline: true,
						},
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text:    "@" + author.String(),
						IconURL: "https://cdn.discordapp.com/avatars/" + author.ID + "/" + author.Avatar + ".png",
					},
				}

				_, err := dg.ChannelMessageSendEmbed(event.ChannelID, e)
				if err != nil {
					fmt.Println(err)
				}
				Events = append(Events[:i], Events[i+1:]...)
				go SaveCalenders()
			}
		}
		<-t.C
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
