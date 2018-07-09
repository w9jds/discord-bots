package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/PuerkitoBio/goquery"
	"github.com/bwmarrin/discordgo"
	"github.com/gocolly/colly"
)

var (
	discord   *discordgo.Session
	collector *colly.Collector
)

func main() {

	collector = colly.NewCollector(
		colly.UserAgent("Beholder/v1 by Jeremy Shore w9jds@live.com"),
	)

	discord, error := discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
	if error != nil {
		fmt.Println("Error creating discord client: ", error)
		return
	}

	discord.AddHandler(ready)
	discord.AddHandler(messageCreate)

	error = discord.Open()
	if error != nil {
		fmt.Println("Error opening connection: ", error)
		return
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()

}

func ready(session *discordgo.Session, ready *discordgo.Ready) {
	fmt.Println("Beholder has arrived! Roll Initiative.")
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID || !strings.HasPrefix(message.Content, "!") {
		return
	}

	var results []int32
	var error error
	sequence := strings.SplitN(message.Content, " ", 2)

	if strings.HasPrefix(strings.ToLower(message.Content), "!roll") {
		switch sequence[1] {
		case "stats":
			results, error = processRolls("6d20")
			// for key, roll := range results {
			// 	results[key] = roll + 6
			// }
		default:
			results, error = processRolls(sequence[1])
		}

		if error != nil {
			session.ChannelMessageSend(message.ChannelID, error.Error())
			return
		}

		session.ChannelMessageSend(message.ChannelID, fmt.Sprint(results))
	}

	if strings.HasPrefix(strings.ToLower(message.Content), "!spell") {
		processSpell(message.ChannelID, session, strings.ToLower(sequence[1]))
	}
}

func processSpell(channelID string, session *discordgo.Session, name string) {
	spell := strings.Replace(name, "'", "", -1)
	spell = strings.Replace(name, " ", "-", -1)

	collector.OnError(func(_ *colly.Response, error error) {
		fmt.Println(error)
		// return nil, error
	})

	collector.OnHTML("body", func(e *colly.HTMLElement) {
		selector := e.DOM
		image, exists := selector.Find(".spell-image").Attr("src")

		message := discordgo.MessageEmbed{
			URL:         "https://www.dndbeyond.com/spells/" + spell,
			Title:       strings.TrimSpace(selector.Find("h1.page-title").Text()),
			Fields:      []*discordgo.MessageEmbedField{},
			Description: strings.TrimSpace(selector.Find(".more-info-content").Find("p").Text()),
			// Provider: &discordgo.MessageEmbedProvider{
			// 	URL:  "https://www.dndbeyond.com",
			// 	Name: "DnD Beyond",
			// },
			Footer: &discordgo.MessageEmbedFooter{
				IconURL: "https://www.dndbeyond.com/Content/1-0-377-0/Skins/Waterdeep/images/dnd-beyond-b-red.png",
				Text:    "DnD Beyond",
			},
		}

		if exists {
			if !strings.HasPrefix(strings.ToLower(image), "https:") {
				image = "https:" + image
			}

			message.Thumbnail = &discordgo.MessageEmbedThumbnail{
				URL: image,
			}
		}

		selector.Find("div.ddb-statblock-item").Each(func(i int, match *goquery.Selection) {
			message.Fields = append(message.Fields, &discordgo.MessageEmbedField{
				Name:   strings.TrimSpace(match.Find(".ddb-statblock-item-label").Text()),
				Value:  strings.TrimSpace(match.Find(".ddb-statblock-item-value").Text()),
				Inline: true,
			})
		})

		components := selector.Find(".components-blurb").Text()

		if components != "" {
			match := regexp.MustCompile(`\((.*?)\)`).FindStringSubmatch(components)
			message.Fields = append(message.Fields, &discordgo.MessageEmbedField{
				Name:   "Components",
				Value:  match[1],
				Inline: false,
			})
		}

		_, error := session.ChannelMessageSendEmbed(channelID, &message)
		if error != nil {
			session.ChannelMessageSend(channelID, error.Error())
		}
	})

	collector.Visit("https://www.dndbeyond.com/spells/" + spell)
}

func processRolls(roll string) ([]int32, error) {
	var rolls []int32
	var error error

	count := 1
	info := regexp.MustCompile("d|D").Split(roll, 3)

	if info[0] != "" {
		count, error = strconv.Atoi(info[0])
		if error != nil {
			return nil, error
		}
	}

	dice, error := strconv.Atoi(info[1])
	if error != nil {
		return nil, error
	}

	if dice != 4 && dice != 6 && dice != 8 && dice != 10 && dice != 12 && dice != 20 {
		return nil, errors.New("Invalid dice type, must be a d4, d6, d8, d10, d12 or d20")
	}

	for i := 0; i < count; i++ {
		rolls = append(rolls, getDice(int32(dice)))
	}

	return rolls, nil
}

func getDice(dice int32) int32 {
	max := dice - 1
	roll := rand.Int31n(max)

	return roll + 1
}

// c.OnRequest(func(r *colly.Request) {
//     fmt.Println("Visiting", r.URL)
// })

// c.OnError(func(_ *colly.Response, err error) {
//     log.Println("Something went wrong:", err)
// })

// c.OnResponse(func(r *colly.Response) {
//     fmt.Println("Visited", r.Request.URL)
// })

// c.OnHTML("a[href]", func(e *colly.HTMLElement) {
//     e.Request.Visit(e.Attr("href"))
// })

// c.OnHTML("tr td:nth-of-type(1)", func(e *colly.HTMLElement) {
//     fmt.Println("First column of a table row:", e.Text)
// })

// c.OnXML("//h1", func(e *colly.XMLElement) {
//     fmt.Println(e.Text)
// })

// c.OnScraped(func(r *colly.Response) {
//     fmt.Println("Finished", r.Request.URL)
// })
