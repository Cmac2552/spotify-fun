package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	// Spotify API base URL
	spotifyAPIURL = "https://api.spotify.com/v1"
	// Redirect URI should match the one registered in your Spotify Developer Dashboard
	redirectURI = "http://localhost:8080/callback"
	// Scopes required for modifying playlists
	scope = "playlist-modify-public playlist-modify-private user-library-read user-library-modify playlist-modify-public"
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

type Playlist struct {
	ID string `json:"id"`
}

var server *http.Server

func runAfterServer() {
	// Temp logic, i will ping server in future
	time.Sleep(5 * time.Second) // Simulating some work after the server starts

	authURL := "https://accounts.spotify.com/authorize?" + "client_id=" + url.QueryEscape(clientID) +
		"&response_type=code" + "&redirect_uri=" + url.QueryEscape(redirectURI) +
		"&scope=" + url.QueryEscape(scope)

	//More temp i want to not have to open a browser
	// will have to deal with auth new tokens issue
	//command line args prob a good idea
	openBrowserWithURL(authURL)
}

func main() {

	go runAfterServer()

	server = &http.Server{
		Addr: ":8080",
	}

	http.HandleFunc("/callback", callback)

	fmt.Println("Starting server on :8080...")
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}

// TODO: clean up error handling
func callback(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	code := query.Get("code")
	if code == "" {
		http.Error(w, "Missing 'code' query parameter", http.StatusBadRequest)
		return
	}

	// Step 3: Exchange the authorization code for an access token
	token, err := getAccessToken(code)
	if err != nil {
		log.Fatal("Error exchanging authorization code for access token:", err)
	}

	playlistID, err := generatePlaylist(token.AccessToken)
	if err != nil {
		log.Fatal("Error creating Playlist:", err)
	}

	trackStrings, err := getTracksFromPlaylist(token.AccessToken)
	if err != nil {
		log.Fatal("Error getting Tracks from playlist:", err)
	}

	err = addTrackToPlaylist(token.AccessToken, playlistID, trackStrings)
	if err != nil {
		log.Fatal("Error adding track to playlist:", err)
	}

	fmt.Println("Tracks added to playlist successfully!")

	err = removeSongsFromLikedSongs(token.AccessToken, trackStrings)
	if err != nil {
		log.Fatal("Error removing tracks from playlist:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
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
	body, err := io.ReadAll(resp.Body)
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

	body, err := io.ReadAll(resp.Body)
	var t Tracks
	err = json.Unmarshal(body, &t)
	if err != nil {
		panic(err)
	}

	uris := make([]string, 0)
	for _, value := range t.Items {
		if value["track"]["uri"] != "" {
			uris = append(uris, value["track"]["uri"])
		}
	}

	return uris, nil
}

func addTrackToPlaylist(accessToken, playlistID string, trackURIs []string) error {

	url := fmt.Sprintf("%s/playlists/%s/tracks", spotifyAPIURL, playlistID)

	payload := map[string]interface{}{
		"uris": trackURIs,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

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

func removeSongsFromLikedSongs(accessToken string, trackURIs []string) error {

	trackIds := make([]string, len(trackURIs))

	for i, value := range trackURIs {
		lastColonIndex := strings.LastIndex(value, ":")
		trackIds[i] = value[lastColonIndex+1:]
	}

	payload := map[string]interface{}{
		"ids": trackIds,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodDelete,
		"https://api.spotify.com/v1/me/tracks", bytes.NewBuffer(payloadBytes))

	request.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, err := client.Do(request)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to remove tracks from playlist: %s", string(body))
	}

	return err
}

func generatePlaylist(accessToken string) (string, error) {
	var userID string
	fmt.Print("Enter user Id: ")
	fmt.Scan(&userID)

	var playlistName string
	fmt.Print("Enter playlist name: ")
	fmt.Scan(&playlistName)

	payload := map[string]interface{}{
		"name": playlistName,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("https://api.spotify.com/v1/users/%s/playlists", userID), bytes.NewBuffer(payloadBytes))

	request.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, err := client.Do(request)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var p Playlist
	err = json.Unmarshal(body, &p)

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to create playlist: %s", string(body))
	}
	fmt.Println(p.ID)

	return p.ID, err

}
