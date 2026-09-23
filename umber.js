// umber.js marker preserve
'use strict';

import {
   date,
   bandcamp,
   http,
   soundcloud,
   youtube
} from '/umber/platform.js';

const template = document.querySelector('template');
const limit = 10;

const sources = {
   bandcamp: bandcamp,
   http: http,
   soundcloud: soundcloud,
   youtube: youtube
};

function build(row) {
   const clone = template.content.cloneNode(true);
   const platform = row.P !== undefined ? row.P : 'youtube';
   const media = sources[platform](row);

   const link = clone.querySelector('a');
   link.target = '_blank';
   link.href = media.href;

   const image = clone.querySelector('img');
   image.src = media.src;

   const title = clone.querySelector('thead td');
   title.textContent = row.T !== undefined ? row.T : '';

   const release = clone.querySelector('.release');
   release.textContent = row.Y !== undefined ? row.Y.toString(10) : '';

   const posted = clone.querySelector('.post');
   posted.textContent = date(row.D);

   return clone;
}

async function main() {
   const response = await fetch('/umber/umber.json');
   let records = await response.json();

   if (query.has('t')) {
      const searchParam = query.get('t');
      if (searchParam !== null) {
         const pattern = new RegExp(searchParam, 'i');
         records = records.filter(row => pattern.test(row.T));
      }
   }

   records.sort((x, y) => y.D - x.D);

   const pageParam = query.get('page');
   const start = pageParam === null ? 0 : records.findIndex(row => String(row.I) === pageParam);

   if (start === -1) {
      return;
   }

   const chunk = records.slice(start, start + limit);
   document.getElementById('figures').append(...chunk.map(build));

   const older = document.getElementById('older');
   if (start + limit < records.length) {
      query.set('page', String(records[start + limit].I));
      older.href = '?' + query.toString();
   } else {
      older.remove();
   }

   const newer = document.getElementById('newer');
   if (start > 0) {
      query.set('page', String(records[Math.max(0, start - limit)].I));
      newer.href = '?' + query.toString();
   } else {
      newer.remove();
   }
}

document.querySelector('form').onsubmit = function() {
   document.querySelector('input').blur();
   this.submit();
   this.reset();
   return false;
};

const query = new URLSearchParams(location.search);
main();
// umber.js marker preserve
