package main

import (
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
type Tracks2 struct {
	Items []map[string]map[string]string `json:"items"`
	Total int                            `json:"total`
}

func main2() {
	var client_id = ""
	var client_secret = ""
	resource := "https://accounts.spotify.com/api/token"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", "AQD4X_lJMHaxauXazyNXXVRMzdJ8vfjJVyPfPpKftpiaafPh33-LQIuKq5TVEeHH-0yrdeAiSnfAbWgHC2qsvKhoEoZIoYGvep46Aw9RI27C2r3xp9WHoB7hYb3ZpDQUaxeDDo1sfsN_jDKZkD_GRfhpjtWHuk8xYVys1EUC7lI2CsBncEuJe-mWCf7Cgl4axt81DE0o8TcBy7S7GYZ1rQchGydSQ2NZ8gisNRHLdnFNhVA")
	data.Set("redirect_uri", "http://localhost:8080/callback")
	data.Set("client_id", client_id)
	data.Set("client_secret", client_secret)
	client := &http.Client{}
	r, err := http.PostForm(resource, data) // URL-encoded payload
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(r.Status)

	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
	}
	var a Access

	err = json.Unmarshal(bodyBytes, &a)
	if err != nil {
		panic(err)
	}

	fmt.Println(a.AccessToken)

	r2, _ := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/playlists/34M6oSFlsM9KpR8uq12opW/tracks?fields=items(track.uri),total", nil)
	r.Header.Set("Authorization", "Bearer "+a.AccessToken)
	resp, _ := client.Do(r2)

	bodyBytes, _ = io.ReadAll(resp.Body)
	var t Tracks
	err = json.Unmarshal(bodyBytes, &t)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Status)

	uris := make([]string, 12)
	for index, value := range t.Items {
		uris[index] = value["track"]["uri"]
	}

	// //Next
	// //https://developer.spotify.com/documentation/web-api/reference/add-tracks-to-playlist
	data = url.Values{}
	data.Set("uris", "["+strings.Join(uris, ",")+"]")
	r2, _ = http.NewRequest(http.MethodPost, "https://api.spotify.com/v1/playlists/7y6xmpQkWU420xWo0Sfbos/tracks", strings.NewReader(data.Encode()))
	r.Header.Set("Authorization", "Bearer "+a.AccessToken)
	r.Header.Set("Content-Type", "Content-Type: application/json")
	resp, _ = client.Do(r2)
	fmt.Println(resp.Status)
	bodyBytes, _ = io.ReadAll(resp.Body)
	fmt.Println(string(bodyBytes))
}
