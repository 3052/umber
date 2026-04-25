'use strict';

function seek() {
   const seconds = parseInt(input.value, 10);
   console.log(seconds);
   document.querySelector('video').currentTime = seconds;
}

const input = document.getElementById('timeInput');

input.addEventListener('keydown', function(event) {
   if (event.key === 'Enter') {
      seek();
   }
});

const main = document.querySelector('main');

// Fast multithreaded downloader to bypass YouTube's 1x speed throttle
async function download(url) {
   try {
      // 1. Get the total file size using a tiny 1-byte request
      const probe = await fetch(url, { headers: { Range: 'bytes=0-0' } });
      const header = probe.headers.get('content-range'); // e.g. "bytes 0-0/1234567"
      await probe.arrayBuffer(); // consume the 1 byte so connection closes cleanly
      
      if (header === null) {
         // If the server doesn't support Range requests, fallback to standard download
         const response = await fetch(url);
         return await response.blob();
      }
      
      const total = parseInt(header.split('/')[1], 10);
      const threads = 2; // 2 concurrent connections to be safe while bypassing throttle
      const chunk = Math.ceil(total / threads);
      const tasks = [];
      
      // 2. Fetch the chunks in parallel
      for (let i = 0; i < threads; i++) {
         const start = i * chunk;
         const end = Math.min((i + 1) * chunk - 1, total - 1);
         tasks.push(
            fetch(url, { headers: { Range: `bytes=${start}-${end}` } })
            .then(response => response.arrayBuffer())
         );
      }
      
      // 3. Reassemble the pieces in order
      const buffers = await Promise.all(tasks);
      return new Blob(buffers);
      
   } catch (error) {
      console.error("Multithreaded download failed, falling back to single thread", error);
      const response = await fetch(url);
      return await response.blob();
   }
}

browser.runtime.onMessage.addListener(async function(message) {
   const template = document.querySelector('template');
   const figure = template.content.firstElementChild.cloneNode(true);
   main.append(figure);
   const video = figure.querySelector('video');
   const caption = figure.querySelector('figcaption');
   
   video.poster = message.poster;
   
   video.onended = function() {
      main.querySelector('figure').remove();
      // Revoke the Blob URL to free up memory once the video is done
      if (video.src.startsWith('blob:')) {
         URL.revokeObjectURL(video.src);
      }
      const next = main.querySelector('video');
      if (next !== null) {
         next.play();
      }
   };

   if (message.src !== '') {
      // Only apply the multithreaded Blob builder to YouTube URLs
      if (message.src.includes('googlevideo.com')) {
         caption.textContent = message.title + ' (Downloading...)';
         try {
            const blob = await download(message.src);
            video.src = URL.createObjectURL(blob);
            caption.textContent = message.title;
         } catch (error) {
            caption.textContent = message.title + ' (Download failed)';
         }
      } else {
         // Bandcamp and SoundCloud don't throttle or expire like YouTube
         video.src = message.src;
         caption.textContent = message.title;
      }
   } else {
      caption.textContent = message.title;
   }
});
