# Button remap

Requires a daily build containing the FS#13975 fix (multiple `ACTION_NONE`
entries per context allowed). All current daily builds include it.

## How to load the remap

1. Copy the remap txt file to the SD card root
2. On device: **Applications → keyremap**
3. **Load keymap → Import Text Keymap** → select the remap txt file
4. The import saves and activates the keymap (`/settings/.keymap`); reboot if
   changes don't apply immediately
5. To disable: keyremap → **Reset to default** (or delete the `.keymap` file)
