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
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//record represents one record.
type record struct {
	datfile              string //cache file name
	Stamp                int64  //unixtime
	ID                   string //md5(bodystr)
	contents             map[string]string
	keyOrder             []string
	noNeedToLoadAttached bool
}

//len returns size of contents
func (r *record) len() int {
	return len(r.contents)
}

//newRecord parse idstr unixtime+"_"+md5(bodystr)), set stamp and id, and return record obj.
//if parse failes returns nil.
func newRecord(datfile, idstr string) *record {
	var err error
	r := &record{datfile: datfile}
	if idstr != "" {
		buf := strings.Split(idstr, "_")
		if len(buf) != 2 {
			log.Println(idstr, ":bad format")
			return nil
		}
		if r.Stamp, err = strconv.ParseInt(buf[0], 10, 64); err != nil {
			log.Println(idstr, ":bad format")
			return nil
		}
		r.ID = buf[1]
	}
	return r
}

//Idstr returns real file name of the record file.
func (r *record) Idstr() string {
	return fmt.Sprintf("%d_%s", r.Stamp, r.ID)
}

//recstr returns one line in the record file.
func (r *record) recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", r.Stamp, r.ID, r.bodystr())
}

//bodystr returns body part of one line in the record file.
func (r *record) bodystr() string {
	rs := make([]string, len(r.contents))
	for i, k := range r.keyOrder {
		rs[i] = k + ":" + r.contents[k]
	}
	return strings.Join(rs, "<>")
}

//HasBodyValue returns true if key k exists
func (r *record) HasBodyValue(k string) bool {
	if _, exist := r.contents[k]; exist {
		return true
	}
	return false
}

//getBodyValue returns value of key k
//return def if not exists.
func (r *record) GetBodyValue(k string, def string) string {
	if v, exist := r.contents[k]; exist {
		return v
	}
	return def
}

//path returns path for real file
func (r *record) path() string {
	if r.Idstr() == "" || r.datfile == "" {
		return ""
	}
	return filepath.Join(cacheDir, r.dathash(), "record", r.Idstr())
}

//bodyPath returns path for body (record without attach field)
func (r *record) bodyPath() string {
	if r.Idstr() == "" || r.datfile == "" {
		return ""
	}
	return filepath.Join(cacheDir, r.dathash(), "body", r.Idstr())
}

//rmPath returns path for removed marker
func (r *record) rmPath() string {
	if r.Idstr() == "" || r.datfile == "" {
		return ""
	}
	return filepath.Join(cacheDir, r.dathash(), "removed", r.Idstr())
}

//dathash returns the same string as datfile if encoding=asis
func (r *record) dathash() string {
	if r.datfile == "" {
		return ""
	}
	return fileHash(r.datfile)
}

//Exists return true if record file exists.
func (r *record) Exists() bool {
	return isFile(r.path())
}

//parse parses one line in record file and set params to record r.
func (r *record) parse(recstr string) error {
	var err error
	recstr = strings.TrimSpace(recstr)
	tmp := strings.Split(recstr, "<>")
	if len(tmp) < 2 {
		err := errors.New(recstr + ":bad format")
		log.Println(err)
		return err
	}
	r.Stamp, err = strconv.ParseInt(tmp[0], 10, 64)
	if err != nil {
		log.Println(tmp[0], "bad format")
		return err
	}
	r.ID = tmp[1]
	r.contents = make(map[string]string)
	r.keyOrder = make([]string, len(tmp))
	for i, kv := range tmp {
		buf := strings.Split(kv, ":")
		if len(buf) < 2 {
			continue
		}
		buf[1] = strings.Replace(buf[1], "<br>", "\n", -1)
		buf[1] = strings.Replace(buf[1], "<", "&lt;", -1)
		buf[1] = strings.Replace(buf[1], ">", "&gt;", -1)
		buf[1] = strings.Replace(buf[1], "\n", "<br>", -1)
		r.keyOrder[i] = buf[0]
		r.contents[buf[0]] = buf[1]
	}
	if r.contents["attach"] != "-1" {
		r.noNeedToLoadAttached = true
	}
	return nil
}

//size returns real file size of record.
func (r *record) size() int64 {
	s, err := os.Stat(r.path())
	if err != nil {
		log.Println(err)
		return 0
	}
	return s.Size()
}

//remove moves the record file  to remove path
//and removes all thumbnails ,attached files and body files.
func (r *record) remove() error {
	err := moveFile(r.path(), r.rmPath())
	if err != nil {
		log.Println(err)
		return err
	}
	for _, path := range r.allthumbnailPath() {
		err := os.Remove(path)
		if err != nil {
			log.Println(err)
		}
	}
	err = os.Remove(r.attachPath("", ""))
	if err != nil {
		log.Println(err)
	}
	err = os.Remove(r.bodyPath())
	if err != nil {
		log.Println(err)
	}
	return nil
}

//_load loads a record file and parses it.
func (r *record) _load(filename string) error {
	if r.size() <= 0 {
		err := r.remove()
		if err != nil {
			log.Println(err)
		}
		return errors.New("file not found")
	}
	c, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	return r.parse(string(c))
}

//load loads a record file if file is attached and parses it.
func (r *record) load() error {
	if r.noNeedToLoadAttached {
		return nil
	}
	return r._load(r.path())
}

//loadBody loads a body file if not yet and parses it.
func (r *record) loadBody() error {
	if r.contents != nil {
		return nil
	}
	if isFile(r.bodyPath()) {
		return r._load(r.bodyPath())
	}
	return r.load()
}

//build sets params in record from args and return id.
func (r *record) build(stamp int64, body map[string]string, passwd string) string {
	r.contents = make(map[string]string)
	r.keyOrder = make([]string, len(body))
	r.Stamp = stamp
	i := 0
	var targets string
	for key, value := range body {
		r.contents[key] = value
		r.keyOrder[i] = key
		i++
	}
	if passwd != "" {
		k := makePrivateKey(passwd)
		pubkey, _ := k.getKeys()
		md := md5digest(r.bodystr())
		sign := k.sign(md)
		r.contents["pubkey"] = pubkey
		r.contents["sign"] = sign
		r.contents["target"] = targets
		r.keyOrder = append(r.keyOrder, "pubkey")
		r.keyOrder = append(r.keyOrder, "sign")
		r.keyOrder = append(r.keyOrder, "target")
	}
	r.ID = md5digest(r.bodystr())
	return r.ID
}

//md5check return true if md5 of bodystr is same as r.id.
func (r *record) md5check() bool {
	return md5digest(r.bodystr()) == r.ID
}

//allthumnailPath finds and returns all thumbnails path in disk
func (r *record) allthumbnailPath() []string {
	if r.path() == "" {
		log.Println("null file name")
		return nil
	}
	dir := filepath.Join(cacheDir, r.dathash(), "attach")
	var thumbnail []string
	err := eachFiles(dir, func(fi os.FileInfo) error {
		dname := fi.Name()
		if strings.HasPrefix(dname, "s"+r.Idstr()) {
			thumbnail = append(thumbnail, filepath.Join(dir, dname))
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return thumbnail
}

//attachPath returns attach path
//if suffix !="" create path from args.
//if suffix =="" find file starting with idstr and returns its name.
//if thumbnailSize!="" find thumbnail.
func (r *record) attachPath(suffix string, thumbnailSize string) string {
	if r.path() == "" {
		log.Println("null file name")
		return ""
	}
	dir := filepath.Join(cacheDir, r.dathash(), "attach")
	if suffix != "" {
		reg := regexp.MustCompile("[^-_.A-Za-z0-9]")
		reg.ReplaceAllString(suffix, "")
		if suffix == "" {
			suffix = "txt"
		}
		if thumbnailSize != "" {
			return filepath.Join(dir, "s"+r.Idstr()+"."+thumbnailSize+"."+suffix)
		}
		return filepath.Join(dir, r.Idstr()+"."+suffix)
	}
	var result string
	err := eachFiles(dir, func(fi os.FileInfo) error {
		dname := fi.Name()
		if strings.HasPrefix(dname, r.Idstr()) {
			result = filepath.Join(dir, dname)
		}
		return nil
	})
	if err != nil {
		return ""
	}
	return result
}

//makeThumbnail fixes suffix,thumbnailSize and calls makeThumbnailInternal.
func (r *record) makeThumbnail(suffix string, thumbnailSize string) {
	if thumbnailSize == "" {
		thumbnailSize = defaultThumbnailSize
	}
	if thumbnailSize == "" {
		return
	}
	if suffix == "" {
		suffix = r.GetBodyValue("suffix", "jpg")
	}

	attachPath := r.attachPath(suffix, "")
	thumbnailPath := r.attachPath(suffix, thumbnailSize)
	if !isDir(thumbnailPath) {
		return
	}
	size := strings.Split(thumbnailSize, "x")
	if len(size) != 2 {
		return
	}
	x, err1 := strconv.Atoi(size[0])
	y, err2 := strconv.Atoi(size[1])
	if err1 != nil || err2 != nil {
		log.Println(thumbnailSize, "is illegal format")
		return
	}
	makeThumbnail(attachPath, thumbnailPath, suffix, uint(x), uint(y))
}

//saveAttached decodes base64 v and saves to attached , make and save thumbnail
func (r *record) saveAttached(v string, force bool) {
	attach, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		log.Println("cannot decode attached file")
		return
	}
	attachPath := r.attachPath(r.GetBodyValue("suffix", "txt"), "")
	thumbnailPath := r.attachPath(r.GetBodyValue("suffix", "jpg"), defaultThumbnailSize)

	if err = writeFile(attachPath, string(attach)); err != nil {
		log.Println(err)
		return
	}
	if force || !isFile(thumbnailPath) {
		r.makeThumbnail("", "")
	}
}

//sync saves recstr to the file. if attached file exists, saves it to attached path.
//and save body part to body path. if signed, also saves body part.
func (r *record) sync(force bool) {
	if isFile(r.rmPath()) {
		return
	}
	if force || !isFile(r.path()) {
		err := writeFile(r.path(), r.recstr()+"\n")
		if err != nil {
			log.Println(err)
		}
	}
	body := r.bodyString() + "\n"
	if v, exist := r.contents["attach"]; exist {
		if err := writeFile(r.bodyPath(), body); err != nil {
			log.Println(err)
			return
		}
		r.saveAttached(v, force)
	}
	if _, exist := r.contents["sign"]; exist {
		err := writeFile(r.bodyPath(), body)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

//bodyString retuns bodystr not including attach field, and shorten pubkey.
func (r *record) bodyString() string {
	buf := []string{
		strconv.FormatInt(r.Stamp, 10),
		r.ID,
	}
	for _, k := range r.keyOrder {
		switch k {
		case "attach":
			buf = append(buf, "attach:1")
		case "sign":
		case "pubkey":
			if r.checkSign() {
				shortKey := cutKey(r.contents["pubkey"])
				buf = append(buf, "pubkey:"+shortKey)
			}
		default:
			buf = append(buf, k+":"+r.contents[k])
		}
	}
	return strings.Join(buf, "<>")
}

//checkSign check signature in the record is valid.
func (r *record) checkSign() bool {
	for _, k := range []string{"pubkey", "sign", "target"} {
		if _, exist := r.contents[k]; !exist {
			return false
		}
	}
	ts := strings.Split(r.contents["target"], ",")
	targets := make([]string, len(ts))
	for i, t := range ts {
		if _, exist := r.contents[t]; !exist {
			return false
		}
		targets[i] = t + ":" + r.contents[t]
	}
	md := md5digest(strings.Join(targets, "<>"))
	if verify(md, r.contents["sign"], r.contents["pubkey"]) {
		return true
	}
	return false
}

//meets checks the record meets condisions of args
func (r *record) meets(i string, stamp int64, id string, begin, end int64) bool {
	return r.parse(i) == nil &&
		(stamp != 0 || r.Stamp == stamp) && (id != "" || r.ID == id) &&
		begin <= r.Stamp && (end <= 0 || r.Stamp <= end) && r.md5check()
}

//getRecords gets the records which have id=head from n
func getRecords(datfile string, n *node, head []string) []string {
	var result []string
	for _, h := range head {
		rec := newRecord(datfile, strings.Replace(strings.TrimSpace(h), "<>", "_", -1))
		if !isFile(rec.path()) && !isFile(rec.rmPath()) {
			res, err := n.talk(fmt.Sprintf("/get/%s/%d/%s", datfile, rec.Stamp, rec.ID))
			if err != nil {
				log.Println("get", err)
				return nil
			}
			result = append(result, strings.TrimSpace(res[0]))
		}
	}
	return result
}
