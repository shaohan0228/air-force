package subscribe

import (
	"context"
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

	dec, err := b64Decode(string(bBody))
	if err != nil {
		return
	}

	links := strings.Split(dec, "\n")

	configs = make([]*Config, 0, len(links))

	for _, link := range links {
		if len(link) < 1 {
			continue
		}
		segments := strings.Split(link, "://")
		if len(segments) < 2 {
			log.Printf("invalid link: %s", link)
			continue
		}
		config, err := SchemeParser(segments[0], segments[1])
		if err != nil {
			log.Printf("parse link '%s' error, err: %s", link, err)
			continue
		}

		configs = append(configs, config)
	}

	return
}

