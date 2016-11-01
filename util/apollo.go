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

//sign based on http://shingetsu.info/protocol/protocol-0.5-2.pdf page 8

package util

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"log"
	"math/big"
)

var (
	rsaPublicE      = big.NewInt(0x10001)
	rsaCreateGiveup = 300
	sprpTestCount   = 10

	base64en = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	base64de = []int64{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 62, 0, 0, 0, 63,
		52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 0, 0, 0, 0, 0, 0,
		0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
		15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 0, 0, 0, 0, 0,
		0, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
		41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
)

//primize makes x a prime number.
//Result will be bigger than x.
func primize(x *big.Int) *big.Int {
	var tmp big.Int
	if x.Bit(0) == 0 {
		x.Add(x, tmp.SetInt64(1))
	}
	for {
		if x.ProbablyPrime(sprpTestCount) {
			return x
		}
		x.Add(x, tmp.SetInt64(2))
	}
}

//PrivateKey reppresents private key,
type PrivateKey struct {
	keyN *big.Int //public part
	keyD *big.Int //private part
}

//GetKeys returns base64 encoded private key
func (p *PrivateKey) GetKeys() (string, string) {
	return intToBase64(p.keyN), intToBase64(p.keyD)
}

//newPrivateKey makes private key from seeds.
func newPrivateKey(pSeed, qSeed big.Int) (*PrivateKey, error) {
	q := &qSeed
	p := &pSeed
	var tmp big.Int
	test := big.NewInt(0x7743)
	var q1, phi, keyD, keyN big.Int
	for count := 0; count < rsaCreateGiveup; count++ {
		q = primize(q)
		q1.Add(q, tmp.SetInt64(-1))
		p = primize(p)
		phi.Add(p, tmp.SetInt64(-1))
		phi.Mul(&phi, &q1)
		keyD.ModInverse(rsaPublicE, &phi)
		if keyD.Cmp(tmp.SetInt64(0)) == 0 {
			continue
		}
		keyN.Mul(p, q)
		tmp.Exp(test, rsaPublicE, &keyN)
		tmp.Exp(&tmp, &keyD, &keyN)
		if tmp.Cmp(test) == 0 {
			return &PrivateKey{&keyN, &keyD}, nil
		}
		p.Add(p, tmp.SetInt64(2))
		q.Add(q, tmp.SetInt64(2))
	}
	err := errors.New("cannot generate private key")
	log.Fatal(err)
	return nil, err
}

//base64ToInt makes int from string.
func base64ToInt(s string) *big.Int {
	var tmp big.Int
	sb := []byte(s)
	for i := len(sb) - 1; i >= 0; i-- {
		b := big.NewInt(base64de[sb[i]])
		tmp.Lsh(&tmp, 6).Or(&tmp, b)
	}
	return &tmp
}

//intToBase64 makes string from int.
func intToBase64(n *big.Int) string {
	var result string
	and := big.NewInt(0x3f)
	var tmp, nn big.Int
	nn.Set(n)

	for nn.Cmp(big.NewInt(0)) > 0 {
		bit := tmp.And(&nn, and).Uint64()
		result += string(base64en[bit])
		nn.Rsh(&nn, 6)
	}
	return result + string(base64en[0]*byte(86-len(result)))
}

//Sign signs mesg by p.
func (p *PrivateKey) Sign(mesg string) string {
	var enc, m big.Int
	setBytesReverse(&m, []byte(mesg))
	enc.Exp(&m, p.keyD, p.keyN)
	return intToBase64(&enc)
}

//Verify verifies testsig by publicKey.
func Verify(mesg, testsig, publicKey string) bool {
	if len(mesg)*4 > len(publicKey)*3 {
		return false
	}
	var m, decrypted big.Int
	setBytesReverse(&m, []byte(mesg))
	n := base64ToInt(publicKey)
	intSig := base64ToInt(testsig)
	decrypted.Exp(intSig, rsaPublicE, n)

	return decrypted.Cmp(&m) == 0
}

//CutKey cuts key to 11words.
//used in a func which is used in templates.
func CutKey(key string) string {
	digest := md5.Sum([]byte(key))
	k := base64.StdEncoding.EncodeToString(digest[:])[:11]
	return k
}

func setBytesReverse(b *big.Int, d []byte) *big.Int {
	buf := make([]byte, len(d))
	for i := 0; i < len(d); i++ {
		buf[len(d)-i-1] = d[i]
	}
	return b.SetBytes(buf)
}

//MakePrivateKey makes privatekey from keystr
func MakePrivateKey(keystr string) (*PrivateKey, error) {
	var seedbuf [64]byte
	seed1 := md5.Sum([]byte(keystr))
	seed2 := md5.Sum([]byte(keystr + "pad1"))
	seed3 := md5.Sum([]byte(keystr + "pad2"))
	seed4 := md5.Sum([]byte(keystr + "pad3"))

	copy(seedbuf[0:16], seed1[:])
	copy(seedbuf[16:32], seed2[:])
	copy(seedbuf[32:48], seed3[:])
	copy(seedbuf[48:64], seed4[:])

	var p, q big.Int
	setBytesReverse(&p, seedbuf[0:28])
	setBytesReverse(&q, seedbuf[28:64])
	p.SetBit(&p, 215, 1)
	q.SetBit(&q, 279, 1)
	return newPrivateKey(p, q)
}
