package fetch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var token string

// client bounds every Google Books call — the default http.Client has no
// timeout and would hang a request goroutine on a stalled upstream.
var client = &http.Client{Timeout: 15 * time.Second}

func getJSON(uri string, dest any) error {
	resp, err := client.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		// Don't decode error payloads into the result type — that silently
		// turns a 403/429 into an empty "no results" response.
		return fmt.Errorf("google books returned %s", resp.Status)
	}

	return json.Unmarshal(body, dest)
}

/*
Search for books. Returns a "list" (google's search object is a list inside an object)

Special search tags:
 1. `intitle:` Return results where the text following this word is found in the title.
 2. `inauthor:` …found in the author's name.
 3. `inpublisher:` …found in the publisher's name.
 4. `subject:` …found in the list of categories.
 5. `isbn:` Where the immediate next word matches the ISBN.
 6. `lccn:` …matches the Library of Congress Control number,
 7. `oclc:` …matches the Online Computer Library Center number.
*/
func GBooksSearch(search string) (*GBooksVolSearch, error) {
	uri := "https://www.googleapis.com/books/v1/volumes?q=" + url.QueryEscape(search)
	if token != "" {
		uri += "&key=" + url.QueryEscape(token)
	}

	ret := &GBooksVolSearch{}
	if err := getJSON(uri, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

// Google Books Volume ID
func GBooksVolume(volume string) (*GBooksVolDetails, error) {
	uri := "https://www.googleapis.com/books/v1/volumes/" + url.PathEscape(volume) + "?projection=full"
	if token != "" {
		uri += "&key=" + url.QueryEscape(token)
	}

	ret := &GBooksVolDetails{}
	if err := getJSON(uri, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func SetAPIToken(newToken string) {
	token = newToken
}
