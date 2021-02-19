package util

import "regexp"

type Group struct {
	Key   string
	Value string
}

func ParseGroup(regexp *regexp.Regexp, str string) []Group {
	names := regexp.SubexpNames()
	subMatch := regexp.FindStringSubmatch(str)
	if subMatch == nil {
		return nil
	}
	res := make([]Group, 0, len(names))
	for i, name := range names {
		if name != "" && subMatch[i] != "" {
			res = append(res, Group{name, subMatch[i]})
		}
	}
	return res
}
