package main

type MqttMessage struct {
	TwitterConsumerKey    string
	TwitterConsumerSecret string
	TwitterAccessToken    string
	TwitterAccessSecret   string

	Message string
	Image   []byte // Deprecated in favor of Images
	Images  [][]byte
}

type twitterMediaResponse struct {
	MediaID       int              `json:"media_id"`
	MediaIDString string           `json:"media_id_string"`
	MediaKey      string           `json:"media_key"`
	Size          int              `json:"size"`
	ExpiresAfter  int              `json:"expires_after_secs"`
	Image         twitterImageInfo `json:"image"`
}

type twitterImageInfo struct {
	Type   string `json:"image_type"`
	Width  int    `json:"w"`
	Height int    `json:"h"`
}

type TwitterMsg struct {
	Text  string       `json:"text"`
	Media TwitterMedia `json:"media,omitempty"`
}

type TwitterMedia struct {
	MediaIDs []string `json:"media_ids,omitempty"`
}
