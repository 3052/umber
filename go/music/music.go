package music

var Albums = []album{
   {
      name: "donkey kong country",
      link: []link{
         {url: "wikipedia.org/wiki/Donkey_Kong_Country"},
         {
            text: "Jammin' Sam Miller",
            url:  "youtube.com/playlist?list=PLKNuUcvZGX21obTJzXuVHUWRcS2cCBBsP",
         },
      },
      track: []track{
         {
            number: 7,
            name:   "aquatic ambience",
            link: []link{
               {
                  text: "Aquatic Ambience [Restored]",
                  url:  "youtube.com/watch?v=-5rAjOjTGtc",
               },
               {
                  text: "Aquatic Ambience [Restored] [2023 Mix]",
                  url:  "youtube.com/watch?v=39hGqV42CkM",
               },
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
