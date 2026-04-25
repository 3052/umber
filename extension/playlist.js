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

browser.runtime.onMessage.addListener(async function(event) {
   const temp = document.querySelector('template');
   const fig = temp.content.firstElementChild.cloneNode(true);
   main.append(fig);
   const vid = fig.querySelector('video');
   
   // poster
   vid.poster = event.poster;
   
   vid.onended = function() {
      main.querySelector('figure').remove();
      // Revoke the Blob URL to free up memory once the video is done
      if (vid.src.startsWith('blob:')) {
         URL.revokeObjectURL(vid.src);
      }
      const nextVid = main.querySelector('video');
      if (nextVid !== null) {
         nextVid.play();
      }
   };

   const caption = fig.querySelector('figcaption');

   // Fetching the source as a Blob fully downloads the file to memory.
   // This entirely bypasses YouTube's 30-second playback block without
   // needing the old play()/pause() hack, and ensures the whole file is preloaded.
   if (event.src) {
      caption.textContent = event.title + ' (Downloading...)';
      try {
         const response = await fetch(event.src);
         const blob = await response.blob();
         vid.src = URL.createObjectURL(blob);
         caption.textContent = event.title;
      } catch (error) {
         console.error("Failed to preload media:", error);
         caption.textContent = event.title + ' (Download failed)';
      }
   } else {
      caption.textContent = event.title;
   }
});
