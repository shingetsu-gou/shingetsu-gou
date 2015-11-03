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

package thread

import (
	"crypto/md5"
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

	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//RecordHead represents one line in updatelist/recentlist
type RecordHead struct {
	Datfile string //cache file name
	Stamp   int64  //unixtime
	ID      string //md5(bodystr)
}

//newUpdateInfoFromLine parse one line in udpate/recent list and returns updateInfo obj.
func newRecordHeadFromLine(line string) (*RecordHead, error) {
	strs := strings.Split(strings.TrimRight(line, "\n\r"), "<>")
	if len(strs) < 3 || util.FileDecode(strs[2]) == "" || !strings.HasPrefix(strs[2], "thread") {
		err := errors.New("illegal format")
		log.Println(err)
		return nil, err
	}
	u := &RecordHead{
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

//equals returns true if u=v
func (u *RecordHead) equals(rec *RecordHead) bool {
	return u.Datfile == rec.Datfile && u.ID == rec.ID && u.Stamp == rec.Stamp
}

//hash returns md5 of RecordHead.
func (u *RecordHead) hash() [16]byte {
	return md5.Sum([]byte(u.Recstr()))
}

//Recstr returns one line of update/recentlist file.
func (u *RecordHead) Recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", u.Stamp, u.ID, u.Datfile)
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

//RecordCfg is the config for Record struct.
//it must be set befor using.
var (
	RecordCfg *RecordConfig
	recordMap = make(map[string]sync.Pool)
)

//RecordConfig is the config for Record struct.
type RecordConfig struct {
	DefaultThumbnailSize string
	CacheDir             string
	Fmutex               *sync.RWMutex
	CachedRule           *util.RegexpList
}

//Record represents one record.
type Record struct {
	*RecordConfig
	RecordHead
	contents map[string]string
	keyOrder []string
	isLoaded bool
	mutex    sync.RWMutex
}

//NewRecord parse idstr unixtime+"_"+md5(bodystr)), set stamp and id, and return record obj.
//if parse failes returns nil.
func NewRecord(datfile, idstr string) *Record {
	if RecordCfg == nil {
		log.Fatal("must set RecordCfg")
	}
	p, exist := recordMap[datfile]
	if !exist {
		p.New = func() interface{} {
			return &Record{
				RecordConfig: RecordCfg,
			}
		}
	}
	r := p.Get().(*Record)
	p.Put(r)

	var err error

	r.Datfile = datfile
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

//CopyRecordHead copies and  returns recordhead.
func (r *Record) CopyRecordHead() RecordHead {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return RecordHead{
		Datfile: r.Datfile,
		Stamp:   r.Stamp,
		ID:      r.ID,
	}
}

//len returns size of contents
func (r *Record) len() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.contents)
}

//makeRecords makes and returns record from Recstr
func makeRecord(line string) *Record {
	line = strings.TrimRight(line, "\r\n")
	buf := strings.Split(line, "<>")
	if len(buf) <= 2 || buf[0] == "" || buf[1] == "" || buf[2] == "" {
		return nil
	}
	idstr := buf[0] + "_" + buf[1]
	if util.FileDecode(buf[2]) == "" || !strings.HasPrefix(buf[2], "thread_") {
		//		log.Println("illegal format",buf[2])
		return nil
	}
	vr := NewRecord(buf[2], idstr)
	if err := vr.parse(line); err != nil {
		log.Println(err)
		return nil
	}
	return vr
}

//bodystr returns body part of one line in the record file.
func (r *Record) bodystr() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	rs := make([]string, len(r.contents))
	for i, k := range r.keyOrder {
		rs[i] = k + ":" + r.contents[k]
	}
	return strings.Join(rs, "<>")
}

//HasBodyValue returns true if key k exists
func (r *Record) HasBodyValue(k string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if _, exist := r.contents[k]; exist {
		return true
	}
	return false
}

//GetBodyValue returns value of key k
//return def if not exists.
func (r *Record) GetBodyValue(k string, def string) string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if v, exist := r.contents[k]; exist {
		return v
	}
	return def
}

//path returns path for real file
func (r *Record) path() string {
	if r.Idstr() == "" || r.Datfile == "" {
		return ""
	}
	return filepath.Join(r.CacheDir, r.dathash(), "record", r.Idstr())
}

//rmPath returns path for removed marker
func (r *Record) rmPath() string {
	if r.Idstr() == "" || r.Datfile == "" {
		return ""
	}
	return filepath.Join(r.CacheDir, r.dathash(), "removed", r.Idstr())
}

//dathash returns the same string as Datfile if encoding=asis
func (r *Record) dathash() string {
	if r.Datfile == "" {
		return ""
	}
	return util.FileHash(r.Datfile)
}

//Exists return true if record file exists.
func (r *Record) Exists() bool {
	return util.IsFile(r.path())
}

//Recstr returns one line in the record file.
func (r *Record) Recstr() string {
	return fmt.Sprintf("%d<>%s<>%s", r.Stamp, r.ID, r.bodystr())
}

//parse parses one line in record file and response of /recent/ and set params to record r.
func (r *Record) parse(Recstr string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var err error
	Recstr = strings.TrimRight(Recstr, "\r\n")
	tmp := strings.Split(Recstr, "<>")
	if len(tmp) < 2 {
		err := errors.New(Recstr + ":bad format")
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
	r.isLoaded = true
	return nil
}

//size returns real file size of record.
func (r *Record) size() int64 {
	r.Fmutex.RLock()
	defer r.Fmutex.RUnlock()
	s, err := os.Stat(r.path())
	if err != nil {
		log.Println(err)
		return 0
	}
	return s.Size()
}

//Remove moves the record file  to remove path
func (r *Record) Remove() error {
	r.Fmutex.Lock()
	err := util.MoveFile(r.path(), r.rmPath())
	r.Fmutex.Unlock()
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

//Load loads a record file and parses it.
func (r *Record) Load() error {
	r.Fmutex.RLock()
	defer r.Fmutex.RUnlock()

	if r.isLoaded {
		return nil
	}

	if !r.Exists() {
		err := r.Remove()
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

//ShortPubkey returns short version of pubkey.
func (r *Record) ShortPubkey() string {
	if v, exist := r.contents["pubkey"]; exist {
		return util.CutKey(v)
	}
	return ""
}

//Build sets params in record from args and return id.
func (r *Record) Build(stamp int64, body map[string]string, passwd string) string {
	r.contents = make(map[string]string)
	r.keyOrder = make([]string, len(body))
	r.Stamp = stamp
	i := 0
	r.mutex.Lock()
	for key, value := range body {
		r.contents[key] = value
		r.keyOrder[i] = key
		i++
	}
	r.mutex.Unlock()
	if passwd != "" {
		k := util.MakePrivateKey(passwd)
		pubkey, _ := k.GetKeys()
		md := util.MD5digest(r.bodystr())
		sign := k.Sign(md)
		r.mutex.Lock()
		r.contents["pubkey"] = pubkey
		r.contents["sign"] = sign
		r.contents["target"] = strings.Join(r.keyOrder, ",")
		r.keyOrder = append(r.keyOrder, "pubkey")
		r.keyOrder = append(r.keyOrder, "sign")
		r.keyOrder = append(r.keyOrder, "target")
		r.mutex.Unlock()
	}

	id := util.MD5digest(r.bodystr())
	r.mutex.Lock()
	r.ID = id
	r.isLoaded = true
	r.mutex.Unlock()
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.ID
}

//md5check return true if md5 of bodystr is same as r.id.
func (r *Record) md5check() bool {
	return util.MD5digest(r.bodystr()) == r.ID
}

//AttachPath returns attach path
//by creating path from args.
func (r *Record) AttachPath(thumbnailSize string) string {
	if r.path() == "" {
		log.Println("null file name")
		return ""
	}
	suffix := r.GetBodyValue("suffix", "")
	if suffix == "" {
		return ""
	}
	dir := filepath.Join(r.CacheDir, r.dathash(), "attach")
	reg := regexp.MustCompile(`[^-_.A-Za-z0-9]`)
	reg.ReplaceAllString(suffix, "")
	if thumbnailSize != "" {
		return filepath.Join(dir, "s"+r.Idstr()+"."+thumbnailSize+"."+suffix)
	}
	return filepath.Join(dir, r.Idstr()+"."+suffix)
}

//Sync saves Recstr to the file. if attached file exists, saves it to attached path.
//if signed, also saves body part.
func (r *Record) Sync() {
	if util.IsFile(r.rmPath()) {
		return
	}
	if !util.IsFile(r.path()) {
		r.Fmutex.Lock()
		err := util.WriteFile(r.path(), r.Recstr()+"\n")
		r.Fmutex.Unlock()
		if err != nil {
			log.Println(err)
		}
	}
}

//Getbody retuns contents of rec after loading if needed.
func (r *Record) Getbody() string {
	err := r.Load()
	if err != nil {
		log.Println(err)
	}
	return r.Recstr()
}

//checkSign check signature in the record is valid.
func (r *Record) checkSign() bool {
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
	md := util.MD5digest(strings.Join(targets, "<>"))
	if util.Verify(md, r.contents["sign"], r.contents["pubkey"]) {
		return true
	}
	return false
}

//meets checks the record meets conditions of args
func (r *Record) meets(i string, stamp int64, id string, begin, end int64) bool {

	if r.parse(i) != nil {
		log.Println("parse NG")
		return false
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
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

//IsSpam returns true if Recstr is listed in spam.txt
func (r *Record) IsSpam() bool {
	return r.CachedRule.Check(r.Recstr())
}

//MakeAttachLink makes and returns attached file link.
func (r *Record) MakeAttachLink(sakuHost string) string {
	if r.GetBodyValue("attach", "") == "" {
		return ""
	}
	url := fmt.Sprintf("http://%s/thread.cgi/%s/%s/%d.%s",
		sakuHost, r.Datfile, r.ID, r.Stamp, r.GetBodyValue("suffix", "txt"))
	return "<br><br>[Attached]<br>" + url
}

//RecordMap is a map key=datfile, value=record.
type RecordMap map[string]*Record

//Get returns records which hav key=i.
//return def if not found.
func (r RecordMap) Get(i string, def *Record) *Record {
	if v, exist := r[i]; exist {
		return v
	}
	return def
}

//Keys returns key strings(ids) of records
func (r RecordMap) Keys() []string {
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
func (r RecordMap) removeRecords(limit int64, saveSize int) {
	ids := r.Keys()
	if saveSize < len(ids) {
		ids = ids[:len(ids)-saveSize]
		if limit > 0 {
			for _, re := range ids {
				if r[re].Stamp+limit < time.Now().Unix() {
					err := r[re].Remove()
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
		if util.IsFile(rec.path()) {
			if _, exist := once[rec.ID]; exist {
				err := rec.Remove()
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
