'use strict';

import {
   date_format,
   new_bandcamp,
   new_http,
   new_soundcloud,
   new_youtube
} from '/umber/platform.js';

const temp = document.querySelector('template');
const per_page = 25;

function figure(row) {
   const param = new URLSearchParams(row.Q);
   const clone = temp.content.cloneNode(true);
   const attr = href_src(param);
   const anc = clone.querySelector('a');
   anc.target = '_blank';
   anc.href = attr.href;
   const img = clone.querySelector('img');
   img.src = attr.src;
   const thead = clone.querySelector('thead td');
   thead.textContent = row.S;
   const rel = clone.querySelector('.release');
   rel.textContent = param.get('y');
   const post = clone.querySelector('.post');
   post.textContent = date_format(param.get('a'));
   const td_view = clone.querySelector('td.view');
   const th_view = clone.querySelector('th.view');
   const view = localStorage.getItem(anc.href);
   if (view !== null) {
      td_view.textContent = view;
   } else {
      th_view.style.display = 'none';
   }
   
   const views = () => {
      const count = Number(localStorage.getItem(anc.href)) + 1;
      localStorage.setItem(anc.href, count);
      th_view.style.display = td_view.style.display = '';
      td_view.textContent = count;
   };
   // web
   anc.addEventListener('click', views);
   // mobile
   anc.addEventListener('contextmenu', views);
   return clone;
}

const platforms = {
   b: new_bandcamp,
   h: new_http,
   s: new_soundcloud,
   y: new_youtube
};

function href_src(query) {
   return platforms[query.get('p')](query);
}

async function main() {
   if (location.search === '' || localStorage.getItem('umber') === null) {
      const resp = await fetch('/umber/umber.json');
      const text = await resp.text();
      localStorage.setItem('umber', text);
   }
   const text = localStorage.getItem('umber');
   let table = JSON.parse(text);

   if (search.has('s')) {
      const re = new RegExp(search.get('s'), 'i');
      table = table.filter(row => re.test(row.S));
   }

   let min_view = Infinity;

   table = table.map(row => {
      const q = new URLSearchParams(row.Q);
      const href = href_src(q).href;
      const raw_view = localStorage.getItem(href);
      const views = raw_view !== null ? Number(raw_view) : 0;
      
      if (views < min_view) {
         min_view = views;
      }
      
      return {
         row: row,
         href: href,
         views: views,
         date: parseInt(q.get('a'), 36)
      };
   });

   if (min_view > 0 && table.length > 0) {
      for (const item of table) {
         localStorage.removeItem(item.href);
         item.views = 0;
      }
   }

   table.sort((x, y) => {
      if (x.views !== y.views) {
         return x.views - y.views;
      }
      return y.date - x.date;
   });

   table = table.map(item => item.row);

   const page = search.has('page') ? parseInt(search.get('page'), 10) : 1;
   const begin = (page - 1) * per_page;

   if (page > 1) {
      document.title = 'Umber - Page ' + page;
   }

   const slice = table.slice(begin, begin + per_page);
   document.getElementById('figures').append(...slice.map(figure));

   const older = document.getElementById('older');
   if (begin + per_page < table.length) {
      search.set('page', page + 1);
      older.href = '?' + search.toString();
   } else {
      older.remove();
   }

   const newer = document.getElementById('newer');
   if (page > 1) {
      search.set('page', page - 1);
      newer.href = '?' + search.toString();
   } else {
      newer.remove();
   }

   const fresh = document.getElementById('fresh');
   if (fresh !== null) {
      fresh.addEventListener('click', () => {
         for (const row of slice) {
            const q = new URLSearchParams(row.Q);
            const href = href_src(q).href;
            const count = Number(localStorage.getItem(href)) + 1;
            localStorage.setItem(href, count);
         }
         location.reload();
      });
   }
}

document.querySelector('form').onsubmit = function() {
   document.querySelector('input').blur();
   this.submit();
   this.reset();
   return false;
};

const search = new URLSearchParams(location.search);
main();
