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

// Config stores relevant configuration information, and is retrieved from a
// YAML file by ybtools.
type Config struct {
	RegexesPageID  string
	PathToArticles string
}
