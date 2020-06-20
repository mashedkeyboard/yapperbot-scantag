package main

import (
	"fmt"
	"regexp"

	"github.com/antonholmquist/jason"
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

// STRegex objects represent individual regexes being used by Scantag.
type STRegex struct {
	Task     string
	Example  string
	NoTagIf  *regexp.Regexp
	UseNTI   bool
	Prefix   string
	Suffix   string
	Detected string
}

func processRegex(regex string, content *jason.Value) (expr *regexp.Regexp, strgx STRegex, testpage string, err error) {
	value, err := content.Object()
	if err != nil {
		err = fmt.Errorf("Scantag.json key `%s` is invalid! Error was %s", regex, err)
		return
	}

	expression, err := regexp.Compile("(?i)" + regex)
	if err != nil {
		err = fmt.Errorf("Regex `%s` is invalid! Error was %s", regex, err)
		return
	}

	detected, err := value.GetString("detected")
	if err != nil {
		err = fmt.Errorf("Scantag.json is invalid, no detected string retrieved for `%s` - error was %s", regex, err)
		return
	}

	var ntiexp *regexp.Regexp
	var useNTI bool = true

	nti, err := value.GetString("noTagIf")
	if err == nil {
		ntiexp, err = regexp.Compile("(?i)" + nti)
		if err != nil {
			err = fmt.Errorf("noTagIf regex for main regex %s is invalid! Error was `%s`", regex, err)
			return
		}
	} else {
		nonti, fetchErr := value.GetBoolean("noTagIf")
		if fetchErr != nil || nonti != false {
			err = fmt.Errorf("NoTagIf invalid (neither regex nor false) retrieved for `%s` - error was %s", regex, err)
			return
		}
		useNTI = false
	}

	prefix, _ := value.GetString("prefix")
	suffix, _ := value.GetString("suffix")

	task, _ := value.GetString("task")
	example, _ := value.GetString("example")
	testpage, _ = value.GetString("testpage")

	return expression, STRegex{
		Task:     task,
		Example:  example,
		Detected: detected,
		Prefix:   prefix,
		Suffix:   suffix,
		NoTagIf:  ntiexp,
		UseNTI:   useNTI,
	}, testpage, nil
}

/* The JSON file containing regexes is expected to be of this format:

{
    "Regex to match (remember, this has to be fully JSON escaped, not just a valid regex, otherwise it ''will not work'')": {
        "task": "Brief description of task",
		"example": "Example of something that would be tagged by the task",
		"noTagIf": "A regex which, if it matches against the page, will cause the page to be ignored. Usually used to avoid tagging pages that already contain maintenance tags. Use boolean false to always tag; be careful with this! Like the key regex, must be JSON escaped as well as valid regex.",
		"prefix": "Something to prefix the articles that the task finds with, with $ signs escaped with an additional sign (i.e. $ in output should read $$); each regex capture group is available as "${n}", replacing n with the one-indexed number of the capture group",
		"suffix": "Same as prefix, but appends to the article rather than prepending",
		"detected": "Describes what was detected and why it's doing something; should come after the word 'detected', and potentially have other detected aspects after it separated with semicolons",
		"testpage": "The page name of a page on which the matching will be tested. When the sandbox is updated, Yapperbot will run Scantag's sandbox rules twice (so that the NoTagIf rule can be tested) over this page. Must be prefixed 'User:Yapperbot/Scantag.sandbox/tests/'."
    }
}

*/
