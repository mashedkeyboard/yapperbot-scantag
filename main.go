package main

//
// Yapperbot-Scantag, the page scanning and tagging bot for Wikipedia
// Copyright (C) 2020 Naypta

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//

import (
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools/v2"
)

var config Config
var regexes = map[*regexp.Regexp]STRegex{}

func init() {
	ybtools.SetupBot(ybtools.BotSettings{TaskName: "Scantag", BotUser: "Yapperbot"})
	ybtools.ParseTaskConfig(&config)
}

func main() {
	// Create a client here with significantly laxer Maxlag params
	w := ybtools.CreateAndAuthenticateClient(mwclient.Maxlag{
		On:      true,
		Timeout: "3",
		Retries: 10,
	})

	regexesJSON := ybtools.LoadJSONFromPageID(config.RegexesPageID)

	for regex, content := range regexesJSON.Map() {
		value, err := content.Object()
		if err != nil {
			ybtools.PanicErr("Scantag.json is invalid! Error was ", err)
		}

		expression, err := regexp.Compile("(?i)" + regex)
		if err != nil {
			ybtools.PanicErr("Regex `"+regex+"` is invalid! Error was ", err)
		}

		detected, err := value.GetString("detected")
		if err != nil {
			ybtools.PanicErr("Scantag.json is invalid, no detected string retrieved for `", regex, "` - error was ", err)
		}

		prefix, _ := value.GetString("prefix")
		suffix, _ := value.GetString("suffix")
		regexes[expression] = STRegex{
			Detected: detected,
			Prefix:   prefix,
			Suffix:   suffix,
		}
	}

	file, err := os.Open(config.PathToArticles)
	if err != nil {
		ybtools.PanicErr("Failed to open PathToArticles with error ", err)
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		ybtools.PanicErr("Failed to create gzip reader with error ", err)
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)

	// Ignore the first line; it's a header, page_title
	scanner.Scan()

	for scanner.Scan() {
		var queueEdit bool
		var articlePrepend strings.Builder
		var articleAppend strings.Builder
		var detected []string
		var title string = scanner.Text()

		log.Println(title)

		text, err := ybtools.FetchWikitextFromTitle(title)
		if err != nil {
			if apierr, ok := err.(mwclient.APIError); ok && apierr.Code == "missingtitle" {
				// just ignore it; probably deleted
				continue
			}
			log.Println("Failed to fetch wikitext from title", title, "with error", err)
			continue
		}

		for regex, rsetup := range regexes {
			match := regex.FindStringSubmatch(text)
			if match == nil {
				continue
			}
			queueEdit = true
			detected = append(detected, rsetup.Detected)

			// Because we have an array of strings, not an array of interfaces,
			// despite the fact that strings obviously implement interface,
			// we have to convert the string array to an interface array here.
			// I don't like this, at all, but it's the way it works... sadly
			// See https://stackoverflow.com/questions/12753805/type-converting-slices-of-interfaces
			// and https://stackoverflow.com/questions/30588581/how-to-pass-variable-parameters-to-sprintf-in-golang

			// Ignore the first key, as that's the full string match, which we don't need at all
			interfaceMatch := make([]interface{}, len(match)-1)
			for i := 1; i < len(match); i++ {
				interfaceMatch[i-1] = match[i]
			}
			if rsetup.Prefix != "" {
				articlePrepend.WriteString(fmt.Sprintf(rsetup.Prefix, interfaceMatch...))
			}
			if rsetup.Suffix != "" {
				articleAppend.WriteString(fmt.Sprintf(rsetup.Suffix, interfaceMatch...))
			}
		}

		if queueEdit {
			var summaryBuilder strings.Builder
			summaryBuilder.WriteString("[[User:Yapperbot/Scantag|Scantag]] detected ")
			summaryBuilder.WriteString(strings.Join(detected, "; "))
			summaryBuilder.WriteString(". Tagging article.")
			prependText := articlePrepend.String()
			appendText := articleAppend.String()
			if ybtools.CanEdit() {
				w.Edit(params.Values{
					"title":       title,
					"summary":     summaryBuilder.String(),
					"bot":         "true",
					"prependtext": prependText,
					"appendtext":  appendText,
					"md5":         fmt.Sprintf("%x", md5.Sum([]byte(prependText+appendText))),
				})
			}
		}
	}
}
