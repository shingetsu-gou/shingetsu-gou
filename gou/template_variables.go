/*
 * Copyright (c) 2015, Shinya Yagyu
 * All rights reserved.
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its
 *    contributors may be used to endorse or promote products derived from this
 *    software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package gou

import (
	"log"
	"net/url"
	"path"
	"regexp"
	"strings"

	"golang.org/x/text/language"
)

//message hold string map.
type message map[string]string

//newMessage reads from the file excpet #comment and stores them with url unescaping value.
func newMessage(file string) message {
	var m message
	re := regexp.MustCompile("^\\s*#")
	err := eachLine(file, func(line string, i int) error {
		line = strings.Trim(line, "\r\n")
		var err error
		if re.MatchString(line) {
			return nil
		}
		buf := strings.Split(line, "<>")
		if len(buf) == 2 {
			buf[1], err = url.QueryUnescape(buf[1])
			m[buf[0]] = buf[1]
		}
		return err
	})
	if err != nil {
		log.Println(file, err)
	}
	return m
}

//searchMessage parse Accept-Language header ,selects most-weighted(biggest q)
//language ,reads the associated message file, and creates and returns message obj.
func searchMessage(acceptLanguage string) message {
	var lang []string
	if acceptLanguage != "" {
		tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
		if err != nil {
			log.Println(err)
		} else {
			for _, tag := range tags {
				lang = append(lang, tag.String())
			}
		}
	}
	lang = append(lang, defaultLanguage)
	for _, l := range lang {
		slang := strings.Split(l, "-")[0]
		for _, j := range []string{l, slang} {
			file := path.Join(fileDir, "message-"+j+".txt")
			if isFile(file) {
				return newMessage(file)
			}
		}
	}
	return nil
}

//GatewayLink is a struct for gateway_link.txt
type GatewayLink struct {
	Message     message
	CGIname     string
	Command     string
	Description string
}

//SetupStruct setups GatewayLink struct to render gateway_link.txt
//GatewayLink.Message must be setted up previously.
func (c *GatewayLink) SetupStruct(cginame, command string) *GatewayLink {
	c.CGIname = cginame
	c.Command = command
	c.Description = c.Message["desc_"+command]
	return c
}

//ListItem is for list_item.txt
type ListItem struct {
	Cache      *cache
	Title      string
	Tags       *tagList
	Sugtags    []*tag
	Target     string
	Remove     bool
	StrOpts    string
	IsAdmin    bool
	GatewayCGI string
	Appli      map[string]string
	filter     string
	tag        string
}

//checkCache checks cache ca has specified tag and datfile doesn't contains filterd string.
func (l *ListItem) checkCache(ca *cache, target string) (string, bool) {
	x := fileDecode(ca.Datfile)
	if x == "" {
		return "", false
	}
	if l.filter != "" && !strings.Contains(l.filter, strings.ToLower(x)) {
		return "", false
	}
	if l.tag != "" {
		switch {
		case ca.tags.hasTagstr(strings.ToLower(l.tag)):
		case target == "recent" && ca.sugtags.hasTagstr(strings.ToLower(l.tag)):
		default:
			return "", false
		}
	}
	return x, true
}

//SetupStruct setups ListItem struct to render list_item.txt
//ListItem.IsAdmin,filter,tag must be setted up previously.
func (l *ListItem) SetupStruct(ca *cache, remove bool, target string, search bool) ListItem {
	x, ok := l.checkCache(ca, target)
	if !ok {
		return *l
	}
	x = escapeSpace(x)
	var strOpts string
	if search {
		strOpts = "?search_new_file=yes"
	}
	var sugtags []*tag
	if target == "recent" {
		strTags := make([]string, ca.tags.Len())
		for i, v := range ca.tags.Tags {
			strTags[i] = strings.ToLower(v.Tagstr)
		}
		for _, st := range ca.sugtags.Tags {
			if !hasString(strTags, strings.ToLower(st.Tagstr)) {
				sugtags = append(sugtags, st)
			}
		}
	}
	l.Cache = ca
	l.Title = x
	l.Tags = ca.tags
	l.Sugtags = sugtags
	l.Target = target
	l.Remove = remove
	l.StrOpts = strOpts
	l.GatewayCGI = gatewayURL
	l.Appli = application
	return *l
}
