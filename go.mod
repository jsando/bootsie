module github.com/jsando/fatimg

go 1.21

toolchain go1.22.5

require (
	github.com/diskfs/go-diskfs v1.4.1
	github.com/dustin/go-humanize v1.0.1
	github.com/klauspost/pgzip v1.2.6
)

replace github.com/diskfs/go-diskfs v1.4.1 => github.com/jsando/go-diskfs v0.0.0-20240831005111-5997b71b4caf

require (
	github.com/djherbis/times v1.6.0 // indirect
	github.com/elliotwutingfeng/asciiset v0.0.0-20230602022725-51bbb787efab // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/klauspost/compress v1.17.5 // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pkg/xattr v0.4.9 // indirect
	github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	golang.org/x/sys v0.6.0 // indirect
)
