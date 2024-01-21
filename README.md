# goyoubbs

goyoubbs is an open source web forum built on Golang, fasthttp and leveldb.

Demo online https://youbbs.org/

## Usage

### Quick start

Go to [Actions page](https://github.com/ego008/goyoubbs/actions) and download the latest binary for your server OS.

Use `goyoubbs-linux-amd64` file for example on linux amd64 system.

```shell
$ unzip goyoubbs-linux-amd64.zip
Archive:  goyoubbs-linux-amd64.zip
  inflating: goyoubbs-linux-amd64    
$ chmod +x goyoubbs-linux-amd64
$ ./goyoubbs-linux-amd64 
2024/01/21 14:31:25 SelfHash: 8nJzaExmKM4
2024/01/21 14:31:25 UploadDir from "upload"
2024/01/21 14:31:25 Serving sdb from directory "localdb"
2024/01/21 14:31:25 TCP address to listen to ":8080"
```

Open the URL `http://127.0.0.1:8080` in your browser.

### Build for yourself
Require go 1.19+

Download source code and build.

```
git clone https://github.com/ego008/goyoubbs
cd goyoubbs
go build .
./goyoubbs
```

## Contributing

Fork me && Pull requests

## License

MIT License