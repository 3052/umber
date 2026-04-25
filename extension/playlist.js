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

// Fast multithreaded downloader to bypass YouTube's 1x speed throttle
async function fastDownload(url) {
   try {
      // 1. Get the total file size using a tiny 1-byte request
      const initRes = await fetch(url, { headers: { Range: 'bytes=0-0' } });
      const rangeMatch = initRes.headers.get('content-range'); // e.g. "bytes 0-0/1234567"
      await initRes.arrayBuffer(); // consume the 1 byte so connection closes cleanly
      
      if (!rangeMatch) {
         // If the server doesn't support Range requests, fallback to standard download
         const res = await fetch(url);
         return await res.blob();
      }
      
      const totalSize = parseInt(rangeMatch.split('/')[1], 10);
      const threads = 2; // 2 concurrent connections to be safe while bypassing throttle
      const chunkSize = Math.ceil(totalSize / threads);
      const promises = [];
      
      // 2. Fetch the chunks in parallel
      for (let i = 0; i < threads; i++) {
         const start = i * chunkSize;
         const end = Math.min((i + 1) * chunkSize - 1, totalSize - 1);
         promises.push(
            fetch(url, { headers: { Range: `bytes=${start}-${end}` } })
            .then(r => r.arrayBuffer())
         );
      }
      
      // 3. Reassemble the pieces in order
      const buffers = await Promise.all(promises);
      return new Blob(buffers);
      
   } catch (err) {
      console.error("Multithreaded download failed, falling back to single thread", err);
      const res = await fetch(url);
      return await res.blob();
   }
}

browser.runtime.onMessage.addListener(async function(event) {
   const temp = document.querySelector('template');
   const fig = temp.content.firstElementChild.cloneNode(true);
   main.append(fig);
   const vid = fig.querySelector('video');
   const caption = fig.querySelector('figcaption');
   
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

   if (event.src) {
      // Only apply the multithreaded Blob builder to YouTube URLs
      if (event.src.includes('googlevideo.com')) {
         caption.textContent = event.title + ' (Downloading...)';
         try {
            const blob = await fastDownload(event.src);
            vid.src = URL.createObjectURL(blob);
            caption.textContent = event.title;
         } catch (error) {
            caption.textContent = event.title + ' (Download failed)';
         }
      } else {
         // Bandcamp and SoundCloud don't throttle or expire like YouTube
         vid.src = event.src;
         caption.textContent = event.title;
      }
   } else {
      caption.textContent = event.title;
   }
});
