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
	"fmt"
	"log"
	"regexp"
	"strings"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools/v2"
)

const sandboxTimestamp string = `{{/ts|%d|%s|%s}}`

// Leave the newlines on either side of the header; they're important.
const sandboxHeader string = `<!-- Remove the ts template to force the sandbox to be regenerated -->
{|class="wikitable"
|-
! Task !! Example !! Don't tag if matches !! Use noTagIf? !! Prefix the article with !! Suffix the article with !! Detected... !! Test page
`

// Leave the newline at the start of this; it's also important.
const sandboxFooter string = `
|}`

const sandboxTemplateOpening string = `|-
! colspan="8" | <code><nowiki>%s</nowiki></code>
|-
| `
const sandboxTemplateCode string = `<code><nowiki>%v</nowiki></code>`
const sandboxTemplateNoCode string = `<nowiki>%v</nowiki>`
const sandboxError string = `colspan="8" {{no O|<code><nowiki>%s</nowiki></code>}}`

func createSandbox(w *mwclient.Client) {
	sandboxMetaQuery, err := w.Get(params.Values{
		"action":  "query",
		"prop":    "revisions",
		"pageids": config.SandboxJSONPageID,
		"rvprop":  "ids|timestamp|user",
	})
	if err != nil {
		ybtools.PanicErr("Failed to fetch sandbox JSON metadata with error ", err)
	}

	sandboxMetaPages := ybtools.GetPagesFromQuery(sandboxMetaQuery)
	sandboxMeta, err := sandboxMetaPages[0].GetObjectArray("revisions")
	if err != nil {
		ybtools.PanicErr("Failed to get revisions from sandbox metadata with error ", err)
	}

	revid, err := sandboxMeta[0].GetInt64("revid")
	if err != nil {
		ybtools.PanicErr("Sandbox JSON revid invalid with error ", err)
	}
	user, err := sandboxMeta[0].GetString("user")
	if err != nil {
		ybtools.PanicErr("Sandbox JSON user invalid with error ", err)
	}
	ts, err := sandboxMeta[0].GetString("timestamp")
	if err != nil {
		ybtools.PanicErr("Sandbox JSON timestamp invalid with error ", err)
	}

	sandboxTS := fmt.Sprintf(sandboxTimestamp, revid, ts, user)

	sandboxJSON := ybtools.LoadJSONFromPageID(config.SandboxJSONPageID)

	sandboxPageText, err := ybtools.FetchWikitext(config.SandboxPageID)
	if err != nil {
		ybtools.PanicErr("Failed to fetch sandbox page text with error ", err)
	}

	if strings.HasPrefix(sandboxPageText, sandboxTS) {
		// No updates since the last time we ran; we can just end here
		log.Println("No sandbox changes to update")
		return
	}

	var sandboxBuilder strings.Builder

	sandboxBuilder.WriteString(sandboxTS)
	sandboxBuilder.WriteString(sandboxHeader)

	for regex, content := range sandboxJSON.Map() {
		sandboxBuilder.WriteString(fmt.Sprintf(sandboxTemplateOpening, regex))
		expr, stregex, testpage, err := processRegex(regex, content)
		if err != nil {
			sandboxBuilder.WriteString(fmt.Sprintf(sandboxError, err))
			continue
		}

		writeCell(&sandboxBuilder, sandboxTemplateNoCode, stregex.Task)

		for _, thing := range []interface{}{stregex.Example, stregex.NoTagIf, stregex.UseNTI, stregex.Prefix, stregex.Suffix} {
			writeCell(&sandboxBuilder, sandboxTemplateCode, thing)
		}

		writeCell(&sandboxBuilder, sandboxTemplateNoCode, stregex.Detected)

		if testpage != "" {
			if strings.HasPrefix(testpage, "User:Yapperbot/Scantag.sandbox/tests/") {
				log.Println("Processing test page", testpage)
				mapThisRegex := map[*regexp.Regexp]STRegex{expr: stregex}
				processArticle(w, testpage, mapThisRegex, true)
				processArticle(w, testpage, mapThisRegex, true)
				sandboxBuilder.WriteString("{{ph|")
				sandboxBuilder.WriteString(testpage)
				sandboxBuilder.WriteString("|Up-to-date}}")
			} else {
				log.Println("Invalid test page", testpage)
			}
		}
	}

	sandboxBuilder.WriteString(sandboxFooter)

	err = w.Edit(params.Values{
		"pageid":  config.SandboxPageID,
		"summary": "Updating sandbox from JSON",
		"bot":     "true",
		"text":    sandboxBuilder.String(),
	})
	if err == nil {
		log.Println("Sandbox updated")
	} else {
		if err == mwclient.ErrEditNoChange {
			log.Println("Detected sandbox changes to update, but looks like there actually weren't any")
		} else {
			switch err.(type) {
			case mwclient.APIError:
				switch err.(mwclient.APIError).Code {
				case "noedit", "writeapidenied", "blocked":
					ybtools.PanicErr("noedit/writeapidenied/blocked code returned, the bot may have been blocked. Dying")
				default:
					ybtools.PanicErr("API error updating sandbox: ", err)
				}
			default:
				ybtools.PanicErr("Non-API error updating sandbox: ", err)
			}
		}
	}
}

func writeCell(builder *strings.Builder, templateType string, thing interface{}) {
	builder.WriteString(fmt.Sprintf(templateType, thing))
	builder.WriteString(" || ")
}
