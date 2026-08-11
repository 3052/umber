# status bar

## `.rockbox\themes\cabbiev2.cfg`

```diff
 ui viewport: -
-statusbar: top
-sbs: -
+statusbar: off
+sbs: /.rockbox/wps/classic_statusbar.sbs
```

## `.rockbox\wps\classic_statusbar.sbs`

Fonts pack (extract `.fnt` files to `.rockbox\fonts\`):
https://download.rockbox.org/daily/fonts/rockbox-fonts.zip

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
