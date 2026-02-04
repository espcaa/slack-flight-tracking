package slackutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
)

type SlackFileObject struct {
	ID                 string   `json:"id"`
	Created            int64    `json:"created"`
	Timestamp          int64    `json:"timestamp"`
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Mimetype           string   `json:"mimetype"`
	Filetype           string   `json:"filetype"`
	PrettyType         string   `json:"pretty_type"`
	User               string   `json:"user"`
	Editable           bool     `json:"editable"`
	Size               int      `json:"size"`
	Mode               string   `json:"mode"`
	IsExternal         bool     `json:"is_external"`
	ExternalType       string   `json:"external_type,omitempty"`
	IsPublic           bool     `json:"is_public"`
	PublicUrlShared    bool     `json:"public_url_shared"`
	DisplayAsBot       bool     `json:"display_as_bot"`
	Username           string   `json:"username,omitempty"`
	UrlPrivate         string   `json:"url_private"`
	UrlPrivateDownload string   `json:"url_private_download"`
	Permalink          string   `json:"permalink"`
	PermalinkPublic    string   `json:"permalink_public,omitempty"`
	Channels           []string `json:"channels,omitempty"`
	Groups             []string `json:"groups,omitempty"`
	Ims                []string `json:"ims,omitempty"`
	CommentsCount      int      `json:"comments_count"`
	IsStarred          bool     `json:"is_starred"`
	HasRichPreview     bool     `json:"has_rich_preview"`
}

func completeUpload(token string, fileID string, fileName string, channelID string) (*SlackFileObject, error) {
	apiURL := "https://slack.com/api/files.completeUploadExternal"

	payload := map[string]any{
		"files": []map[string]string{
			{
				"id":    fileID,
				"title": fileName,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Ok    bool              `json:"ok"`
		Error string            `json:"error"`
		Files []SlackFileObject `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if !response.Ok {
		return nil, fmt.Errorf("slack api error: %s", response.Error)
	}

	if len(response.Files) > 0 {
		return &response.Files[0], nil
	}

	return nil, fmt.Errorf("no files returned in response")
}

func UploadFileAndGetURL(parameters slack.UploadFileV2Parameters, client *slack.Client, token string) (slack.SlackFileObject, error) {
	ctx := context.Background()

	getUploadURLParameters := slack.GetUploadURLExternalParameters{
		FileName: parameters.Filename,
		FileSize: parameters.FileSize,
	}

	uploadURLResponse, err := client.GetUploadURLExternalContext(ctx, getUploadURLParameters)
	if err != nil {
		return slack.SlackFileObject{}, fmt.Errorf("failed to get upload URL: %w", err)
	}

	uploadToUrlParameters := slack.UploadToURLParameters{
		UploadURL: uploadURLResponse.UploadURL,
		Reader:    parameters.Reader,
		File:      parameters.File,
		Content:   parameters.Content,
	}

	err = client.UploadToURL(ctx, uploadToUrlParameters)
	if err != nil {
		return slack.SlackFileObject{}, fmt.Errorf("failed to upload content to URL: %w", err)
	}

	fileo, err := completeUpload(token, uploadURLResponse.FileID, parameters.Filename, parameters.Channel)
	if err != nil {
		return slack.SlackFileObject{}, fmt.Errorf("failed to complete upload: %w", err)
	}

	return slack.SlackFileObject{
		ID: fileo.ID,
	}, nil
}
