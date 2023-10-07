'use strict';

function backBlaze() {
   browser.runtime.sendMessage({
      src: this.href,
      poster: this.querySelector('img').src,
      title: this.parentNode.querySelector('td').textContent
   });
}
