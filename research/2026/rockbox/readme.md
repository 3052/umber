# Rockbox on the HiBy R1

## Download and flash the patched bootloader

Grab the pre-patched update file for firmware (need final slash):

https://download.rockbox.org/bootloader/hiby/r1/

Rename the downloaded file to exactly:

```
r1.upt
```

Place `r1.upt` into the **root** directory of your SD card. On the R1's stock
firmware:
1. Insert the SD card
2. Go to **System → Firmware Update → TF card upgrade**
3. The device will reboot and flash the patched firmware (which includes the
   Rockbox bootloader)

## Install the Rockbox daily build

Get the latest Hosted Port daily build for the R1:

https://rockbox.org/download/devbuilds.html

Look for the **HiBy R1** entry and download the build. Unzip the contents into
the **root** of the SD card. You should end up with a `.rockbox` folder on the
card. To update Rockbox later, just repeat this step — no re-flash needed.

## Status bar — `.rockbox/fonts/27-Adobe-Helvetica.fnt`

https://download.rockbox.org/daily/fonts/rockbox-fonts.zip

## Status bar — `.rockbox\themes\cabbiev2.cfg`

```diff
 ui viewport: -
-statusbar: top
-sbs: -
+statusbar: off
+sbs: /.rockbox/wps/classic_statusbar.sbs
```

## Status bar — `.rockbox\wps\classic_statusbar.sbs`

```diff
 %Vi(-,0,24,-,-,1)
+%Fl(2,27-Adobe-Helvetica.fnt)
 # Conditional for showing volume as number or graphic
```

```diff
-%Vl(b,4,4,36,16,0)
+%Vl(b,4,0,44,24,2)
 %ar%bl
```

```diff
-%Vl(d,60,4,58,16,0)
+%Vl(d,60,0,58,24,2)
 %ac%?pv<%pv|%pv| %pv| %pv>
```

```diff
-%V(-82,4,62,16,0)
+%V(-82,0,62,24,2)
 %?cc<%?ca<%?St(time format)<%cH|%cI>:%cM|--:-->|>
```
