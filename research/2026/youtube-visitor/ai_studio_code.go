package main

import (
   "encoding/json"
   "fmt"
   "log"
   "os"
   "strings"
)

// extractJSON isolates the JSON payload by balancing curly braces
func extractJSON(content, prefix string) (string, error) {
   startIdx := strings.Index(content, prefix)
   if startIdx == -1 {
      return "", fmt.Errorf("prefix %q not found in file", prefix)
   }

   // Move the index forward to where the JSON object actually begins
   jsonStart := startIdx + len(prefix)

   // Make sure we are actually starting at a curly brace
   if content[jsonStart] != '{' {
      return "", fmt.Errorf("expected '{' at the start of JSON, got %c", content[jsonStart])
   }

   openBraces := 0
   inString := false
   escapeNext := false

   // Parse through the characters to find the exact end of the JSON object
   for i := jsonStart; i < len(content); i++ {
      char := content[i]

      // Handle escaped characters (e.g., \")
      if escapeNext {
         escapeNext = false
         continue
      }
      if char == '\\' {
         escapeNext = true
         continue
      }

      // Toggle string state to ignore braces inside strings
      if char == '"' {
         inString = !inString
         continue
      }

      // If we are not inside a string literal, count braces
      if !inString {
         if char == '{' {
            openBraces++
         } else if char == '}' {
            openBraces--
            
            // When the count goes back to 0, we've found the end of the JSON body
            if openBraces == 0 {
               return content[jsonStart : i+1], nil
            }
         }
      }
   }

   return "", fmt.Errorf("could not find the matching closing brace for the JSON object")
}

func main() {
   // 1. Read the file (assuming it's named "youtube.txt")
   fileBytes, err := os.ReadFile("youtube.txt")
   if err != nil {
      log.Fatalf("Failed to read file: %v", err)
   }

   // 2. Extract the JSON string starting immediately after 'ytcfg.set('
   jsonString, err := extractJSON(string(fileBytes), "ytcfg.set(")
   if err != nil {
      log.Fatalf("Extraction failed: %v", err)
   }

   // 3. Parse the extracted JSON into a Go map or struct to prove it is valid
   var parsedData map[string]interface{}
   err = json.Unmarshal([]byte(jsonString), &parsedData)
   if err != nil {
      log.Fatalf("Failed to decode JSON: %v", err)
   }

   // Output success and test a specific key to verify
   fmt.Println("Successfully extracted and parsed JSON!")
   if eventID, ok := parsedData["EVENT_ID"]; ok {
      fmt.Printf("Example data retrieved -> EVENT_ID: %v\n", eventID)
   }
}
