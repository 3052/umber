'use strict';

import {
   date_format,
   new_backblaze,
   new_bandcamp,
   new_vimeo,
   new_soundcloud,
   new_youtube
} from '/umber/assets/platform.js';

function figure(row) {
   const sp = new URLSearchParams(row.Q);
   const temp = document.querySelector('template');
   const a_id = sp.get('a');
   const clone = temp.content.cloneNode(true);
   const attr = href_src(sp, row.S);
   const anc = clone.querySelector('a');
   anc.target = '_blank';
   anc.href = attr.href;
   const img = clone.querySelector('img');
   img.src = attr.src;
   const thead = clone.querySelector('thead td');
   thead.textContent = row.S;
   const rel = clone.querySelector('.release');
   rel.textContent = sp.get('y');
   const post = clone.querySelector('.post');
   post.textContent = date_format(a_id);
   const td_view = clone.querySelector('td.view');
   const th_view = clone.querySelector('th.view');
   const view = localStorage.getItem(anc.href);
   if (view !== null) {
      td_view.textContent = view;
   } else {
      th_view.style.display = 'none';
   }
   // "function" for "this"
   function views() {
      const href = localStorage.getItem(this.href);
      localStorage.setItem(this.href, Number(href) + 1);
      th_view.style.display = td_view.style.display = '';
      td_view.textContent = Number(href) + 1;
   }
   // web
   anc.addEventListener('click', views);
   // mobile
   anc.addEventListener('contextmenu', views);
   return clone;
}

const per_page = 30;

function href_src(query, title) {
   switch (query.get('p')) {
   case 'b':
      return new_backblaze(query, title);
   case 'bandcamp':
      return new_bandcamp(query);
   case 's':
      return new_soundcloud(query);
   case 'v':
      return new_vimeo(query);
   case 'y':
      return new_youtube(query);
   }
}

function get_low(meds) {
   if (!search.has('a')) {
      return 0;
   }
   for (const [i, med] of meds.entries()) {
      const param = new URLSearchParams(med.Q);
      const a_id = param.get('a');
      // account for deleted entries
      if (a_id <= search.get('a')) {
         document.title = 'Umber - ' + date_format(a_id);
         return i;
      }
   }
   return 0;
}

async function main() {
   if (location.search === '') {
      const resp = await fetch('/umber/umber.json');
      const text = await resp.text();
      localStorage.setItem('umber', text);
   }
   const text = localStorage.getItem('umber');
   let table = JSON.parse(text);
   // 1. filter
   if (search.has('s')) {
      function filter(row) {
         // for now, just match one the artist, album and song.
         return RegExp(search.get('s'), 'i').test(row.S);
      }
      table = table.filter(filter);
   }
   const begin = get_low(table);
   const slice = table.slice(begin, begin + per_page);
   for (const row of slice) {
      document.getElementById('figures').append(figure(row));
   }
   const older = document.getElementById('older');
   const old_index = begin + per_page;
   if (old_index < table.length) {
      const sp = new URLSearchParams(table[old_index].Q);
      search.set('a', sp.get('a'));
      older.href = '?' + search.toString();
   } else {
      older.remove();
   }
   const newer = document.getElementById('newer');
   const new_index = begin - per_page;
   if (new_index >= 0) {
      const sp = new URLSearchParams(table[new_index].Q);
      search.set('a', sp.get('a'));
      newer.href = '?' + search.toString();
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

const search = new URLSearchParams(location.search);
main();
