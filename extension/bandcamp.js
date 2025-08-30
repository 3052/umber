'use strict';

async function bandcamp() {
   let ref = new URL('http://bandcamp.com/api/mobile/24/tralbum_details');
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
