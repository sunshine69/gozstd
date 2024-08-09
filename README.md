It is command line for zstd compression and decompression using golang.

I looked around and found nowhere. I need a cross platform to do this to utilize multicore CPU. zstd does not. pzstd is not available and hard to compile on my platform. 

Thus a quick dirty code appear in here.

I do not intend to make the options compeltedly same as zstd but it works for my goal now.
