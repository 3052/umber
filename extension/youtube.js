'use strict';

async function youTube() {
   const ref = new URL(this.href);
   const body = {
      context: {
         client: {
            androidSdkVersion: 99,
            clientName: 'ANDROID',
            clientVersion: '18.19.99'
         }
      },
      videoId: ref.searchParams.get('v')
   };
   const req = {
      body: JSON.stringify(body),
      headers: {'User-Agent': 'com.google.android.youtube/18.99.99'},
      method: 'POST'
   };
   const res = await fetch('https://www.youtube.com/youtubei/v1/player', req);
   const play = await res.json();
   const msg = {
      poster: this.querySelector('img').src,
      title: play.videoDetails.author + ' - ' + play.videoDetails.title
   };
   if (play.playabilityStatus.status == 'OK') {
      play.streamingData.adaptiveFormats.sort(
         (a, b) => b.bitrate - a.bitrate
      );
      for (const form of play.streamingData.adaptiveFormats) {
         // some videos do not offer WebM: 6_lMeEMMbyY
         if (form.audioQuality == 'AUDIO_QUALITY_MEDIUM') {
            msg.src = form.url;
            break;
         }
      }
   } else {
      msg.src = '';
      msg.title += ' - ' + play.playabilityStatus.reason;
   }
   browser.runtime.sendMessage(msg);
}
