package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

func getMentionSearchQuery(username, org string) string {
	return buildSearchQuery("is:open mentions:%v archived:false %v", username, org)
}

func getReviewSearchQuery(username, org string) string {
	return buildSearchQuery("is:pr is:open review-requested:%v archived:false %v", username, org)
}

func getYourPrsSearchQuery(username, org string) string {
	return buildSearchQuery("is:pr is:open author:%v archived:false %v", username, org)
}

func getYourAssigneeSearchQuery(username, org string) string {
	return buildSearchQuery("is:open assignee:%v archived:false %v", username, org)
}

func buildSearchQuery(query, username, org string) string {
	orgField := ""
	if len(org) != 0 {
		orgField = fmt.Sprintf("org:%v", org)
	}

	return fmt.Sprintf(query, username, orgField)
}

func pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])

	if unpadding > length {
		return nil, errors.New("unpad error. This could happen when incorrect encryption key is used")
	}

	return src[:(length - unpadding)], nil
}

func encrypt(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	msg := pad([]byte(text))
	ciphertext := make([]byte, aes.BlockSize+len(msg))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(msg))
	finalMsg := base64.URLEncoding.EncodeToString(ciphertext)
	return finalMsg, nil
}

func decrypt(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	decodedMsg, err := base64.URLEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}

	if (len(decodedMsg) % aes.BlockSize) != 0 {
		return "", errors.New("blocksize must be multipe of decoded message length")
	}

	iv := decodedMsg[:aes.BlockSize]
	msg := decodedMsg[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(msg, msg)

	unpadMsg, err := unpad(msg)
	if err != nil {
		return "", err
	}

	return string(unpadMsg), nil
}

func parseOwnerAndRepo(full, baseURL string) (string, string, string) {
	if baseURL == "" {
		baseURL = "https://github.com/"
	}
	full = strings.TrimSuffix(strings.TrimSpace(strings.Replace(full, baseURL, "", 1)), "/")
	splitStr := strings.Split(full, "/")

	if len(splitStr) == 1 {
		owner := splitStr[0]
		return fmt.Sprintf("%s", owner), owner, ""
	} else if len(splitStr) != 2 {
		return "", "", ""
	}
	owner := splitStr[0]
	repo := splitStr[1]

	return fmt.Sprintf("%s/%s", owner, repo), owner, repo
}

func parseGitHubUsernamesFromText(text string) []string {
	usernameMap := map[string]bool{}
	usernames := []string{}

	for _, word := range strings.FieldsFunc(text, func(c rune) bool {
		return !(c == '-' || c == '@' || unicode.IsLetter(c) || unicode.IsNumber(c))
	}) {
		if len(word) < 2 || word[0] != '@' {
			continue
		}

		if word[1] == '-' || word[len(word)-1] == '-' {
			continue
		}

		if strings.Contains(word, "--") {
			continue
		}

		name := word[1:]
		if !usernameMap[name] {
			usernames = append(usernames, name)
			usernameMap[name] = true
		}
	}

	return usernames
}

func fixGithubNotificationSubjectURL(url string) string {
	url = strings.Replace(url, "api.", "", 1)
	url = strings.Replace(url, "repos/", "", 1)
	url = strings.Replace(url, "/pulls/", "/pull/", 1)
	url = strings.Replace(url, "/api/v3", "", 1)
	return url
}
