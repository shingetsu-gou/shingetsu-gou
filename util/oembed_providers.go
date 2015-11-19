//from https?://oembed.com/providers.json

package util

const oembedProviders = `
[
    {
        "provider_name": "IFTTT",
        "provider_url": "https?:\/\/www.ifttt.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/ifttt.com\/recipes\/*"
                ],
                "url": "http:\/\/www.ifttt.com\/oembed\/",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "YouTube",
        "provider_url": "https?:\/\/www.youtube.com\/",
        "endpoints": [
            {
                "url": "http:\/\/www.youtube.com\/oembed",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Flickr",
        "provider_url": "https?:\/\/www.flickr.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.flickr.com\/photos\/*",
                    "https?:\/\/flic.kr\/p\/*"
                ],
                "url": "http:\/\/www.flickr.com\/services\/oembed\/",
                "discovery": true
            }
        ]
    },

    {
        "provider_name": "Vimeo",
        "provider_url": "https?:\/\/vimeo.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/vimeo.com\/*",
                    "https?:\/\/vimeo.com\/groups\/*\/videos\/*",
                    "https:\/\/vimeo.com\/*",
                    "https:\/\/vimeo.com\/groups\/*\/videos\/*"
                ],
                "url": "http:\/\/vimeo.com\/api\/oembed.{format}"
            }
        ]
    },

    {
        "provider_name": "Embedly",
        "provider_url": "https?:\/\/api.embed.ly\/",
        "endpoints": [
            {
                "url": "http:\/\/api.embed.ly\/1\/oembed"
            }
        ]
    },

    {
        "provider_name": "SlideShare",
        "provider_url": "https?:\/\/www.slideshare.net\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.slideshare.net\/*\/*",
                    "https?:\/\/fr.slideshare.net\/*\/*",
                    "https?:\/\/de.slideshare.net\/*\/*",
                    "https?:\/\/es.slideshare.net\/*\/*",
                    "https?:\/\/pt.slideshare.net\/*\/*"
                ],
                "url": "http:\/\/www.slideshare.net\/api\/oembed\/2",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "WordPress.com",
        "provider_url": "https?:\/\/wordpress.com\/",
        "endpoints": [
            {
                "url": "http:\/\/public-api.wordpress.com\/oembed\/",
                "discovery": true
            }
        ]
    },

    {
        "provider_name": "Dailymotion",
        "provider_url": "https?:\/\/www.dailymotion.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.dailymotion.com\/video\/*"
                ],
                "url": "http:\/\/www.dailymotion.com\/services\/oembed"
            }
        ]
    },

    {
        "provider_name": "Instagram",
        "provider_url": "https:\/\/instagram.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/instagram.com\/p\/*",
                    "https?:\/\/instagr.am\/p\/*",
                    "https:\/\/instagram.com\/p\/*",
                    "https:\/\/instagr.am\/p\/*"
                ],
                "url": "http:\/\/api.instagram.com\/oembed",
                "formats": [
                    "json"
                ]
            }
        ]
    },
    {
        "provider_name": "SoundCloud",
        "provider_url": "https?:\/\/soundcloud.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/soundcloud.com\/*"
                ],
                "url": "https:\/\/soundcloud.com\/oembed"
            }
        ]
    },

    {
        "provider_name": "Kickstarter",
        "provider_url": "https?:\/\/www.kickstarter.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.kickstarter.com\/projects\/*"
                ],
                "url": "http:\/\/www.kickstarter.com\/services\/oembed"
            }
        ]
    },
    {
        "provider_name": "Ustream",
        "provider_url": "https?:\/\/www.ustream.tv",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.ustream.tv\/*",
                    "https?:\/\/*.ustream.com\/*"
                ],
                "url": "http:\/\/www.ustream.tv\/oembed",
                "formats": [
                    "json"
                ]
            }
        ]
    },
 
    {
        "provider_name": "Twitter",
        "provider_url": "https?:\/\/wtitter.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/twitter.com\/"
                ],
                "url": "https:\/\/api.twitter.com\/1\/statuses\/oembed.json",
                "discovery": true
            }
        ]
    },
	    {
        "provider_name": "Hatena",
        "provider_url": "https?:\/\/.*.hatenablog.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/.*.hatenablog.com\/"
                ],
                "url": "http:\/\/hatenablog.com\/oembed",
                "discovery": true
            }
        ]
    }

]
`

/*
   {
       "provider_name": "Viddler",
       "provider_url": "https?:\/\/www.viddler.com\/",
       "endpoints": [
           {
               "schemes": [
                   "https?:\/\/www.viddler.com\/v\/*"
               ],
               "url": "http:\/\/www.viddler.com\/oembed\/"
           }
       ]
   },
   {
       "provider_name": "Hulu",
       "provider_url": "https?:\/\/www.hulu.com\/",
       "endpoints": [
           {
               "schemes": [
                   "https?:\/\/www.hulu.com\/watch\/*"
               ],
               "url": "http:\/\/www.hulu.com\/api\/oembed.{format}"
           }
       ]
   },
    {
        "provider_name": "CollegeHumor",
        "provider_url": "https?:\/\/www.collegehumor.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.collegehumor.com\/video\/*"
                ],
                "url": "http:\/\/www.collegehumor.com\/oembed.{format}",
                "discovery": true
            }
        ]
    },
	   {
        "provider_name": "Daily Mile",
        "provider_url": "https?:\/\/www.dailymile.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.dailymile.com\/people\/*\/entries\/*"
                ],
                "url": "http:\/\/api.dailymile.com\/oembed?format=json",
                "formats": [
                    "json"
                ]
            }
        ]
    },
    {
        "provider_name": "Sketchfab",
        "provider_url": "https?:\/\/sketchfab.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/sketchfab.com\/models\/*",
                    "https:\/\/sketchfab.com\/models\/*",
                    "https:\/\/sketchfab.com\/*\/folders\/*"
                ],
                "url": "http:\/\/sketchfab.com\/oembed",
                "formats": [
                    "json"
                ]
            }
        ]
    },
    {
        "provider_name": "Meetup",
        "provider_url": "https?:\/\/www.meetup.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/meetup.com\/*",
                    "https?:\/\/meetu.ps\/*"
                ],
                "url": "https:\/\/api.meetup.com\/oembed",
                "formats": [
                    "json"
                ]
            }
        ]
    },
    {
        "provider_name": "AudioSnaps",
        "provider_url": "https?:\/\/audiosnaps.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/audiosnaps.com\/k\/*"
                ],
                "url": "http:\/\/audiosnaps.com\/service\/oembed",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "edocr",
        "provider_url": "https?:\/\/www.edocr.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/edocr.com\/docs\/*"
                ],
                "url": "http:\/\/edocr.com\/api\/oembed"
            }
        ]
    },
    {
        "provider_name": "RapidEngage",
        "provider_url": "https:\/\/rapidengage.com",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/rapidengage.com\/s\/*"
                ],
                "url": "https:\/\/rapidengage.com\/api\/oembed"
            }
        ]
    },
    {
        "provider_name": "Ora TV",
        "provider_url": "https?:\/\/www.ora.tv\/",
        "endpoints": [
            {
                "discovery": true,
                "url": "https:\/\/www.ora.tv\/oembed\/*?format={format}"
            }
        ]
    },
    {
        "provider_name": "Getty Images",
        "provider_url": "https?:\/\/www.gettyimages.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/gty.im\/*"
                ],
                "url": "http:\/\/embed.gettyimages.com\/oembed",
                "formats": [
                    "json"
                ]
            }
        ]
    },
    {
        "provider_name": "amCharts Live Editor",
        "provider_url": "https?:\/\/live.amcharts.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/live.amcharts.com\/*"
                ],
                "url": "http:\/\/live.amcharts.com\/oembed"
            }
        ]
    },
    {
        "provider_name": "iSnare Articles",
        "provider_url": "https:\/\/www.isnare.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/www.isnare.com\/*"
                ],
                "url": "https:\/\/www.isnare.com\/oembed\/"
            }
        ]
    },
    {
        "provider_name": "Infogram",
        "provider_url": "https:\/\/infogr.am\/",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/infogr.am\/*"
                ],
                "url": "https:\/\/infogr.am\/oembed"
            }
        ]
    },
    {
        "provider_name": "ChartBlocks",
        "provider_url": "https?:\/\/www.chartblocks.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/public.chartblocks.com\/c\/*"
                ],
                "url": "http:\/\/embed.chartblocks.com\/1.0\/oembed"
            }
        ]
    },
    {
        "provider_name": "ReleaseWire",
        "provider_url": "https?:\/\/www.releasewire.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/rwire.com\/*"
                ],
                "url": "http:\/\/publisher.releasewire.com\/oembed\/",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "They Said So",
        "provider_url": "https:\/\/theysaidso.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/theysaidso.com\/image\/*"
                ],
                "url": "https:\/\/theysaidso.com\/extensions\/oembed\/",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "LearningApps.org",
        "provider_url": "https?:\/\/learningapps.org\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/learningapps.org\/*"
                ],
                "url": "http:\/\/learningapps.org\/oembed.php",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "ShortNote",
        "provider_url": "https:\/\/www.shortnote.jp\/",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/www.shortnote.jp\/view\/notes\/*"
                ],
                "url": "https:\/\/www.shortnote.jp\/oembed\/",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Embed Articles",
        "provider_url": "https?:\/\/embedarticles.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/embedarticles.com\/*"
                ],
                "url": "http:\/\/embedarticles.com\/oembed\/"
            }
        ]
    },
    {
        "provider_name": "Topy",
        "provider_url": "https?:\/\/www.topy.se\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.topy.se\/image\/*"
                ],
                "url": "http:\/\/www.topy.se\/oembed\/",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "ReverbNation",
        "provider_url": "https:\/\/www.reverbnation.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/www.reverbnation.com\/*",
                    "https:\/\/www.reverbnation.com\/*\/songs\/*"
                ],
                "url": "https:\/\/www.reverbnation.com\/oembed",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Blackfire.io",
        "provider_url": "https:\/\/blackfire.io",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/blackfire.io\/profiles\/*\/graph",
                    "https:\/\/blackfire.io\/profiles\/compare\/*\/graph"
                ],
                "url": "https:\/\/blackfire.io\/oembed",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Oumy",
        "provider_url": "https:\/\/www.oumy.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/www.oumy.com\/v\/*"
                ],
                "url": "https:\/\/www.oumy.com\/oembed",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "EgliseInfo",
        "provider_url": "https?:\/\/egliseinfo.catholique.fr\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/egliseinfo.catholique.fr\/*"
                ],
                "url": "http:\/\/egliseinfo.catholique.fr\/api\/oembed",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "RepubHub",
        "provider_url": "https?:\/\/repubhub.icopyright.net\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/repubhub.icopyright.net\/freePost.act"
                ],
                "url": "http:\/\/repubhub.icopyright.net\/oembed.act",
                "discovery": true
            }
        ]
    },
	    {
        "provider_name": "On Aol",
        "provider_url": "https?:\/\/on.aol.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/on.aol.com\/video\/*"
                ],
                "url": "http:\/\/on.aol.com\/api"
            }
        ]
    },
	    {
        "provider_name": "Portfolium",
        "provider_url": "https:\/\/portfolium.com",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/portfolium.com\/entry\/*"
                ],
                "url": "https:\/\/api.portfolium.com\/oembed"
            }
        ]
    },
    {
        "provider_name": "iFixit",
        "provider_url": "https?:\/\/www.iFixit.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.ifixit.com\/Guide\/View\/*"
                ],
                "url": "http:\/\/www.ifixit.com\/Embed"
            }
        ]
    },
    {
        "provider_name": "SmugMug",
        "provider_url": "https?:\/\/www.smugmug.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.smugmug.com\/*"
                ],
                "url": "http:\/\/api.smugmug.com\/services\/oembed\/",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Deviantart.com",
        "provider_url": "https?:\/\/www.deviantart.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.deviantart.com\/art\/*",
                    "https?:\/\/*.deviantart.com\/*#\/d*",
                    "https?:\/\/fav.me\/*",
                    "https?:\/\/sta.sh\/*"
                ],
                "url": "http:\/\/backend.deviantart.com\/oembed"
            }
        ]
    },
	    {
        "provider_name": "chirbit.com",
        "provider_url": "https?:\/\/www.chirbit.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/chirb.it\/*"
                ],
                "url": "http:\/\/chirb.it\/oembed.{format}",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "nfb.ca",
        "provider_url": "https?:\/\/www.nfb.ca\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.nfb.ca\/film\/*"
                ],
                "url": "http:\/\/www.nfb.ca\/remote\/services\/oembed\/",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Scribd",
        "provider_url": "https?:\/\/www.scribd.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.scribd.com\/doc\/*"
                ],
                "url": "http:\/\/www.scribd.com\/services\/oembed\/"
            }
        ]
    },
    {
        "provider_name": "Dotsub",
        "provider_url": "https?:\/\/dotsub.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/dotsub.com\/view\/*"
                ],
                "url": "http:\/\/dotsub.com\/services\/oembed"
            }
        ]
    },
    {
        "provider_name": "Animoto",
        "provider_url": "https?:\/\/animoto.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/animoto.com\/play\/*"
                ],
                "url": "http:\/\/animoto.com\/oembeds\/create"
            }
        ]
    },
    {
        "provider_name": "Rdio",
        "provider_url": "https?:\/\/rdio.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.rdio.com\/artist\/*",
                    "https?:\/\/*.rdio.com\/people\/*"
                ],
                "url": "http:\/\/www.rdio.com\/api\/oembed\/"
            }
        ]
    },
    {
        "provider_name": "MixCloud",
        "provider_url": "https?:\/\/mixcloud.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.mixcloud.com\/*\/*\/"
                ],
                "url": "http:\/\/www.mixcloud.com\/oembed\/"
            }
        ]
    },
    {
        "provider_name": "Clyp",
        "provider_url": "https?:\/\/clyp.it\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/clyp.it\/*",
                    "https?:\/\/clyp.it\/playlist\/*"
                ],
                "url": "http:\/\/api.clyp.it\/oembed\/",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Screenr",
        "provider_url": "https?:\/\/www.screenr.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.screenr.com\/*\/"
                ],
                "url": "http:\/\/www.screenr.com\/api\/oembed.{format}"
            }
        ]
    },
    {
        "provider_name": "FunnyOrDie",
        "provider_url": "https?:\/\/www.funnyordie.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.funnyordie.com\/videos\/*"
                ],
                "url": "http:\/\/www.funnyordie.com\/oembed.{format}"
            }
        ]
    },
    {
        "provider_name": "Poll Daddy",
        "provider_url": "https?:\/\/polldaddy.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.polldaddy.com\/s\/*",
                    "https?:\/\/*.polldaddy.com\/poll\/*",
                    "https?:\/\/*.polldaddy.com\/ratings\/*"
                ],
                "url": "http:\/\/polldaddy.com\/oembed\/"
            }
        ]
    },
    {
        "provider_name": "Ted",
        "provider_url": "https?:\/\/ted.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/ted.com\/talks\/*"
                ],
                "url": "http:\/\/www.ted.com\/talks\/oembed.{format}"
            }
        ]
    },
    {
        "provider_name": "VideoJug",
        "provider_url": "https?:\/\/www.videojug.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.videojug.com\/film\/*",
                    "https?:\/\/www.videojug.com\/interview\/*"
                ],
                "url": "http:\/\/www.videojug.com\/oembed.{format}"
            }
        ]
    },
    {
        "provider_name": "Sapo Videos",
        "provider_url": "https?:\/\/videos.sapo.pt",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/videos.sapo.pt\/*"
                ],
                "url": "http:\/\/videos.sapo.pt\/oembed"
            }
        ]
    },
    {
        "provider_name": "Official FM",
        "provider_url": "https?:\/\/official.fm",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/official.fm\/tracks\/*",
                    "https?:\/\/official.fm\/playlists\/*"
                ],
                "url": "http:\/\/official.fm\/services\/oembed.{format}"
            }
        ]
    },
    {
        "provider_name": "HuffDuffer",
        "provider_url": "https?:\/\/huffduffer.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/huffduffer.com\/*\/*"
                ],
                "url": "http:\/\/huffduffer.com\/oembed"
            }
        ]
    },
    {
        "provider_name": "Shoudio",
        "provider_url": "https?:\/\/shoudio.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/shoudio.com\/*",
                    "https?:\/\/shoud.io\/*"
                ],
                "url": "http:\/\/shoudio.com\/api\/oembed"
            }
        ]
    },
    {
        "provider_name": "Moby Picture",
        "provider_url": "https?:\/\/www.mobypicture.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.mobypicture.com\/user\/*\/view\/*",
                    "https?:\/\/moby.to\/*"
                ],
                "url": "http:\/\/api.mobypicture.com\/oEmbed"
            }
        ]
    },
    {
        "provider_name": "23HQ",
        "provider_url": "https?:\/\/www.23hq.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.23hq.com\/*\/photo\/*"
                ],
                "url": "http:\/\/www.23hq.com\/23\/oembed"
            }
        ]
    },
    {
        "provider_name": "Cacoo",
        "provider_url": "https:\/\/cacoo.com",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/cacoo.com\/diagrams\/*"
                ],
                "url": "http:\/\/cacoo.com\/oembed.{format}"
            }
        ]
    },
    {
        "provider_name": "Dipity",
        "provider_url": "https?:\/\/www.dipity.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.dipity.com\/*\/*\/"
                ],
                "url": "http:\/\/www.dipity.com\/oembed\/timeline\/"
            }
        ]
    },
    {
        "provider_name": "Roomshare",
        "provider_url": "https?:\/\/roomshare.jp",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/roomshare.jp\/post\/*",
                    "https?:\/\/roomshare.jp\/en\/post\/*"
                ],
                "url": "http:\/\/roomshare.jp\/en\/oembed.{format}"
            }
        ]
    },
	    {
        "provider_name": "Crowd Ranking",
        "provider_url": "https?:\/\/crowdranking.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/crowdranking.com\/*\/*"
                ],
                "url": "http:\/\/crowdranking.com\/api\/oembed.{format}"
            }
        ]
    },
    {
        "provider_name": "CircuitLab",
        "provider_url": "https:\/\/www.circuitlab.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/www.circuitlab.com\/circuit\/*"
                ],
                "url": "https:\/\/www.circuitlab.com\/circuit\/oembed\/",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Geograph Britain and Ireland",
        "provider_url": "https:\/\/www.geograph.org.uk\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.geograph.org.uk\/*",
                    "https?:\/\/*.geograph.co.uk\/*",
                    "https?:\/\/*.geograph.ie\/*",
                    "https?:\/\/*.wikimedia.org\/*_geograph.org.uk_*"
                ],
                "url": "http:\/\/api.geograph.org.uk\/api\/oembed"
            }
        ]
    },
    {
        "provider_name": "Geograph Germany",
        "provider_url": "https?:\/\/geo-en.hlipp.de\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/geo-en.hlipp.de\/*",
                    "https?:\/\/geo.hlipp.de\/*",
                    "https?:\/\/germany.geograph.org\/*"
                ],
                "url": "http:\/\/geo.hlipp.de\/restapi.php\/api\/oembed"
            }
        ]
    },
    {
        "provider_name": "Geograph Channel Islands",
        "provider_url": "https?:\/\/channel-islands.geograph.org\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.geograph.org.gg\/*",
                    "https?:\/\/*.geograph.org.je\/*",
                    "https?:\/\/channel-islands.geograph.org\/*",
                    "https?:\/\/channel-islands.geographs.org\/*",
                    "https?:\/\/*.channel.geographs.org\/*"
                ],
                "url": "http:\/\/www.geograph.org.gg\/api\/oembed"
            }
        ]
    },
    {
        "provider_name": "Quiz.biz",
        "provider_url": "https?:\/\/www.quiz.biz\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.quiz.biz\/quizz-*.html"
                ],
                "url": "http:\/\/www.quiz.biz\/api\/oembed",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Quizz.biz",
        "provider_url": "https?:\/\/www.quizz.biz\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/www.quizz.biz\/quizz-*.html"
                ],
                "url": "http:\/\/www.quizz.biz\/api\/oembed",
                "discovery": true
            }
        ]
    },
    {
        "provider_name": "Coub",
        "provider_url": "https?:\/\/coub.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/coub.com\/view\/*",
                    "https?:\/\/coub.com\/embed\/*"
                ],
                "url": "http:\/\/coub.com\/api\/oembed.{format}"
            }
        ]
    },
    {
        "provider_name": "SpeakerDeck",
        "provider_url": "https:\/\/speakerdeck.com",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/speakerdeck.com\/*\/*",
                    "https:\/\/speakerdeck.com\/*\/*"
                ],
                "url": "https:\/\/speakerdeck.com\/oembed.json",
                "discovery": true,
                "formats": [
                    "json"
                ]
            }
        ]
    },
    {
        "provider_name": "Alpha App Net",
        "provider_url": "https:\/\/alpha.app.net\/browse\/posts\/",
        "endpoints": [
            {
                "schemes": [
                    "https:\/\/alpha.app.net\/*\/post\/*",
                    "https:\/\/photos.app.net\/*\/*"
                ],
                "url": "https:\/\/alpha-api.app.net\/oembed",
                "formats": [
                    "json"
                ]
            }
        ]
    },
    {
        "provider_name": "YFrog",
        "provider_url": "https?:\/\/yfrog.com\/",
        "endpoints": [
            {
                "schemes": [
                    "https?:\/\/*.yfrog.com\/*",
                    "https?:\/\/yfrog.us\/*"
                ],
                "url": "http:\/\/www.yfrog.com\/api\/oembed",
                "formats": [
                    "json"
                ]
            }
        ]
    },
*/
