# Button remap

## How to disable the remap

1. Open the keyremap plugin
2. Remove Core Remap (renames the file so it no longer loads at startup)

## What to check in Edit Keymap

The key check — multiple `ACTION_NONE` entries survived:
- `CONTEXT_WPS` should have 9 `ACTION_NONE` entries
- `CONTEXT_MAINMENU`, `CONTEXT_TREE`, `CONTEXT_LIST` should have 5
  `ACTION_NONE` entries each

## How to load the remap

1. Copy the remap txt file to the SD card root
2. On device: Applications → keyremap
3. Import Text Keymap → select the remap txt file
4. Edit Keymap → verify the import (see below)
5. Set Core Remap → activates the remap and makes it load at every
   startup (writes `.rockbox/keyremap.kmf`)
6. Quit — accept the save prompt if it appears
