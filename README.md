# A microcode initramfs image generator.

`ucode-image-gen` is a tool to generate ucode boot images.
The kernel uses such images at the early boot stages to fix CPU bugs.

Some platforms do not have a standard tool to build ucode images (e.g. https://bugzilla.redhat.com/show_bug.cgi?id=1829601), and `ucode-image-gen` fills this gap.


To enable such early microcode loading make sure the following kernel features are enabled:
```
CONFIG_BLK_DEV_INITRD=Y
CONFIG_MICROCODE=y
CONFIG_MICROCODE_INTEL=Y
CONFIG_MICROCODE_AMD=y
```

Then generate the ucode image. One can either generate a universal or host-specific image.

By default `ucode-image-gen` generates a host-specific image that contains only ucode for the current host’s CPU.

Adding `-universal` flag makes the tool generate a universal image that contains all ucode files. Thus the same file can be used by numerous hardware platforms.

Once the image is generated, please configure the host bootloader. The idea here is to load the ucode before the main initramfs image.


#### GRUB
```
$ cat /boot/grub/grub.cfg
...
echo 'Loading initial ramdisk'
initrd /boot/ucode.img /boot/initramfs-linux.img
...
```

#### systemd-boot
```
$ cat /boot/loader/entries/entry.conf
linux   /vmlinuz-linux
initrd  /ucode.img
initrd  /initramfs-linux.img
...
```

#### EFISTUB
append an `initrd=` option as `initrd=\ucode.img initrd=\initramfs-linux.img`


#### rEFInd:
```
$ cat /boot/refind_linux.conf
"Boot"     "root=/dev/xxx rw add_efi_memmap initrd=boot\ucode.img initrd=boot\initramfs-%v.img"
```

#### Syslinux:
```
$ cat /boot/syslinux/syslinux.cfg
LABEL linux
...
   LINUX ../vmlinuz-linux
   INITRD ../ucode.img,../initramfs-linux.img
...
```

### Compile

`ucode-image-gen` is a Golang application and uses the standard way to compile: `go build`.

By default, `ucode-image-gen` uses the following paths to lookup ucode files:

```
hostUcodeDirIntel = "/lib/firmware/intel-ucode"
hostUcodeDirAmd   = "/lib/firmware/amd-ucode"
```

But some platforms use different files location. For example, CentOS7 stores Intel ucode at `/usr/share/microcode_ctl/ucode_with_caveats/intel/intel-ucode/`. To account it please specify the build flag like this: `go build --ldflags="-X main.hostUcodeDirIntel=/usr/share/microcode_ctl/ucode_with_caveats/intel/intel-ucode/"`