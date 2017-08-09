## 快速入门

```
kdt client: ./kdt server -l :4000
kdt server: ./kdt client --remoteaddr 127.0.0.1:4000 bigfile
```
上面的命令会在server端监听4000端口，client端通过连接server的4000端口传送文件名为bigfile的文件到server端  


## Usage

```
$ ./kdt -h
$ ./kdt client -h
$ ./kdt server -h

OPTIONS:
   --key value                   预共享秘钥， 服务端，客户端需要保持一致
   --crypt value                 支持的加密方式，aes, aes-128, aes-192, salsa20, blowfish, twofish, cast5, 3des, tea, xtea, xor, none (default: "none")
   --mode value                  profiles: fast3, fast2, fast, normal (default: "fast")， 越快对带宽消耗越高
   --mtu value                   set maximum transmission unit for UDP packets (default: 1350)
   --datashard value             set reed-solomon erasure coding - datashard (default: 10)  RS-Code参数,如果线路丢包率少，可以设置成0， 服务端，客户端需要保持一致
   --parityshard value           set reed-solomon erasure coding - parityshard (default: 3) RS-Code参数,如果线路丢包率少，可以设置成0， 服务端，客户端需要保持一致
   --dscp value                  set DSCP(6bit) (default: 0)  差分服务代码点
   --comp                        enable compression, 是否启用压缩
   --nodelay value               (default: 0) 是否启用 nodelay模式，0不启用；1启用
   --interval value              (default: 40) 协议内部工作的 interval，单位毫秒，比如 10ms或者 20ms
   --resend value                (default: 0) 快速重传模式，默认0关闭，可以设置2（2次ACK跨越将会直接重传）
   --nc value                    (default: 0) 是否关闭流控，默认是0代表不关闭，1代表关闭
   --log value                   specify a log file to output, default goes to stderr 指定log文件
   -c value                      config from json file, which will override the command from shell 指定配置文件
   --transfer_id value           transfer id  传输id
   --remoteaddr value, -r value  kcp server address  server端地址
   --conn value                  set num of UDP connections to server (default: 1) udp连接个数
   --autoexpire value            set auto expiration time(in seconds) for a single UDP connection, 0 to disable (default: 0) 设置连接超时时间
   --sndwnd value                set send window size(num of packets) (default: 1024) 发送窗口大小
   --rcvwnd value                set receive window size(num of packets) (default: 1024) 接收窗口大小
   --listen value, -l value      kcp server listen address (default: ":29900") 监听端口
   --root value                  root directory (default: ".")  根目录

```


## Basic Tuning Guide
提高吞吐量
```
  在client端增加rcwnd， 在服务端增加sndwnd
```
改善延迟
```
  改变传输模式. fast3 > fast2 > fast > normal > default
```

## 相等参数
下面参数需要在server和client端设置相同的参数
  -key
  -crypt
  -nocomp
  -datashard
  -parityshard

## 建议设置
网络比较好的情况
```
kdt client: ./kdt server -l :4000  --datashard 0 --parityshard 0
kdt server: ./kdt client --remoteaddr 127.0.0.1:4000  --datashard 0 --parityshard 0 bigfile
```
网络有丢包的情况下，datashard和parityshard恢复为默认值， 如果速度不够增大些sndwnd和rcvwnd
```
kdt client: ./kdt server -l :4000  --rcvwnd 8192
kdt server: ./kdt client --remoteaddr 127.0.0.1:4000  --sndwnd 8192  bigfile
```
或者
```
kdt client: ./kdt server -l :4000 --datashard 0 --parityshard 0 --rcvwnd 8192
kdt server: ./kdt client --remoteaddr 127.0.0.1:4000  --datashard 0 --parityshard 0  --sndwnd 8192  bigfile
```

