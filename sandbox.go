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
	"strings"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools/v2"
)

// Leave the newline at the end of the header; it's important.
const sandboxHeader string = `{|class="wikitable"
|-
! Task !! Example !! Don't tag if matches !! Use noTagIf? !! Prefix the article with !! Suffix the article with !! Detected...
`

// Leave the newline at the start of this; it's also important.
const sandboxFooter string = `
|}`

const sandboxTemplateOpening string = `|-
! colspan="7" | <code><nowiki>%s</nowiki></code>
|-
| `
const sandboxTemplate string = `<code><nowiki>%v</nowiki></code>`
const sandboxError string = `colspan="7" {{no O|<code><nowiki>%s</nowiki></code>}}`

func createSandbox(w *mwclient.Client) {
	sandboxJSON := ybtools.LoadJSONFromPageID(config.SandboxJSONPageID)
	var sandboxBuilder strings.Builder

	sandboxBuilder.WriteString(sandboxHeader)

	for regex, content := range sandboxJSON.Map() {
		sandboxBuilder.WriteString(fmt.Sprintf(sandboxTemplateOpening, regex))
		_, stregex, err := processRegex(regex, content)
		if err != nil {
			sandboxBuilder.WriteString(fmt.Sprintf(sandboxError, err))
			continue
		}

		var bits []string
		for _, thing := range []interface{}{stregex.Task, stregex.Example, stregex.NoTagIf, stregex.UseNTI, stregex.Prefix, stregex.Suffix, stregex.Detected} {
			bits = append(bits, fmt.Sprintf(sandboxTemplate, thing))
		}
		sandboxBuilder.WriteString(strings.Join(bits, " || "))
	}

	sandboxBuilder.WriteString(sandboxFooter)

	err := w.Edit(params.Values{
		"pageid":  config.SandboxPageID,
		"summary": "Updating sandbox from JSON",
		"bot":     "true",
		"text":    sandboxBuilder.String(),
	})
	if err == nil {
		log.Println("All done, sandbox updated")
	} else {
		if err == mwclient.ErrEditNoChange {
			log.Println("No sandbox changes to update")
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
