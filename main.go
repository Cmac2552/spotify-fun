package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Access struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   string `json:"expries_in"`
}
type Tracks struct {
	Items []map[string]map[string]string `json:"items"`
	Total int                            `json:"total`
}

func main() {
	var client_id = ""
	var client_secret = ""
	resource := "https://accounts.spotify.com/api/token"
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodPost, resource, strings.NewReader(data.Encode())) // URL-encoded payload
	r.Header.Set("Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(client_id+string(':')+client_secret)))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := client.Do(r)
	fmt.Println(resp.Status)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	var a Access

	err = json.Unmarshal(bodyBytes, &a)
	if err != nil {
		panic(err)
	}

	fmt.Println(a.AccessToken)

	r, _ = http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/playlists/34M6oSFlsM9KpR8uq12opW/tracks?fields=items(track.uri),total", nil)
	r.Header.Set("Authorization", "Bearer "+a.AccessToken)
	resp, _ = client.Do(r)

	bodyBytes, _ = io.ReadAll(resp.Body)
	var t Tracks
	err = json.Unmarshal(bodyBytes, &t)
	if err != nil {
		panic(err)
	}

	uris := make([]string, t.Total)
	for index, value := range t.Items {
		uris[index] = value["track"]["uri"]
	}

	//Next
	//https://developer.spotify.com/documentation/web-api/reference/add-tracks-to-playlist
	data = url.Values{}
	data.Set("uris", "["+strings.Join(uris, ",")+"]")
	r, _ = http.NewRequest(http.MethodPost, "https://api.spotify.com/v1/playlists/7y6xmpQkWU420xWo0Sfbos/tracks", strings.NewReader(data.Encode()))
	r.Header.Set("Authorization", "Bearer "+a.AccessToken)
	r.Header.Set("Content-Type", "Content-Type: application/json")
	resp, _ = client.Do(r)
	fmt.Println(resp.Status)
	bodyBytes, _ = io.ReadAll(resp.Body)
	fmt.Println(string(bodyBytes))
}
