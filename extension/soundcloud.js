'use strict';

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
