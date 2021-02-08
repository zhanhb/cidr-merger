# cidr-merger
A simple utility to merge ip/ip cidr/ip range, support IPv4/IPv6

```
$ cidr-merger --help
Usage: cidr-merger [OPTION]... [FILE]...
Write sorted result to standard output.

Options:
     --batch               batch mode (default), read file content into memory,
                           then write to the specified file
     --cidr                print as ip/cidr (default if not console mode)
 -c, --console             console mode, all input output files are ignored,
                           write to stdout immediately
     --empty-policy=value  indicate how to process empty input file
                             ignore(default): process as if it is not empty
                             skip: don't create output file
                             error: raise an error and exit
 -e, --error-if-empty      same as --empty-policy=error
 -h, --help                show this help menu
     --ignore-empty        same as --empty-policy=ignore
 -k, --skip-empty          same as --empty-policy=skip
     --merge               sort and merge input values (default)
     --original-order      output as the order of input, without merging
 -o, --output=file         output values to <file>, if multiple output files
                           specified, the count should be same as input files,
                           and will be processed respectively
 -r, --range               print as ip ranges
     --simple              output as single ip as possible (default)
                             ie. 192.168.1.2/32 -> 192.168.1.2
                                 192.168.1.2-192.168.1.2 -> 192.168.1.2
 -s, --standard            don't output as single ip
 -v, --version             show version info
```

Sample Usage:
```shell script
$ echo '1.0.0.1-223.255.255.254' | cidr-merger
> 1.0.0.1
  1.0.0.2/31
  1.0.0.4/30
  1.0.0.8/29
  ......
  1.128.0.0/9
  2.0.0.0/7
  4.0.0.0/6
  8.0.0.0/5
  16.0.0.0/4
  32.0.0.0/3
  64.0.0.0/2
  128.0.0.0/2
  192.0.0.0/4
  208.0.0.0/5
  216.0.0.0/6
  220.0.0.0/7
  222.0.0.0/8
  223.0.0.0/9
  ......
  223.255.255.240/29
  223.255.255.248/30
  223.255.255.252/31
  223.255.255.254
$ echo '1.1.1.0' > a; \
    echo '1.1.1.1' > b; \
    echo '1.1.1.2/31' > c; \
    echo '1.1.1.3-1.1.1.7' > d; \
    cidr-merger -o merge a b c d; \
    cat merge; \
    rm a b c d merge
> 1.1.1.0/29
$ wget -O- "https://ftp.apnic.net/stats/apnic/`TZ=UTC date +%Y`/delegated-apnic-`TZ=UTC+24 date +%Y%m%d`.gz" | \
    gzip -d | awk -F\| '!/^\s*(#.*)?$/&&/CN\|ipv4/{print $4 "/" 32-log($5)/log(2)}' | \
    cidr-merger -eo/etc/chinadns_chnroute.txt # update ip on router
$ #              ^ e: means error if input is empty
$ echo 'fe80::/10' | cidr-merger -r
> fe80::-febf:ffff:ffff:ffff:ffff:ffff:ffff:ffff
$ echo '1.1.1.0' > a; echo '1.1.1.1' | cidr-merger - a; rm a
$ #                                                ^ -: means standard input
> 1.1.1.0/31
```

Difference between standard and simple(default)
```shell script
$ echo '1.1.1.1/32' | cidr-merger
> 1.1.1.1
$ echo '1.1.1.1/32' | cidr-merger -s
> 1.1.1.1/32
$ echo '1.1.1.1/32' | cidr-merger -r
> 1.1.1.1
$ echo '1.1.1.1/32' | cidr-merger -rs
> 1.1.1.1-1.1.1.1
```

Difference about empty policy
```shell script
$ cidr-merger -o txt /dev/null # an empty file named `txt` is created.
$ cidr-merger -ko txt /dev/null # no file is created, and this program exit with code zero
$ #            ^ same as `cat /dev/null | cidr-merger --skip-empty --output txt`
$ cidr-merger -eo txt /dev/null # no file is created, and this program exit with code non zero
$ #            ^ same as `cat /dev/null | cidr-merger --error-if-empty --output txt`
$ # option `-e` might be useful when download file from internet and then write to a file

$ # There is no difference if you redirect output to a file such as following
$ cat /dev/null | cidr-merger -e > txt
  # file `txt` is created, but this program exit with code non zero
```
