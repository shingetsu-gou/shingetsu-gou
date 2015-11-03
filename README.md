[![Build Status](https://travis-ci.org/shingetsu-gou/shingetsu-gou.svg?branch=master)](https://travis-ci.org/shingetsu-gou/shingetsu-gou)
[![GoDoc](https://godoc.org/github.com/shingetsu-gou/shingetsu-gou?status.svg)](https://godoc.org/github.com/shingetsu-gou/shingetsu-gou)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/shingetsu-gou/shingetsu-gou/master/LICENSE)


# Gou(合) 

## Overview

Gou（[合](https://ja.wikipedia.org/wiki/%E5%90%88_%28%E5%A4%A9%E6%96%87%29)) is a clone of P2P anonymous BBS shinGETsu [saku](https://github.com/shingetsu/saku) in golang.

The word "Gou(合)" means [conjunction](https://en.wikipedia.org/wiki/Astrological_aspect) in Japanese, when an aspect is an angle the planets make to each other in the horoscope.

Yeah, the sun and moon are in conjunction during the new moon(shingetsu:新月, saku:朔）.

Refer [here](http://www.shingetsu.info/) for more details about shinGETsu.

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

# Differences from Original Saku

1. mch(2ch interface) listens to the same port as admin.cgi/gateway.cgi/serve.cgi/thread.cgi. dat_port setting in config is ignored.
3. Gou can try to open port by uPnP and NAT-PMP. You can enable this function by setting enable_nat:true in [Gateway]  in saku.ini, which is false by default, but is true in attached saku.ini in binary.
4. URL for 2ch interface /2ch_hoehoe/subject.txt in saku is /2ch/hoehoe/subject.txt in Gou.
5. files in template directory are not compatible with Gou and Saku. The default template directory name in Gou is "gou_template/".
6. files below are not used in Gou.
	* in cache directory
		* body directory
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
7. dnsname is now same as server_name in config .

# Note

Files 

* in template/ directory
* in www/ directory
* in file/ directory

are embeded into the exexutable binary in https://github.com/shingetsu-gou/shingetsu-gou/releases.
If these files are not found on your disk, Gou automatically expands these to the disk.
Once expanded, you can change these files as you wish.

This is for easy-use of Gou; just get a binary, and run it!

# Contribution

Improvements to the codebase and pull requests are encouraged.


