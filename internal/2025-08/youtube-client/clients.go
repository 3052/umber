package main

const visitor_id = "CgtNbzlJR19GY24tNCjl_pDABjIKCgJVUxIEGgAgDA=="

// kids
// youtube.com/watch?v=QmpHcrA2hC0
const video_id = "QmpHcrA2hC0"

type ClientVersion struct {
   ID      int    `json:"id"`
   Name    string `json:"name"`
   Version string `json:"version"`
   Status  string `json:"status"`
}

const no_longer_supported = "ERROR YouTube is no longer supported in this application or device."

var clients = []ClientVersion{
   {
      ID:      1,
      Name:    "WEB",
      Version: "2.20250829.01.00",
   },
   {
      ID:      1,
      Name:    "WEB",
      Version: "2.20220918",
   },
   {
      ID:      2,
      Name:    "MWEB",
      Version: "2.20250829.01.00",
   },
   {
      ID:      2,
      Name:    "MWEB",
      Version: "2.20220918",
   },
   {
      ID:      3,
      Name:    "ANDROID",
      Version: "20.34.37",
   },
   {
      ID:      3,
      Name:    "ANDROID",
      Version: "17.36.4",
   },
   {
      ID:      5,
      Name:    "IOS",
      Version: "20.34.2",
   },
   {
      ID:      5,
      Name:    "IOS",
      Version: "17.36.4",
   },
   {
      ID:      7,
      Name:    "TVHTML5",
      Version: "7.20241201.18.00",
   },
   {
      ID:      7,
      Name:    "TVHTML5",
      Version: "7.20220918",
   },
   {
      ID:     8,
      Name:   "TVLITE",
      Status: no_longer_supported,
   },
   {
      ID:      10,
      Name:    "TVANDROID",
      Version: "1.0",
   },
   {
      ID:      13,
      Name:    "XBOXONEGUIDE",
      Version: "1.0",
   },
   {
      ID:      14,
      Name:    "ANDROID_CREATOR",
      Version: "20.34.100",
   },
   {
      ID:      14,
      Name:    "ANDROID_CREATOR",
      Version: "22.36.102",
   },
   {
      ID:      15,
      Name:    "IOS_CREATOR",
      Version: "20.34.x",
   },
   {
      ID:      15,
      Name:    "IOS_CREATOR",
      Version: "22.36.102",
   },
   {
      ID:      16,
      Name:    "TVAPPLE",
      Version: "1.0",
      Status:  no_longer_supported,
   },
   {
      ID:      18,
      Name:    "ANDROID_KIDS",
      Version: "7.36.1",
   },
   {
      ID:      19,
      Name:    "IOS_KIDS",
      Version: "7.36.1",
   },
   {
      ID:      21,
      Name:    "ANDROID_MUSIC",
      Version: "8.34.51",
   },
   {
      ID:      21,
      Name:    "ANDROID_MUSIC",
      Version: "5.26.1",
   },
   {
      ID:      23,
      Name:    "ANDROID_TV",
      Version: "6.18.303",
   },
   {
      ID:      23,
      Name:    "ANDROID_TV",
      Version: "2.19.1.303051424",
   },
   {
      ID:      26,
      Name:    "IOS_MUSIC",
      Version: "08.34",
   },
   {
      ID:      26,
      Name:    "IOS_MUSIC",
      Version: "5.26.1",
   },
   {
      ID:      27,
      Name:    "MWEB_TIER_2",
      Version: "2.20250829.01.00",
   },
   {
      ID:      27,
      Name:    "MWEB_TIER_2",
      Version: "9.20220918",
   },
   {
      ID:      28,
      Name:    "ANDROID_VR",
      Version: "1.37",
   },
   {
      ID:      29,
      Name:    "ANDROID_UNPLUGGED",
      Version: "6.36",
   },
   {
      ID:      30,
      Name:    "ANDROID_TESTSUITE",
      Version: "1.9",
   },
   {
      ID:      31,
      Name:    "WEB_MUSIC_ANALYTICS",
      Version: "0.2",
   },
   {
      ID:      33,
      Name:    "IOS_UNPLUGGED",
      Version: "6.36",
   },
   {
      ID:      38,
      Name:    "ANDROID_LITE",
      Version: "3.26.1",
      Status:  no_longer_supported,
   },
   {
      ID:      39,
      Name:    "IOS_EMBEDDED_PLAYER",
      Version: "2.4",
   },
   {
      ID:      41,
      Name:    "WEB_UNPLUGGED",
      Version: "2.20250829.01.00",
   },
   {
      ID:      41,
      Name:    "WEB_UNPLUGGED",
      Version: "1.20220918",
   },
   {
      ID:      42,
      Name:    "WEB_EXPERIMENTS",
      Version: "2.20250829.01.00",
   },
   {
      ID:      42,
      Name:    "WEB_EXPERIMENTS",
      Version: "1",
   },
   {
      ID:      43,
      Name:    "TVHTML5_CAST",
      Version: "1.1",
   },
   {
      ID:      55,
      Name:    "ANDROID_EMBEDDED_PLAYER",
      Version: "17.36.4",
   },
   {
      ID:      56,
      Name:    "WEB_EMBEDDED_PLAYER",
      Version: "2.20250829.01.00",
   },
   {
      ID:      56,
      Name:    "WEB_EMBEDDED_PLAYER",
      Version: "9.20220918",
   },
   {
      ID:      57,
      Name:    "TVHTML5_AUDIO",
      Version: "2.0",
   },
   {
      ID:      58,
      Name:    "TV_UNPLUGGED_CAST",
      Version: "0.1",
   },
   {
      ID:      59,
      Name:    "TVHTML5_KIDS",
      Version: "3.20220918",
   },
   {
      ID:      60,
      Name:    "WEB_HEROES",
      Version: "2.20250829.01.00",
   },
   {
      ID:      60,
      Name:    "WEB_HEROES",
      Version: "0.1",
   },
   {
      ID:     61,
      Name:   "WEB_MUSIC",
      Status: no_longer_supported,
   },
   {
      ID:      62,
      Name:    "WEB_CREATOR",
      Version: "2.20250829.01.00",
   },
   {
      ID:      62,
      Name:    "WEB_CREATOR",
      Version: "1.20220918",
   },
   {
      ID:      63,
      Name:    "TV_UNPLUGGED_ANDROID",
      Version: "1.37",
   },
   {
      ID:      64,
      Name:    "IOS_LIVE_CREATION_EXTENSION",
      Version: "17.36.4",
   },
   {
      ID:      65,
      Name:    "TVHTML5_UNPLUGGED",
      Version: "6.36",
   },
   {
      ID:      66,
      Name:    "IOS_MESSAGES_EXTENSION",
      Version: "17.36.4",
   },
   {
      ID:      67,
      Name:    "WEB_REMIX",
      Version: "2.20250829.01.00",
   },
   {
      ID:      67,
      Name:    "WEB_REMIX",
      Version: "1.20220918",
   },
   {
      ID:      68,
      Name:    "IOS_UPTIME",
      Version: "1.0",
   },
   {
      ID:     69,
      Name:   "WEB_UNPLUGGED_ONBOARDING",
      Status: no_longer_supported,
   },
   {
      ID:      70,
      Name:    "WEB_UNPLUGGED_OPS",
      Version: "2.20250829.01.00",
   },
   {
      ID:      70,
      Name:    "WEB_UNPLUGGED_OPS",
      Version: "0.1",
   },
   {
      ID:     71,
      Name:   "WEB_UNPLUGGED_PUBLIC",
      Status: no_longer_supported,
   },
   {
      ID:      72,
      Name:    "TVHTML5_VR",
      Version: "0.1",
   },
   {
      ID:      73,
      Name:    "WEB_LIVE_STREAMING",
      Version: "2.20250829.01.00",
   },
   {
      ID:      74,
      Name:    "ANDROID_TV_KIDS",
      Version: "1.19.1",
   },
   {
      ID:      75,
      Name:    "TVHTML5_SIMPLY",
      Version: "1.0",
   },
   {
      ID:      76,
      Name:    "WEB_KIDS",
      Version: "2.20250829.01.00",
   },
   {
      ID:      76,
      Name:    "WEB_KIDS",
      Version: "2.20220918",
   },
   {
      ID:      77,
      Name:    "MUSIC_INTEGRATIONS",
      Version: "0.1",
   },
   {
      ID:     80,
      Name:   "TVHTML5_YONGLE",
      Status: no_longer_supported,
   },
   {
      ID:      84,
      Name:    "GOOGLE_ASSISTANT",
      Version: "0.1",
   },
   {
      ID:      85,
      Name:    "TVHTML5_SIMPLY_EMBEDDED_PLAYER",
      Version: "2.0",
   },
   {
      ID:      86,
      Name:    "WEB_MUSIC_EMBEDDED_PLAYER",
      Version: "2.20250829.01.00",
   },
   {
      ID:      87,
      Name:    "WEB_INTERNAL_ANALYTICS",
      Version: "2.20250829.01.00",
   },
   {
      ID:      87,
      Name:    "WEB_INTERNAL_ANALYTICS",
      Version: "0.1",
   },
   {
      ID:      88,
      Name:    "WEB_PARENT_TOOLS",
      Version: "2.20250829.01.00",
   },
   {
      ID:      88,
      Name:    "WEB_PARENT_TOOLS",
      Version: "1.20220918",
   },
   {
      ID:      89,
      Name:    "GOOGLE_MEDIA_ACTIONS",
      Version: "0.1",
   },
   {
      ID:      90,
      Name:    "WEB_PHONE_VERIFICATION",
      Version: "2.20250829.01.00",
   },
   {
      ID:      90,
      Name:    "WEB_PHONE_VERIFICATION",
      Version: "1.0.0",
   },
   {
      ID:      93,
      Name:    "TVHTML5_FOR_KIDS",
      Version: "7.20220918",
   },
   {
      ID:      94,
      Name:    "GOOGLE_LIST_RECS",
      Version: "0.1",
   },
   {
      ID:      95,
      Name:    "MEDIA_CONNECT_FRONTEND",
      Version: "0.1",
   },
   {
      ID:      98,
      Name:    "WEB_EFFECT_MAKER",
      Version: "2.20250829.01.00",
   },
   {
      ID:      99,
      Name:    "WEB_SHOPPING_EXTENSION",
      Version: "2.20250829.01.00",
   },
   {
      ID:      100,
      Name:    "WEB_PLAYABLES_PORTAL",
      Version: "2.20250829.01.00",
   },
   {
      ID:      102,
      Name:    "WEB_LIVE_APPS",
      Version: "2.20250829.01.00",
   },
   {
      ID:      103,
      Name:    "WEB_MUSIC_INTEGRATIONS",
      Version: "2.20250829.01.00",
   },
}
