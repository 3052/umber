'use strict';

export function new_http(q) {
   const back = {};
   back.href = q.get('b');
   back.src = q.get('c');
   return back;
}

const date_parts = [
   {weekday: 'short'}, {month: 'short'}, {day: 'numeric'}, {year: 'numeric'}
];

export function date_format(id) {
   const parse = parseInt(id, 36);
   const date_var = new Date(parse * 1000);
   function format(part) {
      const time = new Intl.DateTimeFormat('en', part);
      return time.format(date_var);
   }
   return date_parts.map(format).join(' ');
}

export function new_bandcamp(param) {
   const band = {};
   band.href = 'https://bandcamp.com/EmbeddedPlayer/track=' + param.get('b');
   // 350 x 350
   band.src = 'https://f4.bcbits.com/img/a' + param.get('c') + '_2';
   return band;
}

export function new_soundcloud(param) {
   const play = new URLSearchParams({
      url: 'api.soundcloud.com/tracks/' + param.get('b'),
   });
   const sc = {};
   sc.href = 'https://w.soundcloud.com/player?' + play.toString();
   sc.src = 'https://i1.sndcdn.com/' + param.get('c');
   return sc;
}

export function new_youtube(param) {
   const yt = {};
   yt.href = 'https://www.youtube.com/watch?v=' + param.get('b');
   if (param.has('c')) {
      yt.src = param.get('c');
   } else {
      yt.src = 'sddefault.webp';
   }
   yt.src = param.get('b') + '/' + yt.src;
   // need HTTPS to avoid "Parts of this page are not secure"
   if (yt.src.endsWith('.webp')) {
      yt.src = 'https://i.ytimg.com/vi_webp/' + yt.src;
   } else {
      yt.src = 'https://i.ytimg.com/vi/' + yt.src;
   }
   return yt;
}
