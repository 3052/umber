'use strict';

/* --- BANDCAMP --- */
async function bandcamp() {
   let ref = new URL('https://bandcamp.com/api/mobile/24/tralbum_details');
   // bandcamp.com/EmbeddedPlayer/track=4023025438
   let ind = this.href.indexOf('=');
   let param = new URLSearchParams({
      band_id: 1,
      tralbum_id: this.href.slice(ind + 1),
      tralbum_type: 't'
   });
   ref.search = String(param);
   let resp = await fetch(ref);
   let media = await resp.json();
   browser.runtime.sendMessage({
      poster: this.querySelector('img').src,
      src: media.tracks[0].streaming_url['mp3-128'],
      title: this.parentNode.querySelector('td').textContent
   });
}

/* --- SOUNDCLOUD --- */
const client_id = 'KKzJxmw11tYpCs6T24P4uUYhqmjalG6M';

async function soundcloud_track(id) {
   const track = new URL('https://api-v2.soundcloud.com/tracks/' + id);
   const param = new URLSearchParams({client_id: client_id});
   track.search = String(param);
   const resp = await fetch(track);
   return resp.json();
}

async function soundcloud_media(track) {
   for (const code of track.media.transcodings) {
      if (code.format.protocol == 'progressive') {
         const media = new URL(code.url);
         const param = new URLSearchParams({client_id: client_id});
         media.search = String(param);
         const resp = await fetch(media);
         return resp.json();
      }
   }
   return {url: ''};
}

async function soundCloud() {
   const url = new URL(this.href);
   const id = url.searchParams.get('url').split('/').slice(-1);
   const track = await soundcloud_track(id);
   const media = await soundcloud_media(track);
   browser.runtime.sendMessage({
      src: media.url,
      poster: this.querySelector('img').src,
      title: this.parentNode.querySelector('td').textContent
   });
}

/* --- YOUTUBE --- */
async function youTube() {
   const ref = new URL(this.href);
   const req = {};
   const body = {};
   body.videoId = ref.searchParams.get('v');
   body.context = {};
   body.context.client = {};
   req.headers = {};
   //////////////////////////////////////////////////////////////////////////////
   req.headers['X-Goog-Visitor-Id'] = 'Cgt5TEMyT0p5NzNzVSjDou_LBjIKCgJVUxIEGgAgGA==';
   body.context.client.clientName = 'ANDROID_VR';
   body.context.client.clientVersion = '1.71.26';
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

/* --- DELAY / MAIN --- */
const time = 99;
const count = 99;

function delay(callback, time, count) {
   const id = setInterval(function() {
      const ok = callback();
      count--;
      if (ok || count == 0) {
         clearInterval(id);
      }
   }, time);
}

delay(function() {
   const as = document.getElementsByTagName('a');
   if (as.length == 0) {
      return false;
   }
   for (const a of as) {
      // We already have an event for localStorage, so use addEventListener so
      // we dont clobber that.
      switch (true) {
      case a.host == 'bandcamp.com':
         a.addEventListener('contextmenu', bandcamp);
         break;
      case a.host == 'w.soundcloud.com':
         a.addEventListener('contextmenu', soundCloud);
         break;
      case a.host == 'www.youtube.com':
         a.addEventListener('contextmenu', youTube);
         break;
      default:
         continue;
      }
      a.style.borderTop = '2px solid red';
   }
   return true;
}, time, count);
