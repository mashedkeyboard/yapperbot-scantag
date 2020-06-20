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
	"flag"
	"log"
	"os"
	"regexp"

	"cgt.name/pkg/go-mwclient"
	"github.com/mashedkeyboard/ybtools/v2"
)

var config Config
var regexes = map[*regexp.Regexp]STRegex{}
var testTitle string
var sandbox bool

func init() {
	ybtools.SetupBot(ybtools.BotSettings{TaskName: "Scantag", BotUser: "Yapperbot"})
	ybtools.ParseTaskConfig(&config)

	flag.StringVar(&testTitle, "test", "", "Test the regexes against a single title, rather than over all pages. Default is an empty string.")
	flag.BoolVar(&sandbox, "sandbox", false, "Update the sandbox, rather than doing a run of the bot")
}

func main() {
	flag.Parse()

	// Create a client here with significantly laxer Maxlag params
	w := ybtools.CreateAndAuthenticateClient(mwclient.Maxlag{
		On:      true,
		Timeout: "3",
		Retries: 10,
	})
	if sandbox {
		createSandbox(w)
	} else {
		// If we edit-limit out, ybtools panics. This means defers are run,
		// so edit-limiting should still work.
		defer ybtools.SaveEditLimit()

		for {
			log.Println("Retrieving regexes")

			regexesJSON := ybtools.LoadJSONFromPageID(config.RegexesJSONPageID)

			for regex, content := range regexesJSON.Map() {
				expression, stregex, _, err := processRegex(regex, content)
				if err != nil {
					ybtools.PanicErr(err)
				}
				regexes[expression] = stregex
				log.Println("Found regex:", regex)
			}

			log.Println("Starting processing")

			if testTitle == "" {
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

				// at the point at which Wikipedia has more articles than can fit in a uint64, well, this will be fairly obsolete anyway >:)
				var n uint64

				for scanner.Scan() {
					if n%100 == 0 {
						log.Println("Processed", n, "articles so far this run")
					}
					processArticle(w, scanner.Text(), regexes, false, 0)
					n++
				}
			} else {
				processArticle(w, testTitle, regexes, true, 0)
			}

			log.Println("Completed processing, restarting")
		}
	}
}
