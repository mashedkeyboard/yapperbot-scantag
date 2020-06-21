package main

import (
	"regexp"
	"strings"
)

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

// The below regexes are inspired by
// https://github.com/azatoth/twinkle/blob/6575649e13aad887058b404206b531cc3be6b144/modules/friendlytag.js#L1634 and
// https://github.com/reedy/AutoWikiBrowser/blob/9a60e46d148d84ae2bcae4d66882485da9fdd24c/WikiFunctions/WikiRegexes.cs#L1177.
// They have been reconfigured to add additional templates, and to work with Golang's RE2 system of regex matching.
const regexOpen = `(?i)^\s*(?:((?:\s*`
const regexAFD = `(?:<!--.*AfD.*\n\{\{(?:(?:Article for deletion|Afd)\/dated|AfDM).*\}\}\n<!--.*(?:\n<!--.*)?AfD.*(?:\s*\n))|`
const regexTemplateOpen = `\{\{\s*`
const regexSpeedy = `(?:db|delete|db-.*?|speedy deletion-.*?|(?:proposed deletion|prod blp)\/dated(?:\s*\|(?:concern|user|timestamp|help).*)+|`
const regexOtherTemplates = `about|about-distinguish|ambiguous link|correct title|dablink|disambig-acronym|distinguish|distinguish-otheruses|for|further|hatnote|other\s?(?:hurricanes|people|persons|places|ships|uses?(?:\s?of)?)|outline|redirect(?:-(?:acronym|distinguish|several))?|see\s?(?:also|wiktionary)|selfref|short(?:desc| description)|the|this|salt|proposed deletion endorsed) ?\d*\s*`
const regexTemplateParams = `(?:\|(?:\{\{[^{}]*\}\}|[^{}])*)?`
const regexEndTemplateMatch = `\}\}\n?)+`
const regexClose = `(?:\s*\n)?)\s*)?`

var regexBuilder strings.Builder
var startRegex *regexp.Regexp

func init() {
	// Setup the regex string
	regexBuilder.WriteString(regexOpen)
	regexBuilder.WriteString(regexAFD)
	regexBuilder.WriteString(regexTemplateOpen)
	regexBuilder.WriteString(regexSpeedy)
	regexBuilder.WriteString(regexOtherTemplates)
	regexBuilder.WriteString(regexTemplateParams)
	regexBuilder.WriteString(regexEndTemplateMatch)
	regexBuilder.WriteString(regexClose)

	// Compile the regex itself
	startRegex = regexp.MustCompile(regexBuilder.String())
}

func insertMaintenanceTemplate(text, prepend string) string {
	// Replace whatever our start regex matches with itself, plus then the prepend value
	// We replace the dollar signs in the prepend value because those are consumed by ReplaceAllString otherwise
	return startRegex.ReplaceAllString(text, "$1"+strings.ReplaceAll(prepend, "$", "$$"))
}
