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
   body.context.client.clientName = 'IOS';
   body.context.client.clientVersion = '20.03.02';
   // data := base64.RawStdEncoding.EncodeToString([]byte("########"))
   // var message protobuf.Message
   // message.AddBytes(1, []byte(data))
   // return base64.RawStdEncoding.EncodeToString(message.Marshal())
   req.headers['X-Goog-Visitor-Id'] = 'CgtJeU1qSXlNakl5TQ';
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
