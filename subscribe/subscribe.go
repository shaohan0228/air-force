package subscribe

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Config struct {
	Protocol string `json:"net"`
	Addr     string `json:"add"`
	Port     string `json:"port"`
	Name     string `json:"ps"`
}

var allowedSchemes = []string{"vmess", "ss", "socks", "ssr"}

func LoadConfigs(ctx context.Context, url string) (configs []*Config, err error) {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	bBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	dec, err := base64.StdEncoding.DecodeString(string(bBody))
	if err != nil {
		return
	}

	links := strings.Split(string(dec), "\n")

	fmt.Println(links)
	configs = make([]*Config, 0, len(links))

	for _, link := range links {
		if len(link) < 1 {
			continue
		}
		segments := strings.Split(link, "://")
		if len(segments) < 2 || !Contains(allowedSchemes, segments[0]) {
			log.Printf("invalid link: %s", link)
			continue
		}

		dec, err = base64.StdEncoding.DecodeString(segments[1])
		if err != nil {
			log.Printf("decode link '%s' failed, %s", link, err)
			continue
		}

		cfg := &Config{}
		json.Unmarshal(dec, cfg)
		configs = append(configs, cfg)
	}

	return
}

func Contains(strArr []string, str string) bool {
	for _, s := range strArr {
		if s == str {
			return true
		}
	}
	return false
}
