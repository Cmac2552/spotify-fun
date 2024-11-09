package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
)

const (
	// Spotify API base URL
	spotifyAPIURL = "https://api.spotify.com/v1"
	// Redirect URI should match the one registered in your Spotify Developer Dashboard
	redirectURI = "http://localhost:8080/callback"
	// Scopes required for modifying playlists
	scope = "playlist-modify-public playlist-modify-private user-library-read"
)

var (
	clientID     = ""
	clientSecret = ""
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type Tracks struct {
	Items []map[string]map[string]string `json:"items"`
}

func main() {
	// Step 1: Generate the authentication URL and have the user log in.
	authURL := "https://accounts.spotify.com/authorize?" + "client_id=" + url.QueryEscape(clientID) +
		"&response_type=code" + "&redirect_uri=" + url.QueryEscape(redirectURI) +
		"&scope=" + url.QueryEscape(scope)

	openBrowserWithURL(authURL)

	// Step 2: Handle the authorization code
	var authCode string
	fmt.Print("Enter the code: ")
	fmt.Scan(&authCode)

	// Step 3: Exchange the authorization code for an access token
	token, err := getAccessToken(authCode)
	if err != nil {
		log.Fatal("Error exchanging authorization code for access token:", err)
	}

	// Step 4: Use the access token to interact with Spotify API
	// Example: Adding a track to a playlist
	var playlistID string
	fmt.Print("Enter playlist ID: ")
	fmt.Scan(&playlistID)
	// trackURI := "spotify:track:4iV5W9uYEdYUVa79Axb7Rh" // The URI of the track you want to add to the playlist
	trackStrings, err := getTracksFromPlaylist(token.AccessToken)
	if err != nil {
		log.Fatal("Error getting Tracks from playlist:", err)
	}
	// fmt.Println(trackStrings[0])
	err = addTrackToPlaylist(token.AccessToken, playlistID, trackStrings)
	if err != nil {
		log.Fatal("Error adding track to playlist:", err)
	}

	fmt.Println("Track added to playlist successfully!")
}

// getAccessToken exchanges an authorization code for an access token.
func getAccessToken(authCode string) (*TokenResponse, error) {
	// Prepare the data to send to the Spotify token endpoint
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", authCode)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	// Make the POST request
	resp, err := http.PostForm("https://accounts.spotify.com/api/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get access token: %s", string(body))
	}

	// Parse the response JSON
	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func getTracksFromPlaylist(accessToken string) ([]string, error) {
	r2, _ := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/me/tracks?limit=50&fields=items(track.uri),total", nil)
	r2.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, _ := client.Do(r2)

	bodyBytes, err := io.ReadAll(resp.Body)
	var t Tracks
	err = json.Unmarshal(bodyBytes, &t)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Status)

	uris := make([]string, 0)
	for _, value := range t.Items {
		if value["track"]["uri"] != "" {
			uris = append(uris, value["track"]["uri"])
		}
	}

	return uris, nil
}

// addTrackToPlaylist adds a track to a Spotify playlist.
func addTrackToPlaylist(accessToken, playlistID string, trackURIs []string) error {
	// Create the URL for adding tracks to a playlist
	url := fmt.Sprintf("%s/playlists/%s/tracks", spotifyAPIURL, playlistID)

	// Prepare the request payload
	payload := map[string]interface{}{
		"uris": trackURIs,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create a new POST request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}

	// Set the Authorization header with the access token
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Check for a successful response
	fmt.Println(resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to add track to playlist: %s", string(body))
	}

	return nil
}

func openBrowserWithURL(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal("Error opening browser:", err)
	}
}
