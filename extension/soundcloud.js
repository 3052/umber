'use strict';

const client_id = 'iZIs9mchVcX5lhVRyQGGAYlNPVldzAoX';

async function soundcloud_media(track) {
   const param = new URLSearchParams({client_id: client_id});
   for (const code of track[0].media.transcodings) {
      if (code.format.protocol != 'progressive') {
         continue;
      }
      const media = new URL(code.url);
      media.search = String(param);
      const resp = await fetch(media);
      return resp.json();
   }
   return {url: ''};
}

async function soundcloud_track(id) {
   const param = new URLSearchParams({client_id: client_id, ids: id});
   const track = new URL('https://api-v2.soundcloud.com/tracks');
   track.search = String(param);
   const resp = await fetch(track);
   return resp.json();
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
