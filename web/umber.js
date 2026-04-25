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

const sources = {
   'bandcamp': bandcamp,
   'http': http,
   'soundcloud': soundcloud,
   'youtube': youtube
};

function build(row) {
   const clone = template.content.cloneNode(true);
   const platform = 'P' in row ? row.P : 'youtube';
   const media = sources[platform](row);
   
   const link = clone.querySelector('a');
   link.target = '_blank';
   link.href = media.href;
   
   const image = clone.querySelector('img');
   image.src = media.src;
   
   const title = clone.querySelector('thead td');
   title.textContent = 'T' in row ? row.T : '';
   
   const release = clone.querySelector('.release');
   release.textContent = 'Y' in row ? row.Y.toString(10) : '';
   
   const posted = clone.querySelector('.post');
   posted.textContent = date(row.D);
   
   const counter = clone.querySelector('.count');
   const saved = localStorage.getItem(link.href);
   
   if (saved !== null) {
      counter.textContent = saved;
   }
   
   const upvote = clone.querySelector('.up');
   const downvote = clone.querySelector('.down');

   upvote.addEventListener('click', () => {
      const stored = localStorage.getItem(link.href);
      const score = (stored === null ? 0 : Number(stored)) + 1;
      
      if (score === 0) {
         localStorage.removeItem(link.href);
         counter.textContent = '';
      } else {
         localStorage.setItem(link.href, score.toString(10));
         counter.textContent = score.toString(10);
      }
   });

   downvote.addEventListener('click', () => {
      const stored = localStorage.getItem(link.href);
      const score = (stored === null ? 0 : Number(stored)) - 1;
      
      if (score === 0) {
         localStorage.removeItem(link.href);
         counter.textContent = '';
      } else {
         localStorage.setItem(link.href, score.toString(10));
         counter.textContent = score.toString(10);
      }
   });

   return clone;
}

async function main() {
   let text = localStorage.getItem('umber');
   
   if (text === null) {
      const response = await fetch('/umber/umber.json');
      text = await response.text();
      localStorage.setItem('umber', text);
   }
   
   let records = JSON.parse(text);

   if (query.has('t')) {
      const searchParam = query.get('t');
      if (searchParam !== null) {
         const pattern = new RegExp(searchParam, 'i');
         records = records.filter(row => pattern.test(row.T));
      }
   }

   let minimum = Number.POSITIVE_INFINITY;

   records = records.map(row => {
      const platform = 'P' in row ? row.P : 'youtube';
      const url = sources[platform](row).href;
      const stored = localStorage.getItem(url);
      const score = stored !== null ? Number(stored) : 0;
      
      if (Math.abs(score) < minimum) {
         minimum = Math.abs(score);
      }
      
      return {
         row,
         url,
         score,
         time: row.D
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

   const pageParam = query.get('page');
   const page = pageParam !== null ? parseInt(pageParam, 10) : 1;
   const start = (page - 1) * limit;

   if (page > 1) {
      document.title = 'Umber - Page ' + page.toString(10);
   }

   const chunk = records.slice(start, start + limit);
   document.getElementById('figures').append(...chunk.map(build));

   const older = document.getElementById('older');
   if (start + limit < records.length) {
      query.set('page', (page + 1).toString(10));
      older.href = '?' + query.toString();
   } else {
      older.remove();
   }

   const newer = document.getElementById('newer');
   if (page > 1) {
      query.set('page', (page - 1).toString(10));
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
