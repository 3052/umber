'use strict';

export function http(row) {
   return {
      href: row.I,
      src: row.A
   };
}

const formatter = new Intl.DateTimeFormat('en', {
   weekday: 'short', month: 'short', day: 'numeric', year: 'numeric'
});

export function date(timestamp) {
   const time = new Date(timestamp * 1000);
   
   return formatter.formatToParts(time)
      .filter(part => part.type !== 'literal')
      .map(part => part.value)
      .join(' ');
}

export function bandcamp(row) {
   return {
      href: 'https://bandcamp.com/EmbeddedPlayer/track=' + row.I,
      src: 'https://f4.bcbits.com/img/a' + row.A + '_2'
   };
}

export function soundcloud(row) {
   const params = new URLSearchParams();
   params.set('url', 'api.soundcloud.com/tracks/' + row.I);
   
   return {
      href: 'https://w.soundcloud.com/player?' + params.toString(),
      src: 'https://i1.sndcdn.com/' + row.A
   };
}

export function youtube(row) {
   const image = 'A' in row ? row.A : 'sddefault.webp';
   const path = row.I + '/' + image;
   
   return {
      href: 'https://www.youtube.com/watch?v=' + row.I,
      src: image.endsWith('.webp') ? 'https://i.ytimg.com/vi_webp/' + path : 'https://i.ytimg.com/vi/' + path
   };
}
