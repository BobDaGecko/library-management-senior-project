package common

import (
	"html/template"
	"strings"
)

var _pagePrev *template.Template
var _pageInput *template.Template
var _pageNext *template.Template

func pagePrev(id string) string {
	if _pagePrev == nil {
		var err error
		_pagePrev, err = template.New("pagePrev").Parse(
			`on click send Set(page:#{{ .Pager }}.value-1) to #{{ .Pager }}`,
		)

		if err != nil {
			panic(err)
		}
	}

	ret := strings.Builder{}
	_pagePrev.Execute(&ret, map[string]any{
		"Pager": id,
	})
	return ret.String()
}

func pageInput(id string) string {
	if _pageInput == nil {
		var err error
		_pageInput, err = template.New("pageInput").Parse(`
			on Set(page)
				if page as Int <= my.min as Int
					set page to my.min as Int
					add .disabled to #{{ .Pager }}-prev
				else
					remove .disabled from #{{ .Pager }}-prev
				end

				if page as Int >= my.max as Int
					set page to my.max as Int
					add .disabled to #{{ .Pager }}-next
				else
					remove .disabled from #{{ .Pager }}-next
				end

				if page as Int != my.value as Int
					put page into my.value
					trigger change on me
					trigger submit on #{{ .Pager }}-form
				end
			end

			on change
				send Set(page:my.value) to me
			end

			on wheel
				if event.wheelDeltaY < 0
					send Set(page:my.value-1) to me
				else
					send Set(page:my.value--1) to me
				end
			end
		`)

		if err != nil {
			panic(err)
		}
	}

	ret := strings.Builder{}
	_pageInput.Execute(&ret, map[string]any{
		"Pager": id,
	})
	return ret.String()
}

func pageNext(id string) string {
	if _pageNext == nil {
		var err error
		_pageNext, err = template.New("pageNext").Parse(
			`on click send Set(page:#{{ .Pager }}.value--1) to #{{ .Pager }}`,
		)

		if err != nil {
			panic(err)
		}
	}

	ret := strings.Builder{}
	_pageNext.Execute(&ret, map[string]any{
		"Pager": id,
	})
	return ret.String()
}
