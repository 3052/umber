'use strict';

export function http(row) {
   return {
      href: row.B,
      src: row.C
   };
}

const formatter = new Intl.DateTimeFormat('en', {
   weekday: 'short', month: 'short', day: 'numeric', year: 'numeric'
});

export function date(timestamp) {
   // Timestamp is now natively a base10 Number
   const time = new Date(timestamp * 1000);
   
   return formatter.formatToParts(time)
      .filter(part => part.type !== 'literal')
      .map(part => part.value)
      .join(' ');
}

export function bandcamp(row) {
   return {
      href: 'https://bandcamp.com/EmbeddedPlayer/track=' + row.B,
      src: 'https://f4.bcbits.com/img/a' + row.C + '_2'
   };
}

export function soundcloud(row) {
   const params = new URLSearchParams();
   params.set('url', 'api.soundcloud.com/tracks/' + row.B);
   
   return {
      href: 'https://w.soundcloud.com/player?' + params.toString(),
      src: 'https://i1.sndcdn.com/' + row.C
   };
}

export function youtube(row) {
   const image = 'C' in row ? row.C : 'sddefault.webp';
   const path = row.B + '/' + image;
   
   return {
      href: 'https://www.youtube.com/watch?v=' + row.B,
      src: image.endsWith('.webp') ? 'https://i.ytimg.com/vi_webp/' + path : 'https://i.ytimg.com/vi/' + path
   };
}
