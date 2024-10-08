## What is

It is command line for zstd compression and decompression using golang.

I looked around and found nowhere. I need a cross platform to do this to utilize multicore CPU. zstd does not. pzstd is not available and hard to compile on my platform. 

Thus a quick dirty code appear in here. To my dismay the encoder only support 2 threads in stream mode? Well I feel it is a bit faster than run zstd thus I will keep using it lol.

look at this benchmark :P

```
root@work-mbp2018 /m/p/tmp [SIGINT]# time gozstd /mnt/portdata/port-os/u24/base/001-u2404-base-x86_64.xzm > test.zstd

________________________________________________________
Executed in    4.04 secs    fish           external
   usr time    3.17 secs    0.77 millis    3.17 secs
   sys time    3.36 secs  436.19 millis    2.92 secs

root@work-mbp2018 /m/p/tmp# time zstd -3 -c /mnt/portdata/port-os/u24/base/001-u2404-base-x86_64.xzm > test.zstd

________________________________________________________
Executed in    6.01 secs    fish           external
   usr time    5.19 secs    1.31 millis    5.19 secs
   sys time    3.57 secs  682.89 millis    2.88 secs

```

Yay! winning!

I do not intend to make the options completely same as zstd but it works for my goal now.

The block mode does support multicore threading though. If you use compression level > 9 (eg. 15) then use it will significantly faster than stream mode. 

## Build and run

```
env CGO_ENABLED=0 go build -trimpath -ldflags="-X main.version=v0.3 -extldflags=-static -w -s" .

(main)> ./gozstd -h
Version: v0.3
Build time: 
  -T int
        Number of threads (default: library default)
  -c    Write output to stdout
  -d    Decompress instead of compress
  -l int
        Set compression level (1-19, default: 3. Good tradoff is 9) (default 3)
  -o string
        Output file (default: stdout)

```

As it write to stdout by default and read from stdin if no file provided you can use it in pipe. Something like

```
tar czf - somedir | gozstd > outputfile.tar.zstd
```

If you need better compression add option `-l 9`; I found 9 is pretty good in terms of speed and size. 

## Download

To lock it off, I will update a linux and windows binary :P. You can hit to release section and download if you do not want to build it yourself.
