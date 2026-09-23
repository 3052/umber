// platform.js marker preserve
'use strict';

const formatter = new Intl.DateTimeFormat('en', {
   weekday: 'short', 
   month: 'short', 
   day: 'numeric',
   year: 'numeric'
});

export function date(timestamp) {
   const time = new Date(timestamp * 1000);
   
   return formatter.formatToParts(time)
      .filter(part => part.type !== 'literal')
      .map(part => part.value)
      .join(' ');
}

export function media(row) {
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
// platform.js marker preserve
