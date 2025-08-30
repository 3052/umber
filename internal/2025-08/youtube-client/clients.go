package main

const visitor_id = "CgtNbzlJR19GY24tNCjl_pDABjIKCgJVUxIEGgAgDA=="

// youtube.com/watch?v=QmpHcrA2hC0
const video_id_kids = "QmpHcrA2hC0"

// youtube.com/watch?v=fD0qZRK1lQ0
const video_id_vr = "fD0qZRK1lQ0"

type ClientVersion struct {
   id       int
   name     string
   status   string
   version  string
   video_id string
}

const (
   no_longer_supported = "YouTube is no longer supported in this application or device."
   ok                  = "OK"
   sign_in             = "Please sign in"
)

var clients = []ClientVersion{
   {
      id:       1,
      name:     "WEB",
      version:  "2.20250829.01.00",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       2,
      name:     "MWEB",
      version:  "2.20250829.01.00",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       3,
      name:     "ANDROID",
      version:  "20.34.37",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       5,
      name:     "IOS",
      version:  "20.34.2",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       7,
      name:     "TVHTML5",
      version:  "7.20241201.18.00",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       8,
      name:     "TVLITE",
      status:   no_longer_supported,
   },
   {
      id:       10,
      name:     "TVANDROID",
      version:  "1.0",
      video_id: video_id_kids,
   },
   {
      id:       13,
      name:     "XBOXONEGUIDE",
      version:  "1.0",
      video_id: video_id_kids,
   },
   {
      id:       14,
      name:     "ANDROID_CREATOR",
      version:  "22.36.102",
      video_id: video_id_kids,
   },
   {
      id:       15,
      name:     "IOS_CREATOR",
      version:  "22.36.102",
      video_id: video_id_kids,
   },
   {
      id:       16,
      name:     "TVAPPLE",
      status:   no_longer_supported,
   },
   {
      id:       18,
      name:     "ANDROID_KIDS",
      version:  "7.36.1",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       19,
      name:     "IOS_KIDS",
      version:  "7.36.1",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       21,
      name:     "ANDROID_MUSIC",
      status:   sign_in,
   },
   {
      id:       23,
      name:     "ANDROID_TV",
      version:  "6.18.303",
      video_id: video_id_kids,
   },
   {
      id:       26,
      name:     "IOS_MUSIC",
      status:   sign_in,
   },
   {
      id:       27,
      name:     "MWEB_TIER_2",
      version:  "9.20220918",
      video_id: video_id_kids,
   },
   {
      id:       28,
      name:     "ANDROID_VR",
      version:  "1.37",
      video_id: video_id_kids,
   },
   {
      id:       29,
      name:     "ANDROID_UNPLUGGED",
      status:   sign_in,
   },
   {
      id:       30,
      name:     "ANDROID_TESTSUITE",
      version:  "1.9",
      video_id: video_id_kids,
   },
   {
      id:       31,
      name:     "WEB_MUSIC_ANALYTICS",
      version:  "0.2",
      video_id: video_id_kids,
   },
   {
      id:       33,
      name:     "IOS_UNPLUGGED",
      status:   sign_in,
   },
   {
      id:       38,
      name:     "ANDROID_LITE",
      status:   no_longer_supported,
   },
   {
      id:       39,
      name:     "IOS_EMBEDDED_PLAYER",
      version:  "2.4",
      video_id: video_id_kids,
   },
   {
      id:       41,
      name:     "WEB_UNPLUGGED",
      status:   sign_in,
   },
   {
      id:       42,
      name:     "WEB_EXPERIMENTS",
      version:  "1",
      video_id: video_id_kids,
   },
   {
      id:       43,
      name:     "TVHTML5_CAST",
      version:  "1.1",
      video_id: video_id_kids,
   },
   {
      id:       55,
      name:     "ANDROID_EMBEDDED_PLAYER",
      version:  "17.36.4",
      video_id: video_id_kids,
   },
   {
      id:       56,
      name:     "WEB_EMBEDDED_PLAYER",
      version:  "9.20220918",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       57,
      name:     "TVHTML5_AUDIO",
      status:   sign_in,
   },
   {
      id:       58,
      name:     "TV_UNPLUGGED_CAST",
      version:  "0.1",
      video_id: video_id_kids,
   },
   {
      id:       59,
      name:     "TVHTML5_KIDS",
      version:  "3.20220918",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       60,
      name:     "WEB_HEROES",
      version:  "0.1",
      video_id: video_id_kids,
   },
   {
      id:       61,
      name:     "WEB_MUSIC",
      status:   no_longer_supported,
   },
   {
      id:       62,
      name:     "WEB_CREATOR",
      status:   sign_in,
   },
   {
      id:       63,
      name:     "TV_UNPLUGGED_ANDROID",
      version:  "1.37",
      video_id: video_id_kids,
   },
   {
      id:       64,
      name:     "IOS_LIVE_CREATION_EXTENSION",
      version:  "17.36.4",
      video_id: video_id_kids,
   },
   {
      id:       65,
      name:     "TVHTML5_UNPLUGGED",
      status:   sign_in,
   },
   {
      id:       66,
      name:     "IOS_MESSAGES_EXTENSION",
      version:  "17.36.4",
      video_id: video_id_kids,
   },
   {
      id:       67,
      name:     "WEB_REMIX",
      version:  "1.20220918",
      video_id: video_id_kids,
   },
   {
      id:       68,
      name:     "IOS_UPTIME",
      version:  "1.0",
      video_id: video_id_kids,
   },
   {
      id:       69,
      name:     "WEB_UNPLUGGED_ONBOARDING",
      status:   no_longer_supported,
   },
   {
      id:       70,
      name:     "WEB_UNPLUGGED_OPS",
      status:   sign_in,
   },
   {
      id:       71,
      name:     "WEB_UNPLUGGED_PUBLIC",
      status:   no_longer_supported,
   },
   {
      id:       72,
      name:     "TVHTML5_VR",
      version:  "0.1",
      video_id: video_id_kids,
   },
   {
      id:       73,
      name:     "WEB_LIVE_STREAMING",
      version:  "2.20250829.01.00",
      video_id: video_id_kids,
   },
   {
      id:       74,
      name:     "ANDROID_TV_KIDS",
      version:  "1.19.1",
      video_id: video_id_kids,
   },
   {
      id:       75,
      name:     "TVHTML5_SIMPLY",
      version:  "1.0",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       76,
      name:     "WEB_KIDS",
      version:  "2.20250829.01.00",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       77,
      name:     "MUSIC_INTEGRATIONS",
      version:  "0.1",
      video_id: video_id_kids,
   },
   {
      id:       80,
      name:     "TVHTML5_YONGLE",
      status:   no_longer_supported,
   },
   {
      id:       84,
      name:     "GOOGLE_ASSISTANT",
      version:  "0.1",
      video_id: video_id_kids,
   },
   {
      id:       85,
      name:     "TVHTML5_SIMPLY_EMBEDDED_PLAYER",
      status:   sign_in,
   },
   {
      id:       86,
      name:     "WEB_MUSIC_EMBEDDED_PLAYER",
      version:  "2.20250829.01.00",
      video_id: video_id_kids,
   },
   {
      id:       87,
      name:     "WEB_INTERNAL_ANALYTICS",
      version:  "0.1",
      video_id: video_id_kids,
   },
   {
      id:       88,
      name:     "WEB_PARENT_TOOLS",
      version:  "1.20220918",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       89,
      name:     "GOOGLE_MEDIA_ACTIONS",
      version:  "0.1",
      video_id: video_id_kids,
   },
   {
      id:       90,
      name:     "WEB_PHONE_VERIFICATION",
      version:  "1.0.0",
      video_id: video_id_kids,
   },
   {
      id:       93,
      name:     "TVHTML5_FOR_KIDS",
      version:  "7.20220918",
      status:   ok,
      video_id: video_id_kids,
   },
   {
      id:       94,
      name:     "GOOGLE_LIST_RECS",
      version:  "0.1",
      video_id: video_id_kids,
   },
   {
      id:       95,
      name:     "MEDIA_CONNECT_FRONTEND",
      version:  "0.1",
      video_id: video_id_kids,
   },
   {
      id:       98,
      name:     "WEB_EFFECT_MAKER",
      version:  "2.20250829.01.00",
      video_id: video_id_kids,
   },
   {
      id:       99,
      name:     "WEB_SHOPPING_EXTENSION",
      version:  "2.20250829.01.00",
      video_id: video_id_kids,
   },
   {
      id:       100,
      name:     "WEB_PLAYABLES_PORTAL",
      version:  "2.20250829.01.00",
      video_id: video_id_kids,
   },
   {
      id:       102,
      name:     "WEB_LIVE_APPS",
      version:  "2.20250829.01.00",
      video_id: video_id_kids,
   },
   {
      id:       103,
      name:     "WEB_MUSIC_INTEGRATIONS",
      version:  "2.20250829.01.00",
      video_id: video_id_kids,
   },
}
