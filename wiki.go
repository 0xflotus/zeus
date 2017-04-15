/*
 *  ZEUS - An Electrifying Build System
 *  Copyright (c) 2017 Philipp Mieden <dreadl0ck [at] protonmail [dot] ch>
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/russross/blackfriday"
)

// serve wiki index page
var wikiIndexHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html")

	index, err := ioutil.ReadFile("wiki/INDEX.md")
	if err != nil {
		Log.WithError(err).Error("failed to read wiki index markdown")
		return
	}

	tpl, err := assetBox.String("wiki_index.html")
	if err != nil {
		Log.WithError(err).Fatal("failed to read wiki index HTML")
	}

	t, err := template.New("wiki").Parse(tpl)
	if err != nil {
		Log.WithError(err).Fatal("failed to create index template")
	}

	err = t.Execute(w, template.HTML(blackfriday.MarkdownCommon(index)))
	if err != nil {
		Log.WithError(err).Fatal("failed to exec template")
	}
})

// serve wiki documents
var wikiDocsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	fileName := strings.TrimPrefix(r.RequestURI, "/")

	Log.Debug("wikiDocsHandler: ", fileName)

	if strings.HasSuffix(fileName, ".html") {
		tpl, err := ioutil.ReadFile(fileName)
		if err != nil {
			Log.WithError(err).Fatal("failed to read wiki HTML file: ", fileName)
		}
		t, err := template.New("wiki").Parse(string(tpl))
		if err != nil {
			Log.WithError(err).Fatal("failed to create index template")
		}

		err = t.Execute(w, nil)
		if err != nil {
			Log.WithError(err).Fatal("failed to exec template")
		}

		return
	}

	// get file
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		Log.WithError(err).Error("unknown file")
		w.WriteHeader(404)
		w.Write([]byte("Not found"))
		return
	}

	// handle images
	if strings.HasSuffix(fileName, ".png") {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(200)
		w.Write(b)
		return
	} else if strings.HasSuffix(fileName, ".jpg") {
		w.Header().Set("Content-Type", "image/jpg")
		w.WriteHeader(200)
		w.Write(b)
		return
	} else if strings.HasSuffix(fileName, ".pdf") {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(200)
		w.Write(b)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	// get template
	tpl, err := assetBox.String("wiki_index.html")
	if err != nil {
		Log.WithError(err).Fatal("failed to read wiki index HTML")
	}

	// parse template
	t, err := template.New("wiki").Parse(tpl)
	if err != nil {
		Log.WithError(err).Fatal("failed to create index template")
	}

	// execute template
	err = t.Execute(w, template.HTML(blackfriday.MarkdownCommon(b)))
	if err != nil {
		Log.WithError(err).Fatal("failed to exec template")
	}
})
