[![Build Status](https://travis-ci.org/shingetsu-gou/shingetsu-gou.svg?branch=master)](https://travis-ci.org/shingetsu-gou/shingetsu-gou)
[![GoDoc](https://godoc.org/github.com/shingetsu-gou/shingetsu-gou?status.svg)](https://godoc.org/github.com/shingetsu-gou/shingetsu-gou)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/shingetsu-gou/shingetsu-gou/master/LICENSE)


# Gou(合) 

tl;dr

## 特徴
* Go言語で開発
* sakuと設定ファイル互換
* cacheは非互換（sakuで使ってるファイルの一部を使っていないため）だが変換コマンドで対応可能
* ポータブル：各プラットフォーム別に実行ファイル1個
* 省メモリ（ざっくりsakuの７割～５割くらい？）
* 速度は早いかもしれない。 

## 新規に始める
1.  [ここ](https://github.com/shingetsu-gou/shingetsu-gou/releases)から自分のOSの実行バイナリダウンロード、展開
3. ./shingetsu-gouで実行
4. ブラウザでhttp://localhost:8000/をアクセス
5. uPnP/NAT-PMPでポートを自動で開ける努力はしますが、不可の場合、朔同様自分で開けてください。

## 朔から移行
1.  [ここ](https://github.com/shingetsu-gou/shingetsu-gou/releases)から自分のOSの実行バイナリダウンロード、展開
2. 合を実行するディレクトリに実行ファイル(shingetsu-gou)と、朔のcacheディレクトリとfileディレクトリをコピー（朔のcache/fileディレクトリは別に残しておくことを推奨します）
3. ./shingetsu-gouで実行

## 合のキャッシュを朔で使う（朔に戻る）
1. 合のキャッシュをバックアップ（推奨）
2. 合の実行ディレクトリで./shingetsu-gou --sakurifice実行
3. 合のcacheディレクトリとfileディレクトリを朔のディレクトリにコピー


## Overview

Gou（[合](https://ja.wikipedia.org/wiki/%E5%90%88_%28%E5%A4%A9%E6%96%87%29)) is an _unofficial_ clone of P2P anonymous BBS shinGETsu [saku](https://github.com/shingetsu/saku) in golang.

The word "Gou(合)" means [conjunction](https://en.wikipedia.org/wiki/Astrological_aspect) in Japanese, when an aspect is an angle the planets make to each other in the horoscope.

Yeah, the sun and moon are in conjunction during the new moon(shingetsu:新月, saku:朔）.

Refer [here](http://www.shingetsu.info/) for more details about shinGETsu.


## Feature

1. Setting files are compatible with ones of saku 4.6.1.
2. You can use cache of saku without modification, but you cannot use cache of Gou with saku.
   But there is --sakurifice option to convert Gou cache to saku one.
2. Gou uses less (about half of ) memory usage than saku.
3. Portable because there is only one binary file for each platforms and no need to prepare runtime. 
   Just download and click one binary to run.
4. (should be) faster than saku (?).

## Platform
  * MacOS darwin/Plan9 on i386
  * Windows/OpenBSD on i386/amd64
  * Linux/NetBSD/FreeBSD on i386/amd64/arm
  * Solaris on amd64

## Requirements

* git
* go 1.4+

are required to compile.

## Compile

    $ mkdir gou
    $ cd gou
    $ mkdir src
    $ mkdir bin
    $ mkdir pkg
    $ exoprt GOPATH=`pwd`
    $ go get github.com/shingetsu-gou/shingetsu-gou
	
Or you can download executable binaries from [here](https://github.com/shingetsu-gou/shingetsu-gou/releases).

## Command Options
```
shingetsu-gou <options>
  -sakurifice
        makes caches compatible with saku
  -silent
        suppress logs
  -v    print logs
  -verbose
        print logs
```

# Differences from Original Saku

1. mch(2ch interface) listens to the same port as admin.cgi/gateway.cgi/serve.cgi/thread.cgi. dat_port setting in config is ignored.
3. Gou can try to open port by uPnP and NAT-PMP. You can enable this function by setting enable_nat:true in [Gateway]  in saku.ini, which is false by default, but is true in attached saku.ini in binary.
4. URL for 2ch interface /2ch_hoehoe/subject.txt in saku is /2ch/hoehoe/subject.txt in Gou.
5. files in template directory are not compatible with Gou and Saku. The default template directory name in Gou is "gou_template/".
6. Duplicate files are not used. i.e. files below are not used in Gou. If you want to use saku after using Gou to same cache files, you must run gou command once with --sakurifice option before using skau to complement some indispensable files.
	* in cache directory
		* body directory
		* attach directory
		* count.stat
		* dat.stat
		* size.stat
		* stamp.stat
		* validstamp.stat
		* velocity.stat
	* in run directory
		* client.txt
		* node.txt
		* search.txt
		* tag.txt
		* update.txt
7. dnsname in config.py is same as server_name in saku.ini in Gou.
8. Gou has moonlight-like function (I believe), _heavymoon_. Add [Gateway] moonlight:true in saku.ini if you want to use. THIS FUNCTION IS NOT RECOMMENDED because of _heavy_ network load.
9. Contents of some links are embed into the thread. If you don't like it you can disable by [Gateway] enable_embed:false.

# Note

Files 

* in template/ directory
* in www/ directory
* in file/ directory

are embeded into the exexutable binary in https://github.com/shingetsu-gou/shingetsu-gou/releases.
If files in file/ dir are not found on your disk, Gou automatically expands these to the disk
(but not overwrite). Other files are not expanded.
But you can add files to www/ or  template/ if you wish. These files will override embded ones.

This is for easy to use Gou; just get a binary, and run it!

## License

MIT License

Original program comes from [saku](https://github.com/shingetsu/saku), which is under [2-clause BSD license](https://github.com/shingetsu/saku/blob/master/LICENSE)
Copyrighted by 2005-2015 shinGETsu Project.

See also

 * www/bootstrap/css/bootstrap.min.css
 * www/jquery/MIT-LICENSE.txt
 * www/jquery/jquery.min.js
 * www/jquery/jquery.lazy.min.js
 * www/jquery/spoiler/authors.txt

# Contribution

Improvements to the codebase and pull requests are encouraged.
