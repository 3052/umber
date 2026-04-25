'use strict';

import {
   date,
   bandcamp,
   http,
   soundcloud,
   youtube
} from '/umber/platform.js';

const template = document.querySelector('template');
const limit = 25;

function build(row) {
   const query = new URLSearchParams(row.Q);
   const clone = template.content.cloneNode(true);
   const media = resolve(query);
   
   const link = clone.querySelector('a');
   link.target = '_blank';
   link.href = media.href;
   
   const image = clone.querySelector('img');
   image.src = media.src;
   
   const title = clone.querySelector('thead td');
   title.textContent = row.S;
   
   const release = clone.querySelector('.release');
   release.textContent = query.get('y');
   
   const posted = clone.querySelector('.post');
   posted.textContent = date(query.get('a'));
   
   const counter = clone.querySelector('.count');
   const saved = localStorage.getItem(link.href);
   if (saved !== null) {
      counter.textContent = saved;
   }
   
   const upvote = clone.querySelector('.up');
   const downvote = clone.querySelector('.down');

   upvote.addEventListener('click', () => {
      const score = Number(localStorage.getItem(link.href)) + 1;
      if (score === 0) {
         localStorage.removeItem(link.href);
         counter.textContent = '';
      } else {
         localStorage.setItem(link.href, score);
         counter.textContent = score;
      }
   });

   downvote.addEventListener('click', () => {
      const score = Number(localStorage.getItem(link.href)) - 1;
      if (score === 0) {
         localStorage.removeItem(link.href);
         counter.textContent = '';
      } else {
         localStorage.setItem(link.href, score);
         counter.textContent = score;
      }
   });

   return clone;
}

const sources = {
   b: bandcamp,
   h: http,
   s: soundcloud,
   y: youtube
};

function resolve(query) {
   return sources[query.get('p')](query);
}

async function main() {
   if (location.search === '' || localStorage.getItem('umber') === null) {
      const response = await fetch('/umber/umber.json');
      const text = await response.text();
      localStorage.setItem('umber', text);
   }
   const text = localStorage.getItem('umber');
   let records = JSON.parse(text);

   if (query.has('s')) {
      const pattern = new RegExp(query.get('s'), 'i');
      records = records.filter(row => pattern.test(row.S));
   }

   let minimum = Infinity;

   records = records.map(row => {
      const params = new URLSearchParams(row.Q);
      const url = resolve(params).href;
      const stored = localStorage.getItem(url);
      const score = stored !== null ? Number(stored) : 0;
      
      if (Math.abs(score) < minimum) {
         minimum = Math.abs(score);
      }
      
      return {
         row,
         url,
         score,
         time: parseInt(params.get('a'), 36)
      };
   });

   if (minimum > 0 && records.length > 0) {
      for (const item of records) {
         localStorage.removeItem(item.url);
         item.score = 0;
      }
   }

   records.sort((x, y) => {
      const left = Math.abs(x.score);
      const right = Math.abs(y.score);
      if (left !== right) {
         return left - right;
      }
      return y.time - x.time;
   });

   records = records.map(item => item.row);

   const page = query.has('page') ? parseInt(query.get('page'), 10) : 1;
   const start = (page - 1) * limit;

   if (page > 1) {
      document.title = 'Umber - Page ' + page;
   }

   const chunk = records.slice(start, start + limit);
   document.getElementById('figures').append(...chunk.map(build));

   const older = document.getElementById('older');
   if (start + limit < records.length) {
      query.set('page', page + 1);
      older.href = '?' + query.toString();
   } else {
      older.remove();
   }

   const newer = document.getElementById('newer');
   if (page > 1) {
      query.set('page', page - 1);
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
