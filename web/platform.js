'use strict';

export function http(query) {
   return {
      href: query.get('b'),
      src: query.get('c')
   };
}

const formatter = new Intl.DateTimeFormat('en', {
   weekday: 'short', month: 'short', day: 'numeric', year: 'numeric'
});

export function date(timestamp) {
   const time = new Date(parseInt(timestamp, 36) * 1000);
   
   return formatter.formatToParts(time)
      .filter(part => part.type !== 'literal')
      .map(part => part.value)
      .join(' ');
}

export function bandcamp(query) {
   return {
      href: 'https://bandcamp.com/EmbeddedPlayer/track=' + query.get('b'),
      // 350 x 350
      src: 'https://f4.bcbits.com/img/a' + query.get('c') + '_2'
   };
}

export function soundcloud(query) {
   const params = new URLSearchParams({
      url: 'api.soundcloud.com/tracks/' + query.get('b'),
   });
   return {
      href: 'https://w.soundcloud.com/player?' + params.toString(),
      src: 'https://i1.sndcdn.com/' + query.get('c')
   };
}

export function youtube(query) {
   const image = query.has('c') ? query.get('c') : 'sddefault.webp';
   const path = query.get('b') + '/' + image;
   
   return {
      href: 'https://www.youtube.com/watch?v=' + query.get('b'),
      src: image.endsWith('.webp') ? 'https://i.ytimg.com/vi_webp/' + path : 'https://i.ytimg.com/vi/' + path
   };
}
