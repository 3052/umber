'use strict';

const time = 99;
const count = 99;

function delay(callback, time, count) {
   const id = setInterval(function() {
      const ok = callback();
      count--;
      if (ok || count == 0) {
         clearInterval(id);
      }
   }, time);
}

delay(function() {
   const as = document.getElementsByTagName('a');
   if (as.length == 0) {
      return false;
   }
   for (const a of as) {
      // We already have an event for localStorage, so use addEventListener so
      // we dont clobber that.
      switch (true) {
      case a.host == 'bandcamp.com':
         a.addEventListener('contextmenu', bandcamp);
         break;
      case a.host == 'w.soundcloud.com':
         a.addEventListener('contextmenu', soundCloud);
         break;
      case a.host == 'www.youtube.com':
         a.addEventListener('contextmenu', youTube);
         break;
      case a.host.endsWith('.backblazeb2.com'):
         a.addEventListener('contextmenu', backBlaze);
         break;
      default:
         continue;
      }
      a.style.borderTop = '2px solid red';
   }
   return true;
}, time, count);
