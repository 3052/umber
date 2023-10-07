'use strict';

function append(msg) {
   const fig = temp.content.firstElementChild.cloneNode(true);
   main.append(fig);
   const vid = fig.querySelector('video');
   // src
   vid.src = msg.src;
   vid.onended = next;
   // poster
   vid.poster = msg.poster;
   // If you dont start playback within 30 seconds, then YouTube blocks playback.
   // `preload` doesnt help anything. `load()` doesnt help anything.
   vid.play();
   setTimeout(function() {
      vid.pause();
   }, 99);
   // title
   fig.querySelector('figcaption').textContent = msg.title;
}

function next() {
   main.querySelector('figure').remove();
   const vid = main.querySelector('video');
   if (vid !== null) {
      vid.play();
   }
}

const main = document.querySelector('main');
const temp = document.querySelector('template');
browser.runtime.onMessage.addListener(append);
