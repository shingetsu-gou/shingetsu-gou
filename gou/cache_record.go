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
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nfnt/resize"
)

type record struct {
	recstr       string
	datfile      string
	stamp        int64
	id           string
	idstr        string
	path         string
	bodyPath     string
	rmPath       string
	flagLoad     bool
	flagLoadBody bool
	dathash      string
	dict         map[string]string
}

func newRecord(datfile, idstr string) *record {
	var err error
	r := &record{
		datfile: datfile,
		idstr:   idstr,
		dict:    make(map[string]string),
	}
	if idstr != "" {
		buf := strings.Split(idstr, "_")
		if len(buf) != 2 {
			log.Println(idstr, ":bad format")
			return r
		}
		if r.stamp, err = strconv.ParseInt(buf[0], 10, 64); err != nil {
			log.Println(idstr, ":bad format")
			return r
		}
		r.id = buf[1]
	}
	r.setpath()
	return r
}
func (r *record) exists() bool {
	return isFile(r.path)
}

func (r *record) Len() int {
	return len(r.dict)
}

func (r *record) Get(k string, def string) string {
	if v, exist := r.dict[k]; exist {
		return v
	}
	return def
}

func (r *record)add(k,v string){
	r.dict[k]=v	
}

func (r *record) virtualRecordString() string {
	return strings.Join([]string{strconv.FormatInt(r.stamp, 10), r.id, r.datfile}, "<>")
}
func (r *record) virtualRecordEqual(rr *record) bool {
	return r.stamp == rr.stamp && r.id == rr.id && r.datfile == rr.datfile
}

func (r *record) free() {
	r.flagLoad = false
	r.flagLoadBody = false
	r.recstr = ""
	r.dict = make(map[string]string)
}

func (r *record) gt(y *record) bool {
	if r.stamp != y.stamp {
		return r.stamp > y.stamp
	}
	return r.idstr > y.idstr
}

func (r *record) lt(y *record) bool {
	if r.stamp != y.stamp {
		return r.stamp < y.stamp
	}
	return r.idstr < y.idstr
}

func (r *record) setpath() {
	if r.idstr == "" || r.datfile == "" {
		return
	}
	r.dathash = fileHash(r.datfile)
	r.path = filepath.Join(cache_dir, r.dathash, "record", r.idstr)
	r.bodyPath = filepath.Join(cache_dir, r.dathash, "body", r.idstr)
	r.rmPath = filepath.Join(cache_dir, r.dathash, "removed", r.idstr)
}

func (r *record) parse(recstr string) error {
	var err error
	repl := strings.NewReplacer("\r", "", "\n", "")
	r.recstr = repl.Replace(r.recstr)
	tmp := strings.Split(r.recstr, "<>")
	if len(tmp) < 2 {
		err := errors.New(r.recstr + ":bad format")
		log.Println(err)
		return err
	}
	r.dict["stamp"] = tmp[0]
	r.dict["id"] = tmp[1]
	r.idstr = r.dict["stamp"] + "_" + r.dict["id"]
	r.stamp, err = strconv.ParseInt(r.dict["stamp"], 10, 64)
	if err != nil {
		log.Println(tmp[0], "bad format")
		return err
	}
	r.id = r.dict["id"]
	for _, i := range tmp {
		buf := strings.Split(i, ":")
		if len(buf) < 2 {
			continue
		}
		buf[1] = strings.Replace(buf[1], "<br>", "\n", -1)
		buf[1] = strings.Replace(buf[1], "<", "&lt;", -1)
		buf[1] = strings.Replace(buf[1], ">", "&gt;", -1)
		buf[1] = strings.Replace(buf[1], "\n", "<br>", -1)
		r.dict[buf[0]] = buf[1]
	}
	if s, ok := r.dict["attach"]; !ok || s != "1" {
		r.flagLoad = true
	}
	r.flagLoadBody = true
	r.setpath()
	return nil
}

func (r *record) size() int64 {
	s, err := os.Stat(r.path)
	if err != nil {
		log.Println(err)
		return 0
	}
	return s.Size()
}

func (r *record) remove() error {
	err := moveFile(r.path, r.rmPath)
	if err != nil {
		log.Println(err)
		r.free()
		return err
	}
	for _, path := range r.allthumbnailPath() {
		os.Remove(path)
	}
	os.Remove(r.attachPath("", ""))
	os.Remove(r.bodyPath)
	r.free()
	return nil
}

func (r *record) _load(filename string) error {
	if r.size() <= 0 {
		r.remove()
		return errors.New("file not found")
	}
	c, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	return r.parse(string(c))
}

func (r *record) load() error {
	if !r.flagLoad {
		return r._load(r.path)
	}
	return nil
}

func (r *record) loadBody() error {
	if r.flagLoadBody {
		return nil
	}
	if isFile(r.bodyPath) {
		return r._load(r.bodyPath)
	}
	return r.load()
}

func (r *record) build(stamp int64, body map[string]string, passwd string) string {
	bodyary := make([]string, len(body))
	i := 0
	var targets string
	for key, value := range body {
		bodyary[i] = key + ":" + value
		r.dict[key] = value
		targets += key
		if i < len(body)-1 {
			targets += ","
		}
		i++
	}
	bodystr := strings.Join(bodyary, "<>")
	if passwd != "" {
		k := makePrivateKey(passwd)
		pubkey, _ := k.getKeys()
		md := md5digest(bodystr)
		sign := k.sign(md)
		r.dict["pubkey"] = pubkey
		r.dict["sign"] = sign
		r.dict["target"] = targets
		bodystr += "<>pubkey:" + pubkey + "<>sign:" + sign + "<>target:" + targets
	}
	id := md5digest(bodystr)
	r.stamp = stamp
	s := strconv.FormatInt(stamp, 10)
	r.recstr = strings.Join([]string{s, id, bodystr}, "<>")
	r.idstr = s + "_" + id
	r.dict["stamp"] = s
	r.dict["id"] = id
	r.id = id
	r.setpath()
	return id
}

func (r *record) md5check() bool {
	buf := strings.Split(r.recstr, "<>")
	if len(buf) > 2 {
		return md5digest(buf[2]) == r.dict["id"]
	}
	return false
}

func (r *record) allthumbnailPath() []string {
	dir := filepath.Join(cache_dir, r.dathash, "attach")
	thumbnail := make([]string, 0)
	err := eachFiles(dir, func(fi os.FileInfo) error {
		dname := fi.Name()
		if strings.HasPrefix(dname, "s"+r.idstr) {
			thumbnail = append(thumbnail, filepath.Join(dir, dname))
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return thumbnail
}

func (r *record) attachPath(suffix string, thumbnailSize string) string {
	dir := filepath.Join(cache_dir, r.dathash, "attach")
	if suffix != "" {
		if thumbnailSize != "" {
			return filepath.Join(dir, "/", "s"+r.idstr+"."+thumbnailSize+"."+suffix)
		}
		return filepath.Join(dir, "/", r.idstr+"."+suffix)
	}
	var result string
	err := eachFiles(dir, func(fi os.FileInfo) error {
		dname := fi.Name()
		if strings.HasPrefix(dname, r.idstr) {
			result = filepath.Join(dir, dname)
		}
		return nil
	})
	if err != nil {
		return ""
	}
	return result
}

func (r *record) attachSize(path, suffix, thumbnailSize string) int64 {
	if path == "" {
		path = r.attachPath(suffix, thumbnailSize)
	}
	st, err := os.Stat(path)
	if err != nil {
		log.Println(err)
		return 0
	}
	return st.Size()
}

func (r *record) writeFile(path, data string) error {
	err := ioutil.WriteFile(path, []byte(data), 0666)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (r *record) makeThumbnail(suffix string, thumbnailSize string) {
	if thumbnailSize == "" {
		thumbnailSize = thumbnail_size
	}
	if thumbnailSize == "" {
		return
	}
	if suffix == "" {
		suffix = r.getDict("suffix", "jpg")
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
	file, err := os.Open(attachPath)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		log.Println(err)
		return
	}
	m := resize.Resize(uint(x), uint(y), img, resize.Lanczos3)
	out, err := os.Create(thumbnailPath)
	if err != nil {
		log.Println(err)
		return
	}
	defer out.Close()
	switch suffix {
	case "jpg", "jpeg":
		jpeg.Encode(out, m, nil)
	case "png":
		png.Encode(out, m)
	case "gif":
		gif.Encode(out, m, nil)
	default:
		log.Println("illegal format", suffix)
	}
}

func (r *record) getDict(key, def string) string {
	if v, exist := r.dict[key]; exist {
		return v
	}
	return def
}

func (r *record) sync(force bool) {
	if isFile(r.rmPath) {
		return
	}
	if force || !isFile(r.path) {
		r.writeFile(r.path, r.recstr+"\n")
	}
	body := r.bodyString() + "\n"
	if v, exist := r.dict["attach"]; exist {
		attach, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			log.Println("cannot decode attached file")
			return
		}
		attachPath := r.attachPath(r.getDict("suffix", "txt"), "")
		thumbnailPath := r.attachPath(r.getDict("suffix", "jpg"), thumbnail_size)
		r.writeFile(r.bodyPath, body)
		r.writeFile(attachPath, string(attach))
		if force || !isFile(thumbnailPath) {
			r.makeThumbnail("", "")
		}
	}
	if _, exist := r.dict["sign"]; exist {
		r.writeFile(r.bodyPath, body)
	}
}

//Remove attach field
func (r *record) bodyString() string {
	buf := []string{r.dict["stamp"], r.dict["id"]}
	for _, k := range sortKeys(r.dict) {
		switch k {
		case "stamp", "id":
		case "attach":
			buf = append(buf, "attach:1")
		case "sign":
		case "pubkey":
			if r.checkSign() {
				shortKey := cutKey(r.dict["pubkey"])
				buf = append(buf, "pubkey:"+shortKey)
			}
		default:
			buf = append(buf, k+":"+r.dict[k])
		}
	}
	return strings.Join(buf, "<>")
}

func (r *record) checkSign() bool {
	for _, k := range []string{"pubkey", "sign", "target"} {
		if hasString(stringSlice(sortKeys(r.dict)), k) {
			return false
		}
	}
	target := ""
	for _, t := range strings.Split(r.dict["target"], ",") {
		if _, exist := r.dict[t]; !exist {
			return false
		}
		target += "<>" + t + ":" + r.dict[t]
	}
	target = target[2:] // remove ^<>

	md := md5digest(target)
	if verify(md, r.dict["sign"], r.dict["pubkey"]) {
		return true
	}
	return false
}

func getRecords(datfile string, n *node, head []string) []string {
	result := make([]string, 0)
	for _, h := range head {
		rec := newRecord(datfile, strings.Replace(strings.TrimSpace(h), "<>", "_", -1))
		if !isFile(rec.path) && !isFile(rec.rmPath) {
			res, err := n.talk("/get/" + datfile + "/" + strconv.FormatInt(rec.stamp, 10) + "/" + rec.id)
			if err != nil {
				log.Println("get", err)
				return nil
			}
			result = append(result, strings.TrimSpace(res[0]))
		}
	}
	return result
}
