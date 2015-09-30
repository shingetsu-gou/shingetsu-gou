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

package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	fout, err := os.Create(filepath.Base(os.Args[1]))
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()
	fout.WriteString("{{$root=.}}\n")
	scanner := bufio.NewScanner(f)
	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		line = strings.Replace(line, "{{", "{{.", -1)
		line = strings.Replace(line, "{%", "{{", -1)
		line = strings.Replace(line, "%}", "}}", -1)
		line = strings.Replace(line, "endif", "end", -1)
		line = strings.Replace(line, "endfor", "end", -1)
		line = strings.Replace(line, " for ", " range $", -1)
		line = strings.Replace(line, " in ", ":=", -1)
		for i := 'a'; i <= 'z'; i++ {
			b := i - 'a' + 'A'
			line = strings.Replace(line, "."+string(i), "."+string(b), -1)
			line = strings.Replace(line, ":= "+string(i), ":=$root."+string(b), -1)
		}
		fout.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
