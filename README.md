# LinuxMetrics

## Description:
This is Go program that runs a bunch of commands specified in config.json file. \
Exactly commands are BCC's tools for eBPF-based linux analysis. Using them we can \
extract many metrics and details about inner communication and processes running on \
linux.\
    This repository presents nineteen of those tools. Each one allows us to get insights \
related with traffic and networks in considered linux instance. included metrics are:
1. gethostlatency traces host name lookup calls getaddrinfo(), gethostbyname(), and gethostbyname2()
2. bindsnoop traces the kernel function performing socket binding
3. netqtop traces the kernel functions performing packet transmit (xmit_one) and packet receive
4. sofdsnoop traces FDs passed through unix sockets
5. solisten traces the kernel function called when a program wants to listenfor TCP connections
6. sslsniff traces the write/send and read/recv functions of OpenSSL
7. tcpaccept traces the kernel function accepting TCP socket connections
8. tcpconnect traces the kernel function performing active TCP connections
9. tcpconnlat traces the kernel function performing active TCP connections
10. tcpdrop prints details of TCP packets or segments that were dropped by the kernel
11. tcplife summarizes TCP sessions that open and close while tracing.
12. tcpretrans traces the kernel TCP retransmit function
13. tcprtt traces TCP RTT(round-trip time) to analyze the quality of network
14. tcpstates prints TCP state change information
15. tcpsubnet summarizes throughput for IPv4
16. tcpsynbl shows the TCP SYN backlog size during SYN arrival
17. tcptop summarizes throughput by host and port
18. tcptracer traces the kernel function performing TCP connections
19. tcpcong traces linux kernel's tcp congestion control status change functions

## Requirements:
Installed Go compiler to compile solution.\
Installed bcc for program runtime.\
    - on AWS: https://repost.aws/knowledge-center/ec2-linux-tools-performance-bottlenecks?\
    - locally: https://github.com/iovisor/bcc/blob/master/INSTALL.md\

## Setup:
Consider only production directory.\
If kafka is working and eligible address is specified in config.json file then\
in the same file You have to alter variable kafka_connected to **true**.\
Ctrl+C to close app.

#### Locally(on linux):
1. Enter kafka directory.
2. Establish kafka instance: `docker-compose up`
3. Update go: `go mod tidy`
4. Build solution: `go build run.go`
5. Run solution with admin privilages: `sudo ./run`

#### AWS EC2:
1. Copy files: config.json, go.mod, run.go
2. Update go: `go mod tidy`
3. Build solution: `go build run.go`
4. Run solution with admin privilages: `sudo ./run`
\
or build solution locally and then:
1. copy files: config.json, run
2. Run solution with admin privilages: `sudo ./run`
