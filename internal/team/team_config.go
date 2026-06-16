package team

import (
	"fmt"
	"regexp"
	"strings"
)

// TeamConfig is parsed from the team's hosted config file.
type TeamConfig struct {
	Username string
	PushURL  string
	Members  map[string][]string // group name → []username
}

// DeployConfig is the locally persisted team config for this machine.
type DeployConfig struct {
	Username string
	PushURL  string
	Members  map[string][]string
}

var (
	reKeyValue = regexp.MustCompile(`^\s*(\w+)\s*=\s*(.+)$`)
	reSection  = regexp.MustCompile(`^\s*\[(\w+)\]\s*$`)
	reComment  = regexp.MustCompile(`^\s*[#;]`)
)

// TeamConfigParser parses the INI/YAML-like team config format.
// Uses regex-based parsing; not a general YAML parser.
type TeamConfigParser struct{}

// Parse parses the raw team config content. Format:
//
//	username = alice
//	push_url = https://example.com/exports
//
//	[members]
//	group1 = alice, bob
//	group2 = charlie, dave
func (TeamConfigParser) Parse(raw string) (TeamConfig, error) {
	cfg := TeamConfig{Members: make(map[string][]string)}
	inMembers := false

	for line := range strings.SplitSeq(raw, "\n") {
		if reComment.MatchString(line) || strings.TrimSpace(line) == "" {
			continue
		}
		if m := reSection.FindStringSubmatch(line); m != nil {
			inMembers = strings.ToLower(m[1]) == "members"
			continue
		}
		m := reKeyValue.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, val := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		if inMembers {
			var members []string
			for u := range strings.SplitSeq(val, ",") {
				u = strings.TrimSpace(u)
				if u != "" {
					members = append(members, u)
				}
			}
			cfg.Members[key] = members
		} else {
			switch key {
			case "username":
				cfg.Username = val
			case "push_url":
				cfg.PushURL = val
			}
		}
	}

	if cfg.Username == "" {
		return TeamConfig{}, fmt.Errorf("team config missing required field: username")
	}
	return cfg, nil
}
