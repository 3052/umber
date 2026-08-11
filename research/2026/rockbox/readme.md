# rockbox

## 1. Download the patched bootloader update file

Grab the pre-patched update file for firmware (need final slash):

https://download.rockbox.org/bootloader/hiby/r1/

Rename the downloaded file to exactly:

```
r1.upt
```

Place `r1.upt` into the **root** directory of your SD card.

## 2. Download the Rockbox daily build
Get the latest Hosted Port daily build for the R1 from the Rockbox build server:

https://rockbox.org/download/devbuilds.html

Look for the **HiBy R1** entry and download the build. Unzip the contents into
the **root** of the SD card. You should end up with a `.rockbox` folder on the
card.

## 3. Flash the bootloader
On the R1's stock firmware:
1. Insert the SD card
2. Go to **System → Firmware Update → TF card upgrade**
3. The device will reboot and flash the patched firmware (which includes the
   Rockbox bootloader)
