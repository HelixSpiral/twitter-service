package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

func uploadImage(httpClient *http.Client, image []byte) (twitterMediaResponse, error) {
	var mediaResp twitterMediaResponse

	imageReader := bytes.NewReader(image)
	imageBuf := &bytes.Buffer{}
	form := multipart.NewWriter(imageBuf)

	fw, err := form.CreateFormFile("media", "twitterPicture.jpg")
	if err != nil {
		log.Println(err)
	}

	_, err = io.Copy(fw, imageReader)
	if err != nil {
		log.Println(err)
	}
	form.Close()

	resp, err := httpClient.Post("https://upload.twitter.com/1.1/media/upload.json?media_category=tweet_image", form.FormDataContentType(), bytes.NewReader(imageBuf.Bytes()))
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	err = json.Unmarshal(body, &mediaResp)
	if err != nil {
		log.Println(err)
	}

	return mediaResp, nil
}

func sendMessage(httpClient *http.Client, message, mediaID string) (*http.Response, error) {
	// We have to define these because of some odd scoping in the if's
	var resp *http.Response
	var err error

	// If we have a media ID, use it, otherwise don't.
	if mediaID != "" {
		resp, err = httpClient.Post("https://api.twitter.com/2/tweets", "application/json",
			bytes.NewBuffer([]byte(fmt.Sprintf(`{"text": "%s", "media": {"media_ids": ["%s"]}}`, message, mediaID))))
		if err != nil {
			return nil, fmt.Errorf("error in http POST: %w", err)
		}
	} else {
		resp, err = httpClient.Post("https://api.twitter.com/2/tweets", "application/json",
			bytes.NewBuffer([]byte(fmt.Sprintf(`{"text": "%s"}`, message))))
		if err != nil {
			return nil, fmt.Errorf("erorr in http POST: %w", err)
		}
	}

	return resp, nil
}
