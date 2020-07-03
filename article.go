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
	"crypto/md5"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools/v2"
)

func processArticle(w *mwclient.Client, title, text, revTS, curTS string, regexes map[*regexp.Regexp]STRegex, test bool, attempt int8) {
	var articlePrepend strings.Builder
	var articleAppend strings.Builder
	var detected []string

	// Check for and respect nobots before we do anything else
	if ybtools.BotAllowed(text) {
		for regex, rsetup := range regexes {
			match := regex.FindStringSubmatchIndex(text)
			if match == nil {
				continue
			}

			// make sure that there are no matches of NoTagIf
			if rsetup.UseNTI && rsetup.NoTagIf.MatchString(text) {
				// match found; ignore this regex
				continue
			}

			var edited bool

			if rsetup.Prefix != "" {
				edited = tagIfNeeded(&articlePrepend, regex, rsetup, text, match)
			}
			if rsetup.Suffix != "" {
				// set edited true if tagIfNeeded returns true; else, leave the previous value
				edited = tagIfNeeded(&articleAppend, regex, rsetup, text, match) || edited
			}

			if edited {
				detected = append(detected, rsetup.Detected)
			}
		}

		if articlePrepend.Len() != 0 || articleAppend.Len() != 0 {
			// there's something to edit!
			var summaryBuilder strings.Builder
			var detectedBits string = strings.Join(detected, "; ")
			if test {
				summaryBuilder.WriteString("SANDBOX: ")
			}
			summaryBuilder.WriteString("[[User:Yapperbot/Scantag|Scantag]] detected ")
			summaryBuilder.WriteString(detectedBits)
			summaryBuilder.WriteString(". Tagging article.")
			prependText := articlePrepend.String()
			appendText := articleAppend.String()

			// don't edit limit tests - they should never be in anything other than userspace
			if test || ybtools.CanEdit() {
				if prependText != "" {
					text = insertMaintenanceTemplate(text, prependText)
				}
				if appendText != "" {
					text = text + appendText
				}

				err := w.Edit(params.Values{
					"title":          title,
					"summary":        summaryBuilder.String(),
					"bot":            "true",
					"basetimestamp":  revTS,
					"starttimestamp": curTS,
					"text":           text,
					"md5":            fmt.Sprintf("%x", md5.Sum([]byte(text))),
				})
				if err == nil {
					log.Println("Edited", title, "with", detectedBits)
					time.Sleep(10 * time.Second)
				} else {
					switch err.(type) {
					case mwclient.APIError:
						switch err.(mwclient.APIError).Code {
						case "noedit", "writeapidenied", "blocked":
							ybtools.PanicErr("noedit/writeapidenied/blocked code returned, the bot may have been blocked. Dying")
						case "pagedeleted":
							log.Println("Page", title, "was deleted before we could get to it")
						case "protectedpage":
							log.Println("Page", title, "is protected; we detected", detectedBits)
							// future TODO: post to talk page, maybe?
						case "editconflict":
							if attempt < 3 {
								processArticle(w, title, text, revTS, curTS, regexes, test, attempt+1)
								return
							}
							// we've already tried three times, we've edit conflicted every time
							// not worth it, just ignore the article for now; we'll come back later
							return
						default:
							log.Println("Error editing page", title, ". The error was", err)
						}
					default:
						ybtools.PanicErr("Non-API error returned when trying to write to page ", title, " so dying. Error was ", err)
					}
				}
			} else {
				ybtools.PanicErr("Edit limited out, stopping")
			}
		}
	}
}

func tagIfNeeded(builder *strings.Builder, regex *regexp.Regexp, rsetup STRegex, text string, match []int) bool {
	// ${n} returns the nth capture group, 1-indexed
	// $$ returns a literal $
	formatted := regex.ExpandString([]byte{}, rsetup.Prefix, text, match)

	// make sure we don't tag an article more than once
	if strings.Contains(text, string(formatted)) {
		return false
	}

	builder.Write(formatted)
	return true
}
