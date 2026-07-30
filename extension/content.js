'use strict';

/* --- YOUTUBE --- */
async function youtube() {
   const figure = this.closest('figure');
   const link = figure.querySelector('a');
   const parsed = new URL(link.href);
   const request = {};
   const payload = {};
   
   payload.videoId = parsed.searchParams.get('v');
   payload.context = {};
   payload.context.client = {};
   request.headers = {};
   
   //////////////////////////////////////////////////////////////////////////////
   //request.headers['X-Goog-Visitor-Id'] = 'Cgt5TEMyT0p5NzNzVSjDou_LBjIKCgJVUxIEGgAgGA==';
   
   request.headers['X-Goog-Visitor-Id'] = 'CgtpRmlLOFdEaHZVRSji5a_TBjIKCgJVUxIEGgAgHmLfAgrcAjIwLllUPWo1MnJ3V2t2YU5iMXotRHNSVUF4WHVlYjhIUXg1V3Qxdkg4Q1NRTWQ0cWdPRHdPM2x1eVlkdGgzQzVxYmljZDI3Y3lyNUdOV2pxZTQ0X3RCNkhIdkhUWmM0b2RRMnNLSTlHaWpGWHRyX2ptdmFxckVVQzR1cHJrX2ZqRDh2c01NQ0ltblpTbGtwVndHdEMtTk5jTi10OUZub1N0RnctNTdJbUZ3UTNua3dYbDhlRG9yUEUwMlVQNjVZdGc3cGwtRjhKM3ZjVGhEWVZrME5CUEZtQWg5dmpUbmtHd054ZG1pNHA0ZWxTSkNHODgzMWQwSFpYT1NKSkpDM2lmaXFYVllYRk5fWUJXRW1KSlhuVFBCS0xwMGxiRDVHUnNnYXA4Q3FjTmZUbUlVVWdTbW94T2FZV21ocW1KRjZPdk91Q0EyelJkdFhLOHFhREhBa0twd1Q5aUhBUQ==';
   payload.context.client.clientName = 'ANDROID_VR';
   payload.context.client.clientVersion = '1.65.10';
   //////////////////////////////////////////////////////////////////////////////
   
   request.body = JSON.stringify(payload);
   request.method = 'POST';
   
   const response = await fetch('https://www.youtube.com/youtubei/v1/player', request);
   const player = await response.json();
   const message = {
      poster: link.querySelector('img').src,
      title: player.videoDetails.author + ' - ' + player.videoDetails.title
   };
   
   if (player.playabilityStatus.status == 'OK') {
      player.streamingData.adaptiveFormats.sort(
         (a, b) => b.bitrate - a.bitrate
      );
      for (const format of player.streamingData.adaptiveFormats) {
         // some videos do not offer WebM: 6_lMeEMMbyY
         if (format.audioQuality == 'AUDIO_QUALITY_MEDIUM') {
            message.src = format.url;
            break;
         }
      }
   } else {
      message.src = '';
      message.title += ' - ' + player.playabilityStatus.reason;
   }
   browser.runtime.sendMessage(message);
}

/* --- DELAY / MAIN --- */
const interval = 99;
let retries = 99;

function delay(callback, interval, retries) {
   const timer = setInterval(function() {
      const ready = callback();
      retries--;
      if (ready || retries == 0) {
         clearInterval(timer);
      }
   }, interval);
}

delay(function() {
   const upvotes = document.querySelectorAll('.up');
   if (upvotes.length == 0) {
      return false;
   }
   for (const upvote of upvotes) {
      const link = upvote.closest('figure').querySelector('a');
      switch (true) {
      case link.host == 'bandcamp.com':
         upvote.addEventListener('click', bandcamp);
         break;
      case link.host == 'w.soundcloud.com':
         upvote.addEventListener('click', soundcloud);
         break;
      case link.host == 'www.youtube.com':
         upvote.addEventListener('click', youtube);
         break;
      default:
         continue;
      }
   }
   return true;
}, interval, retries);
/* --- BANDCAMP --- */
async function bandcamp() {
   const figure = this.closest('figure');
   const link = figure.querySelector('a');
   const endpoint = new URL('https://bandcamp.com/api/mobile/24/tralbum_details');
   const index = link.href.indexOf('=');
   const query = new URLSearchParams({
      band_id: 1,
      tralbum_id: link.href.slice(index + 1),
      tralbum_type: 't'
   });
   endpoint.search = String(query);
   const response = await fetch(endpoint);
   const data = await response.json();
   
   browser.runtime.sendMessage({
      poster: link.querySelector('img').src,
      src: data.tracks[0].streaming_url['mp3-128'],
      title: figure.querySelector('thead td').textContent
   });
}

/* --- SOUNDCLOUD --- */
const token = 'KKzJxmw11tYpCs6T24P4uUYhqmjalG6M';

async function getTrack(id) {
   const endpoint = new URL('https://api-v2.soundcloud.com/tracks/' + id);
   const query = new URLSearchParams({ client_id: token });
   endpoint.search = String(query);
   const response = await fetch(endpoint);
   return response.json();
}

async function getMedia(track) {
   for (const format of track.media.transcodings) {
      if (format.format.protocol == 'progressive') {
         const endpoint = new URL(format.url);
         const query = new URLSearchParams({ client_id: token });
         endpoint.search = String(query);
         const response = await fetch(endpoint);
         return response.json();
      }
   }
   return { url: '' };
}

async function soundcloud() {
   const figure = this.closest('figure');
   const link = figure.querySelector('a');
   const parsed = new URL(link.href);
   const id = parsed.searchParams.get('url').split('/').slice(-1);
   const track = await getTrack(id);
   const media = await getMedia(track);
   
   browser.runtime.sendMessage({
      src: media.url,
      poster: link.querySelector('img').src,
      title: figure.querySelector('thead td').textContent
   });
}

