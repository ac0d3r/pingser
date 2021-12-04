# pingser

Use `pingser` to create client and server based on ICMP Protocol to send and receive custom message content.

## examples

source code: [./examples](./examples)

![](https://user-images.githubusercontent.com/26270009/144586946-fd8bdb6a-34f7-4db6-a22c-9ba89901d0fa.png)


Usage:

- 需要准备一台公网 IP 的 linux 主机运行server

-  修改 [client.go](./examples/client/client.go) 文件中 `pingser.NewClient("xxx.xxx.xxx.xxx")` 的 `xxx` 地址 & 再本地运行

-  可选）禁用系统默认ping: `echo 1 > /proc/sys/net/ipv4/icmp_echo_ignore_all`
