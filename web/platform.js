'use strict';

export function new_http(q) {
   const back = {};
   back.href = q.get('b');
   back.src = q.get('c');
   return back;
}

export function date_format(id) {
   const date_var = new Date(parseInt(id, 36) * 1000);
   const time_fmt = new Intl.DateTimeFormat('en', {
      weekday: 'short', month: 'short', day: 'numeric', year: 'numeric'
   });
   
   return time_fmt.formatToParts(date_var).filter(function(p) {
      return p.type !== 'literal';
   }).map(function(p) {
      return p.value;
   }).join(' ');
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
   
   let img = 'sddefault.webp';
   if (param.has('c')) {
      img = param.get('c');
   }
   
   const path = param.get('b') + '/' + img;
   
   if (img.endsWith('.webp')) {
      yt.src = 'https://i.ytimg.com/vi_webp/' + path;
   } else {
      yt.src = 'https://i.ytimg.com/vi/' + path;
   }
   return yt;
}
