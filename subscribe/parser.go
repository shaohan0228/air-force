package subscribe

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
)

type IParser interface {
	Parse(data []byte, config *Config) (*Config, error)
}

var parserMap = map[string]IParser{}

func init() {
	parserMap = map[string]IParser{
		"vmess": VParser{},
		"ssr":   NewSSRParser(),
		"ss":    NewSSParser(),
	}
}

func SchemeParser(scheme string, content string) (*Config, error) {
	parser, ok := parserMap[scheme]
	if !ok {
		return nil, errors.New(fmt.Sprintf("unsupported scheme: %s", scheme))
	}

	cfg := &Config{}

	if scheme == "ss" {
		return parser.Parse([]byte(content), cfg)
	}

	dec, err := b64Decode(content)
	if err != nil {
		return nil, err
	}

	return parser.Parse([]byte(dec), cfg)
}

type VParser struct {
}

func (p VParser) Parse(data []byte, config *Config) (*Config, error) {
	err := json.Unmarshal(data, config)
	return config, err
}

func NewSSRParser() SSRParser{
	p := SSRParser{}
	p.regexp = regexp.MustCompile(`remarks=(\S+)&`)
	return p
}

type SSRParser struct {
	regexp *regexp.Regexp
}

func (p SSRParser) Parse(data []byte, config *Config) (*Config, error) {
	segments := strings.Split(string(data), ":")
	if len(segments) < 6 {
		return nil, errors.New(fmt.Sprintf("unknown content: %s", string(data)))
	}

	config.Addr = segments[0]
	config.Port = segments[1]

	match := p.regexp.FindStringSubmatch(segments[5])
	if len(match) < 2 {
		config.Name = "unknown"
		return config, nil
	}

	name, err := b64Decode(match[1])
	if err != nil {
		config.Name = "unknown"
		log.Printf("decode name '%s' error, err: %s\n", match[1], err)
		return config, nil
	}
	config.Name = name
	return config, nil
}

func NewSSParser() SSParser {
	p := SSParser{}
	p.regexp = regexp.MustCompile(`@(\S+):(\d+)\/\?(\S+)#(\S+)`)
	return p
}

type SSParser struct {
	regexp *regexp.Regexp
}

func (p SSParser) Parse(data []byte, config *Config) (*Config, error) {
	match := p.regexp.FindStringSubmatch(string(data))
	if len(match) < 5 {
		return nil, errors.New(fmt.Sprintf("can not find sub match in '%s'", string(data)))
	}
	// group 1
	config.Addr = match[1]
	// group 2
	config.Port = match[2]

	// group 4
	name, err := url.QueryUnescape(match[4])
	if err != nil {
		return nil, err
	}
	config.Name = name
	return config, nil
}

func b64Decode(content string) (data string, err error) {
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
	}

	for _, enc := range encodings {
		if data, err = _b64Decode(content, enc); err == nil {
			break
		}
	}
	return
}

func _b64Decode(content string, encoding *base64.Encoding) (string, error) {
	dec, err := encoding.DecodeString(content)
	if err != nil {
		if _, ok := err.(base64.CorruptInputError); ok {
			err = errors.New("base64 input is corrupt, check service Key")
		}
		return "", err
	}
	return string(dec), nil
}
