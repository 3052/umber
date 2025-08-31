'use strict';

async function youTube() {
   const ref = new URL(this.href);
   const req = {};
   const body = {};
   body.videoId = ref.searchParams.get('v');
   body.context = {};
   body.context.client = {};
   req.headers = {};
   //////////////////////////////////////////////////////////////////////////////
   req.headers['X-Goog-Visitor-Id'] = 'CgtNbzlJR19GY24tNCjl_pDABjIKCgJVUxIEGgAgDA==';
   body.context.client.clientName = 'ANDROID';
   body.context.client.clientVersion = '20.34.37';
   //////////////////////////////////////////////////////////////////////////////
   req.body = JSON.stringify(body);
   req.method = 'POST';
   const resp = await fetch('https://www.youtube.com/youtubei/v1/player', req);
   const play = await resp.json();
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
