[![Build Status](https://travis-ci.org/shingetsu-gou/shingetsu-gou.svg?branch=master)](https://travis-ci.org/shingetsu-gou/shingetsu-gou)
[![GoDoc](https://godoc.org/github.com/shingetsu-gou/shingetsu-gou?status.svg)](https://godoc.org/github.com/shingetsu-gou/shingetsu-gou)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/shingetsu-gou/shingetsu-gou/master/LICENSE)


# Gou(合) 

## Overview

Gou（[合](https://ja.wikipedia.org/wiki/%E5%90%88_%28%E5%A4%A9%E6%96%87%29)) is an _unofficial_ clone of P2P anonymous BBS shinGETsu [saku](https://github.com/shingetsu/saku) in golang.

The word "Gou(合)" means [conjunction](https://en.wikipedia.org/wiki/Astrological_aspect) in Japanese, when an aspect is an angle the planets make to each other in the horoscope.

Yeah, the sun and moon are in conjunction during the new moon(shingetsu:新月, saku:朔）.

Refer [here](http://www.shingetsu.info/) for more details about shinGETsu.


## Feature

1. Setting files are compatible with ones of saku 4.6.1.
2. You can use cache of saku without modification, but you cannot cache of Gou with saku.
   But there is --sakurifice option to convert Gou cache.
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

## Installation

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

# Note

Files 

* in template/ directory
* in www/ directory
* in file/ directory

are embeded into the exexutable binary in https://github.com/shingetsu-gou/shingetsu-gou/releases.
If these files are not found on your disk, Gou automatically expands these to the disk
(not overwrite). Once expanded, you can change these files as you wish.

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


