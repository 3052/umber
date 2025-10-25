'use strict';

function seekToSeconds() {
   const seconds = parseInt(timeInputElement.value, 10);
   console.log(seconds);
   document.querySelector('video').currentTime = seconds;
}

const timeInputElement = document.getElementById('timeInput');

timeInputElement.addEventListener('keydown', function(event) {
   if (event.key === 'Enter') {
      seekToSeconds();
   }
});

const main = document.querySelector('main');

browser.runtime.onMessage.addListener(function(event) {
   const temp = document.querySelector('template');
   const fig = temp.content.firstElementChild.cloneNode(true);
   main.append(fig);
   const vid = fig.querySelector('video');
   // src
   vid.src = event.src;
   vid.onended = function() {
      main.querySelector('figure').remove();
      const vid = main.querySelector('video');
      if (vid !== null) {
         vid.play();
      }
   };
   // poster
   vid.poster = event.poster;
   // If you dont start playback within 30 seconds, then YouTube blocks playback.
   // `preload` doesnt help anything. `load()` doesnt help anything.
   vid.play();
   setTimeout(function() {
      vid.pause();
   }, 99);
   // title
   fig.querySelector('figcaption').textContent = event.title;
});
