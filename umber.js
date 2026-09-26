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
   const R = row.I.startsWith('https://youtube.com/') && row.R.endsWith(' - Topic')
      ? row.R.slice(0, -' - Topic'.length)
      : row.R;

   return row.T.toLowerCase().includes(R.toLowerCase()) ? row.T : R + ' - ' + row.T;
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
   release.textContent = row.Y.toString(10);

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
   const start = timeParam === null ? 0 : records.findIndex(row => row.D <= Number(timeParam));

   if (start === -1) {
      return;
   }

   const chunk = records.slice(start, start + limit);
   document.getElementById('figures').append(...chunk.map(build));

   // the roundest timestamp that still opens the page starting at `index`:
   // any value in [records[index].D, records[index - 1].D) does, so take the
   // one with the most trailing zeroes that fits below the previous record
   const landmark = index => {
      const target = records[index].D;
      const before = index === 0 ? Infinity : records[index - 1].D;

      for (let digits = 9; digits > 0; digits--) {
         const power = 10 ** digits;
         const remainder = target % power;
         const round = remainder ? target - remainder + power : target;
         if (round < before) {
            return round;
         }
      }

      return target; // neighbours too close to round at all
   };

   const older = document.getElementById('older');
   if (start + limit < records.length) {
      query.set('d', String(landmark(start + limit)));
      older.href = '?' + query.toString();
   } else {
      older.remove();
   }

   const newer = document.getElementById('newer');
   if (start > 0) {
      const target = Math.max(0, start - limit);
      if (target === 0) {
         query.delete('d'); // rounder than any round number: no d at all
      } else {
         query.set('d', String(landmark(target)));
      }
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
