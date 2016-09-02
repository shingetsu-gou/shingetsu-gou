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

package record

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/shingetsu-gou/shingetsu-gou/cfg"
	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//Head represents one line in updatelist/recentlist
type Head struct {
	Datfile string //cache file name
	Stamp   int64  //unixtime
	ID      string //md5(bodystr)
}

func newHead() *Head {
	return &Head{}
}

//NewHeadFromLine parse one line in udpate/recent list and returns updateInfo obj.
func NewHeadFromLine(line string) (*Head, error) {
	regnum := regexp.MustCompile(`^\d+$`)
	reghex := regexp.MustCompile(`^[0-9a-z]+$`)
	strs := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
	if len(strs) < 3 || util.FileDecode(strs[2]) == "" || !strings.HasPrefix(strs[2], "thread") ||
		!regnum.MatchString(strs[0]) || !reghex.MatchString(strs[1]) {
		err := errors.New("illegal format")
		log.Println(err)
		return nil, err
	}

	u := &Head{
		ID:      strs[1],
		Datfile: strs[2],
	}
	var err error
	u.Stamp, err = strconv.ParseInt(strs[0], 10, 64)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return u, nil
}

//path returns path for real file
func (u *Head) path() string {
	if u.Idstr() == "" || u.Datfile == "" {
		return ""
	}
	return filepath.Join(cfg.CacheDir, u.dathash(), "record", u.Idstr())
}

//rmPath returns path for removed marker
func (u *Head) rmPath() string {
	if u.Idstr() == "" || u.Datfile == "" {
		return ""
	}
	return filepath.Join(cfg.CacheDir, u.dathash(), "removed", u.Idstr())
}

//dathash returns the same string as Datfile if encoding=asis
func (u *Head) dathash() string {
	if u.Datfile == "" {
		return ""
	}
	return util.FileHash(u.Datfile)
}

//Exists return true if record file exists.
func (u *Head) Exists() bool {
	return util.IsFile(u.path())
}

//Removed return true if record is removed (i.e. exists.in removed path)
func (u *Head) Removed() bool {
	return util.IsFile(u.rmPath())
}

//Remove moves the record file  to remove path
func (u *Head) Remove() error {
	cfg.Fmutex.Lock()
	err := util.MoveFile(u.path(), u.rmPath())
	cfg.Fmutex.Unlock()
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

//Equals returns true if u=v
func (u *Head) Equals(rec *Head) bool {
	return u.Datfile == rec.Datfile && u.ID == rec.ID && u.Stamp == rec.Stamp
}

//Hash returns md5 of Head.
func (u *Head) Hash() [16]byte {
	return md5.Sum([]byte(u.Recstr()))
}

//Recstr returns one line of update/recentlist file.
func (u *Head) Recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", u.Stamp, u.ID, u.Datfile)
}

//Idstr returns real file name of the record file.
func (u *Head) Idstr() string {
	return fmt.Sprintf("%d_%s", u.Stamp, u.ID)
}

type Heads []*Head

//Less returns true if stamp of infos[i] < [j]
func (r Heads) Less(i, j int) bool {
	return r[i].Stamp < r[j].Stamp
}

//Swap swaps infos order.
func (r Heads) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

//Len returns size of infos
func (r Heads) Len() int {
	return len(r)
}

//has returns true if Heads has rec.
func (r Heads) has(rec *Head) bool {
	for _, v := range r {
		if v.Equals(rec) {
			return true
		}
	}
	return false
}

func ParseHeadResponse(res []string, datfile string) map[string]*Head {
	m := make(map[string]*Head)
	for _, line := range res {
		strs := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
		if len(strs) < 2 {
			log.Println("illegal format")
			return nil
		}
		u := &Head{
			ID:      strs[1],
			Datfile: datfile,
		}
		var err error
		u.Stamp, err = strconv.ParseInt(strs[0], 10, 64)
		if err != nil {
			log.Println(err)
			return nil
		}
		m[u.Idstr()] = u
	}
	return m
}
