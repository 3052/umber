// umber.js marker preserve
'use strict';

const formatter = new Intl.DateTimeFormat('en', {
   weekday: 'short',
   month: 'short',
   day: 'numeric',
   year: 'numeric'
});

function date(timestamp) {
   const time = new Date(timestamp * 1000);

   return formatter.formatToParts(time)
      .filter(part => part.type !== 'literal')
      .map(part => part.value)
      .join(' ');
}

function media(row) {
   let image = row.A;
   if (image === undefined) {
      // A is omitted only for YouTube rows on the default thumbnail
      const video = new URL(row.I).searchParams.get('v');
      image = 'https://i.ytimg.com/vi_webp/' + video + '/sddefault.webp';
   }

   return {
      href: row.I,
      src: image
   };
}

const template = document.querySelector('template');
const limit = 10;

function label(row) {
   return row.R === undefined ? row.T : row.R + ' - ' + row.T;
}

function build(row) {
   const clone = template.content.cloneNode(true);
   const { href, src } = media(row);

   const link = clone.querySelector('a');
   link.target = '_blank';
   link.href = href;

   const image = clone.querySelector('img');
   image.src = src;

   const title = clone.querySelector('thead td');
   title.textContent = label(row);

   const release = clone.querySelector('.release');
   release.textContent = row.Y !== undefined ? row.Y.toString(10) : '';

   const posted = clone.querySelector('.post');
   posted.textContent = date(row.D);

   const td_view = clone.querySelector('td.view');
   const th_view = clone.querySelector('th.view');
   const view = localStorage.getItem(link.href);
   if (view !== null) {
      td_view.textContent = view;
   } else {
      th_view.style.display = 'none';
   }

   const views = () => {
      const count = Number(localStorage.getItem(link.href)) + 1;
      localStorage.setItem(link.href, count);
      th_view.style.display = td_view.style.display = '';
      td_view.textContent = count;
   };
   link.addEventListener('click', views);

   return clone;
}

async function main() {
   const response = await fetch('/umber/umber.json');
   let records = await response.json();

   if (query.has('t')) {
      const searchParam = query.get('t');
      if (searchParam !== null) {
         const pattern = new RegExp(searchParam, 'i');
         records = records.filter(row => pattern.test(label(row)));
      }
   }

   records.sort((x, y) => y.D - x.D);

   const timeParam = query.get('d');
   const start = timeParam === null ? 0 : records.findIndex(row => String(row.D) === timeParam);

   if (start === -1) {
      return;
   }

   const chunk = records.slice(start, start + limit);
   document.getElementById('figures').append(...chunk.map(build));

   const older = document.getElementById('older');
   if (start + limit < records.length) {
      query.set('d', String(records[start + limit].D));
      older.href = '?' + query.toString();
   } else {
      older.remove();
   }

   const newer = document.getElementById('newer');
   if (start > 0) {
      query.set('d', String(records[Math.max(0, start - limit)].D));
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
