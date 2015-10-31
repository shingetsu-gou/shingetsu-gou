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
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//RecordHead represents one line in updatelist/recentlist
type RecordHead struct {
	datfile string //cache file name
	Stamp   int64  //unixtime
	ID      string //md5(bodystr)
}

//newUpdateInfoFromLine parse one line in udpate/recent list and returns updateInfo obj.
func newRecordHeadFromLine(line string) (*RecordHead, error) {
	strs := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
	if len(strs) < 3 {
		err := errors.New("illegal format")
		log.Println(err)
		return nil, err
	}
	u := &RecordHead{
		ID:      strs[1],
		datfile: strs[2],
	}
	var err error
	u.Stamp, err = strconv.ParseInt(strs[0], 10, 64)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return u, nil
}

//equals returns true if u=v
func (u *RecordHead) equals(rec *RecordHead) bool {
	return u.datfile == rec.datfile && u.ID == rec.ID && u.Stamp == rec.Stamp
}

//hash returns md5 of RecordHead.
func (u *RecordHead) hash() [16]byte {
	m := md5.New()
	m.Write([]byte(u.datfile))
	binary.Write(m, binary.LittleEndian, u.Stamp)
	m.Write([]byte(u.ID))
	var r [16]byte
	m.Sum(r[:])
	return r
}

//recstr returns one line of update/recentlist file.
func (u *RecordHead) recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", u.Stamp, u.ID, u.datfile)
}

//Idstr returns real file name of the record file.
func (u *RecordHead) Idstr() string {
	return fmt.Sprintf("%d_%s", u.Stamp, u.ID)
}

type recordHeads []*RecordHead

//Less returns true if stamp of infos[i] < [j]
func (r recordHeads) Less(i, j int) bool {
	return r[i].Stamp < r[j].Stamp
}

//Swap swaps infos order.
func (r recordHeads) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

//Len returns size of infos
func (r recordHeads) Len() int {
	return len(r)
}

//has returns true if recordHeads has rec.
func (r recordHeads) has(rec *RecordHead) bool {
	for _, v := range r {
		if v.equals(rec) {
			return true
		}
	}
	return false
}

//record represents one record.
type record struct {
	*Config
	RecordHead
	contents map[string]string
	keyOrder []string
	mutex    sync.RWMutex
}

//len returns size of contents
func (r *record) len() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.contents)
}

//newRecord parse idstr unixtime+"_"+md5(bodystr)), set stamp and id, and return record obj.
//if parse failes returns nil.
func newRecord(datfile, idstr string, c *Config) *record {
	var err error
	r := &record{
		Config: c,
	}
	r.datfile = datfile
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

//makeRecords makes and returns record from recstr
func makeRecord(line string, cfg *Config) *record {
	line = strings.TrimRight(line, "\r\n")
	buf := strings.Split(line, "<>")
	if len(buf) <= 2 || buf[0] == "" || buf[1] == "" || buf[2] == "" {
		return nil
	}
	idstr := buf[0] + "_" + buf[1]
	vr := newRecord(buf[2], idstr, cfg)
	if err := vr.parse(line); err != nil {
		log.Println(err)
		return nil
	}
	return vr
}

//bodystr returns body part of one line in the record file.
func (r *record) bodystr() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	rs := make([]string, len(r.contents))
	for i, k := range r.keyOrder {
		rs[i] = k + ":" + r.contents[k]
	}
	return strings.Join(rs, "<>")
}

//HasBodyValue returns true if key k exists
func (r *record) HasBodyValue(k string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if _, exist := r.contents[k]; exist {
		return true
	}
	return false
}

//getBodyValue returns value of key k
//return def if not exists.
func (r *record) GetBodyValue(k string, def string) string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

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
	return filepath.Join(r.CacheDir, r.dathash(), "record", r.Idstr())
}

//rmPath returns path for removed marker
func (r *record) rmPath() string {
	if r.Idstr() == "" || r.datfile == "" {
		return ""
	}
	return filepath.Join(r.CacheDir, r.dathash(), "removed", r.Idstr())
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
	return IsFile(r.path())
}

//parse parses one line in record file and response of /recent/ and set params to record r.
func (r *record) parse(recstr string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var err error
	recstr = strings.TrimRight(recstr, "\r\n")
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
	r.keyOrder = nil
	//reposense of recentlist  : stamp<>id<>thread_***<>tag:***
	//record str : stamp<>id<>body:***<>...
	for _, kv := range tmp[2:] {
		buf := strings.SplitN(kv, ":", 2)
		if len(buf) < 2 {
			continue
		}
		buf[1] = strings.Replace(buf[1], "<br>", "\n", -1)
		buf[1] = strings.Replace(buf[1], "<", "&lt;", -1)
		buf[1] = strings.Replace(buf[1], ">", "&gt;", -1)
		buf[1] = strings.Replace(buf[1], "\n", "<br>", -1)
		r.keyOrder = append(r.keyOrder, buf[0])
		r.contents[buf[0]] = buf[1]
	}
	return nil
}

//size returns real file size of record.
func (r *record) size() int64 {
	r.Fmutex.RLock()
	defer r.Fmutex.RUnlock()
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
	r.Fmutex.Lock()
	defer r.Fmutex.Unlock()
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
	return nil
}

//load loads a record file and parses it.
func (r *record) load() error {
	r.Fmutex.RLock()
	defer r.Fmutex.RUnlock()

	if !r.Exists() {
		err := r.remove()
		if err != nil {
			log.Println(err)
		}
		return errors.New("file not found")
	}
	c, err := ioutil.ReadFile(r.path())
	if err != nil {
		log.Println(err)
		return err
	}
	return r.parse(string(c))
}

//build sets params in record from args and return id.
func (r *record) build(stamp int64, body map[string]string, passwd string) string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.contents = make(map[string]string)
	r.keyOrder = make([]string, len(body))
	r.Stamp = stamp
	i := 0
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
		r.contents["target"] = strings.Join(r.keyOrder, ",")
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
	r.Fmutex.RLock()
	defer r.Fmutex.RUnlock()
	if r.path() == "" {
		log.Println("null file name")
		return nil
	}
	dir := filepath.Join(r.CacheDir, r.dathash(), "attach")
	var thumbnail []string
	err := eachFiles(dir, func(fi os.FileInfo) error {
		dname := fi.Name()
		if strings.HasPrefix(dname, "s"+r.Idstr()) {
			thumbnail = append(thumbnail, filepath.Join(dir, dname))
		}
		return nil
	})
	if err != nil {
		log.Println(err)
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
	dir := filepath.Join(r.CacheDir, r.dathash(), "attach")
	if suffix != "" {
		reg := regexp.MustCompile(`[^-_.A-Za-z0-9]`)
		reg.ReplaceAllString(suffix, "")
		if suffix == "" {
			suffix = "txt"
		}
		if thumbnailSize != "" {
			return filepath.Join(dir, "s"+r.Idstr()+"."+thumbnailSize+"."+suffix)
		}
		return filepath.Join(dir, r.Idstr()+"."+suffix)
	}
	r.Fmutex.RLock()
	defer r.Fmutex.RUnlock()
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
		thumbnailSize = r.DefaultThumbnailSize
	}
	if thumbnailSize == "" {
		return
	}
	if suffix == "" {
		suffix = r.GetBodyValue("suffix", "jpg")
	}

	attachPath := r.attachPath(suffix, "")
	thumbnailPath := r.attachPath(suffix, thumbnailSize)
	log.Println(attachPath, thumbnailPath)
	if IsFile(thumbnailPath) {
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
	r.Fmutex.Lock()
	defer r.Fmutex.Unlock()
	makeThumbnail(attachPath, thumbnailPath, suffix, uint(x), uint(y))
}

//saveAttached decodes base64 v and saves to attached , make and save thumbnail
func (r *record) saveAttached(v string) {
	r.Fmutex.Lock()
	attach, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		log.Println("cannot decode attached file")
		return
	}
	attachPath := r.attachPath(r.GetBodyValue("suffix", "txt"), "")
	thumbnailPath := r.attachPath(r.GetBodyValue("suffix", "jpg"), r.DefaultThumbnailSize)

	if err = writeFile(attachPath, string(attach)); err != nil {
		log.Println(err)
		return
	}
	r.Fmutex.Unlock()
	if !IsFile(thumbnailPath) {
		r.makeThumbnail("", "")
	}
}

//sync saves recstr to the file. if attached file exists, saves it to attached path.
//and save body part to body path. if signed, also saves body part.
func (r *record) sync() {
	r.Fmutex.Lock()
	defer r.Fmutex.Unlock()

	if IsFile(r.rmPath()) {
		return
	}
	if !IsFile(r.path()) {
		err := writeFile(r.path(), r.recstr()+"\n")
		if err != nil {
			log.Println(err)
		}
	}
	if v, exist := r.contents["attach"]; exist {
		r.saveAttached(v)
	}
}

//bodyString retuns bodystr not including attach field, and shorten pubkey.
func (r *record) bodyString() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

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
	r.mutex.RLock()
	defer r.mutex.RUnlock()

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
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.parse(i) != nil {
		log.Println("parse NG")
		return false
	}
	if stamp > 0 && r.Stamp != stamp {
		log.Println("stamp NG", r.Stamp, stamp)
		return false
	}
	if id != "" && r.ID != id {
		log.Println("id NG", id, r.ID)
		return false
	}
	if begin > r.Stamp || (end > 0 && r.Stamp > end) {
		log.Println("stamp range NG", begin, end, r.Stamp)
		return false
	}
	if !r.md5check() {
		log.Println("md5 NG")
		return false
	}
	return true
}

//isSpam returns true if recstr is listed in spam.txt
func (r *record) isSpam() bool {
	return cachedRule.check(r.recstr())
}

type recordMap map[string]*record

//get returns records which hav key=i.
//return def if not found.
func (r recordMap) get(i string, def *record) *record {
	if v, exist := r[i]; exist {
		return v
	}
	return def
}

//keys returns key strings(ids) of records
func (r recordMap) keys() []string {
	ks := make([]string, len(r))
	i := 0
	for k := range r {
		ks[i] = k
		i++
	}
	sort.Strings(ks)
	return ks
}

//removeRecords remove old records while remaing #saveSize records.
//and also removes duplicates recs.
func (r recordMap) removeRecords(limit int64, saveSize int) {
	ids := r.keys()
	if saveSize < len(ids) {
		ids = ids[:len(ids)-saveSize]
		if limit > 0 {
			for _, re := range ids {
				if r[re].Stamp+limit < time.Now().Unix() {
					err := r[re].remove()
					if err != nil {
						log.Println(err)
					}
					delete(r, re)
				}
			}
		}
	}
	once := make(map[string]struct{})
	for k, rec := range r {
		if IsFile(rec.path()) {
			if _, exist := once[rec.ID]; exist {
				err := rec.remove()
				if err != nil {
					log.Println(err)
				}
				delete(r, k)
			} else {
				once[rec.ID] = struct{}{}
			}
		}
	}
}
