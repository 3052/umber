// platform.js marker preserve
'use strict';

export function bandcamp(row) {
   return {
      href: 'https://bandcamp.com/EmbeddedPlayer/track=' + row.I,
      src: row.A
   };
}

export function http(row) {
   return {
      href: row.I,
      src: row.A
   };
}

const formatter = new Intl.DateTimeFormat('en', {
   weekday: 'short', 
   month: 'short', 
   day: 'numeric',
   year: 'numeric'
});

export function date(timestamp) {
   const time = new Date(timestamp * 1000);
   
   return formatter.formatToParts(time)
      .filter(part => part.type !== 'literal')
      .map(part => part.value)
      .join(' ');
}

export function soundcloud(row) {
   const params = new URLSearchParams();
   params.set('url', 'https://api.soundcloud.com/tracks/' + row.I);
   
   return {
      href: 'https://w.soundcloud.com/player?' + params.toString(),
      src: 'https://i1.sndcdn.com/' + row.A
   };
}

export function youtube(row) {
   const ytimg = 'https://i.ytimg.com/vi_webp/' + row.I + '/sddefault.webp';
   const image = row.A !== undefined ? row.A : ytimg;
   
   return {
      href: 'https://www.youtube.com/watch?v=' + row.I,
      src: image
   };
}

// platform.js marker preserve
