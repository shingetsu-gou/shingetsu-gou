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
	"errors"
	"net/http"
	"time"

	"github.com/axgle/mahonia"
)

var spamError = errors.New("this is spam")

func postComment(threadKey, name, mail, body, passwd, tag string) error {
	stamp := time.Now().Unix()
	recbody := make(map[string]string)
	recbody["body"] = cgiEscape(body, true)
	recbody["name"] = cgiEscape(name, true)
	recbody["mail"] = cgiEscape(mail, true)

	c := newCache(threadKey, nil, nil)
	rec := newRecord(c.datfile, "")
	id := rec.build(stamp, recbody, passwd)
	if spamCheck(rec.recstr) {
		return spamError
	}
	c.addData(rec, false)
	c.syncStatus()
	if tag != "" {
		saveTag(c, tag)
	}
	queue.append(c.datfile, stamp, id, nil)
	queue.run()
	return nil
}

func errorResp(msg string, wr http.ResponseWriter, host, name, mail, body string) string {
	info := map[string]string{
		"Message": msg,
		"Host":    host,
		"Name":    name,
		"Mail":    mail,
		"Body":    body,
	}
	str := executeTemplate("2ch_error", info)
	wr.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
	return mahonia.NewEncoder("cp932").ConvertString(str)
}

var successMsg = `<html lang="ja"><head><meta http-equiv="Content-Type" content="text/html"><title>書きこみました。</title></head>
<body>書きこみが終わりました。<br><br></body></html>`

func getCommentData(env map[string]string) []string {

}
