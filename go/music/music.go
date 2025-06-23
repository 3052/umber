package music

var KellyMoran = []album{
   {
      name: "helix (edit)",
      link: []link{
         {url: "youtube.com/playlist?list=OLAK5uy_kB2WFbIR3N8VhQWq0G94nUUoa1275EcMU"},
      },
      track: []track{
         {
            number: 1,
            name:   "helix (edit)",
            link: []link{
               {url: "youtube.com/watch?v=tdeZ4ecFaxE"},
            },
         },
      },
   },
   {
      name: "moves in the field",
      link: []link{
         {url: "youtube.com/playlist?list=OLAK5uy_mI7sh2DxPKFiColG-vykv9e74yqdKQuMI"},
      },
      track: []track{
         {
            number: 4,
            name:   "Dancer Polynomials",
            link: []link{
               {url: "youtube.com/watch?v=E6LxCW7cl4E"},
            },
         },
         {
            number: 5,
            name:   "sodalis",
            link: []link{
               {url: "youtube.com/watch?v=VBlhTlz1LXA"},
            },
         },
         {
            number: 6,
            name:   "leitmotif",
            link: []link{
               {url: "youtube.com/watch?v=o63kYG-j6Lg"},
            },
         },
      },
   },
   {
      name: "ultraviolet",
      link: []link{
         {url: "youtube.com/playlist?list=OLAK5uy_mQz8ftbRzC-chKW0YSJ6yyPmgdrHv12CI"},
      },
      track: []track{
         {
            number: 1,
            name:   "autowave",
            link: []link{
               {url: "youtube.com/watch?v=vD1m4flw_Ko"},
            },
         },
      },
   },
}

type album struct {
   name  string
   link  []link
   track []track
}

type track struct {
   number int
   name   string
   link   []link
}

type link struct {
   text string
   url  string
}
