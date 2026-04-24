'use strict';

export function new_http(q) {
   return {
      href: q.get('b'),
      src: q.get('c')
   };
}

const time_fmt = new Intl.DateTimeFormat('en', {
   weekday: 'short', month: 'short', day: 'numeric', year: 'numeric'
});

export function date_format(id) {
   const new_date = new Date(parseInt(id, 36) * 1000);
   
   return time_fmt.formatToParts(new_date)
      .filter(p => p.type !== 'literal')
      .map(p => p.value)
      .join(' ');
}

export function new_bandcamp(param) {
   return {
      href: 'https://bandcamp.com/EmbeddedPlayer/track=' + param.get('b'),
      // 350 x 350
      src: 'https://f4.bcbits.com/img/a' + param.get('c') + '_2'
   };
}

export function new_soundcloud(param) {
   const play = new URLSearchParams({
      url: 'api.soundcloud.com/tracks/' + param.get('b'),
   });
   return {
      href: 'https://w.soundcloud.com/player?' + play.toString(),
      src: 'https://i1.sndcdn.com/' + param.get('c')
   };
}

export function new_youtube(param) {
   const img = param.has('c') ? param.get('c') : 'sddefault.webp';
   const path = param.get('b') + '/' + img;
   
   return {
      href: 'https://www.youtube.com/watch?v=' + param.get('b'),
      src: img.endsWith('.webp') ? 'https://i.ytimg.com/vi_webp/' + path : 'https://i.ytimg.com/vi/' + path
   };
}
