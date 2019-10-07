package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
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

func getIssuesSearchQuery(username, org, searchTerm string) string {
	query := "is:open is:issue assignee:%v archived:false %v %v"
	orgField := ""
	if len(org) != 0 {
		orgField = fmt.Sprintf("org:%v", org)
	}

	return fmt.Sprintf(query, username, orgField, searchTerm)
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
		return fmt.Sprintf("%s/", owner), owner, ""
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
	url = strings.Replace(url, "/commits/", "/commit/", 1)
	url = strings.Replace(url, "/api/v3", "", 1)
	return url
}

func fullNameFromOwnerAndRepo(owner, repo string) string {
	return fmt.Sprintf("%s/%s", owner, repo)
}

// filter lines in a string from start to end
func filterLines(s string, start, end int) string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	var buf strings.Builder
	i := 1
	for scanner.Scan() {
		if i < start {
			i++
			continue
		}
		if i > end {
			break
		}
		buf.Write(scanner.Bytes())
		buf.WriteByte(byte('\n'))
		i++
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}

	return buf.String()
}

// getLineNumbers return the start and end lines from an anchor tag
// of a github permalink.
func getLineNumbers(s string) (start, end int) {
	// split till -
	parts := strings.Split(s, "-")

	if len(parts) > 2 {
		return -1, -1
	}

	switch len(parts) {
	case 1: // just a single line
		l := getLine(parts[0])
		if l == -1 {
			return -1, -1
		}
		if l < 3 {
			return 0, l + 3
		}
		return l - 3, l + 3
	case 2: // a line rage
		start := getLine(parts[0])
		end := getLine(parts[1])
		return start, end
	}
	return -1, -1
}

// getLine returns the line number in int from a string
// of form L<num>.
func getLine(s string) int {
	// check starting L and minimum length
	if !strings.HasPrefix(s, "L") || len(s) < 2 {
		return -1
	}

	line, err := strconv.Atoi(s[1:])
	if err != nil {
		return -1
	}
	return line
}

// isInsideLink reports whether the given index in a string is preceeded
// by zero or more space, then (, then ].
//
// It is a poor man's version of checking markdown hyperlinks without
// using a full-blown markdown parser. The idea is to quickly confirm
// whether a permalink is inside a markdown link or not. Something like
// "text ]( permalink" is rare enough. Even then, it is okay if
// there are false positives, but there cannot be any false negatives.
//
// Note: it is fine to go one byte at a time instead of one rune because
// we are anyways looking for ASCII chars.
func isInsideLink(msg string, index int) bool {
	stage := 0 // 0 is looking for space or ( and 1 for ]

	for i := index; i > 0; i-- {
		char := msg[i-1]
		switch stage {
		case 0:
			if char == ' ' {
				continue
			}
			if char == '(' {
				stage++
				continue
			}
			return false
		case 1:
			if char == ']' {
				return true
			}
			return false
		}
	}
	return false
}
