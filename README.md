[![Build Status](https://travis-ci.org/shingetsu-gou/shingetsu-gou.svg?branch=master)](https://travis-ci.org/shingetsu-gou/shingetsu-gou)
[![GoDoc](https://godoc.org/github.com/shingetsu-gou/shingetsu-gou?status.svg)](https://godoc.org/github.com/shingetsu-gou/shingetsu-gou)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/shingetsu-gou/shingetsu-gou/master/LICENSE)


# Gou(合) 

## Overview

Gou（[合](https://ja.wikipedia.org/wiki/%E5%90%88_%28%E5%A4%A9%E6%96%87%29)) is a clone of P2P anonymous BBS shinGETsu saku in golang.

The word "Gou(合)" means [conjunction](https://en.wikipedia.org/wiki/Astrological_aspect) in Japanese, when an aspect is an angle the planets make to each other in the horoscope.

Yeah, the sun and moon are in conjunction during the new moon(新月, 朔）.


## License

MIT License

Original Program comes from [saku](https://github.com/shingetsu/saku), which is under [2-clause BSD license](https://github.com/shingetsu/saku/blob/master/LICENSE)
Copyrighted by 2005-2015 shinGETsu Project.

See also

 * www/bootstrap/css/bootstrap.min.css
 * www/jquery/MIT-LICENSE.txt
 * www/jquery/jquery.min.js
 * www/jquery/jquery.lazy.min.js
 * www/jquery/spoiler/authors.txt


## Requirements

This requires

* git
* go 1.4+


## Installation

    $ mkdir gou
    $ cd gou
    $ mkdir src
    $ mkdir bin
    $ mkdir pkg
    $ exoprt GOPATH=`pwd`
    $ go get github.com/shingetsu-gou/gou

# Differences from Original Saku

1. mch(2ch interface) listens to the same port as admin.cgi/gateway.cgi/serve.cgi/thread.cgi. dat_port setting in config.txt is ignored.

# Contribution

Improvements to the codebase and pull requests are encouraged.


