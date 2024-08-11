package main

import (
   "154.pages.dev/platform/youtube"
   "fmt"
   "os"
   "strings"
   "time"
)

func main() {
   text, err := os.ReadFile("bot.txt")
   if err != nil {
      panic(err)
   }
   lines := strings.FieldsFunc(string(text), func(r rune) bool {
      return r == '\n'
   })
   for i, line := range lines {
      var req youtube.Request
      req.VideoId = strings.Fields(line)[1]
      req.Android()
      var play youtube.Player
      err := play.Post(req, nil)
      if err != nil {
         panic(err)
      }
      fmt.Println(play.PlayabilityStatus.Status, req.VideoId, len(lines)-i)
      time.Sleep(199 * time.Millisecond)
   }
}
