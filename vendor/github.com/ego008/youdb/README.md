# youdb
A Bolt wrapper that allows easy store hash, zset data.

## Install

```
go get -u github.com/ego008/youdb
```

## Example

```
package main

import (
	"fmt"
	"github.com/ego008/youdb"
)

func main() {
	db, err := youdb.Open("my.db")
	if err != nil {
		fmt.Println("open err", err)
		return
	}
	defer db.Close()

	// hash

	db.Hset("mytest", []byte("key1"), []byte("value1"))

	rs := db.Hget("mytest", []byte("key1"))
	if rs.State == "ok" {
		fmt.Println(rs.Data[0], rs.String())
	} else {
		fmt.Println("key not found")
	}

	data := [][]byte{}
	data = append(data, []byte("k1"), []byte("12987887762987"))
	data = append(data, []byte("k2"), []byte("abc"))
	data = append(data, []byte("k3"), []byte("qwasww"))
	data = append(data, []byte("k4"), []byte("444"))
	data = append(data, []byte("k5"), []byte("555"))
	data = append(data, []byte("k6"), []byte("aaaa556"))
	data = append(data, []byte("k7"), []byte("77777"))
	data = append(data, []byte("k8"), []byte("88888"))

	db.Hmset("myhmset", data...)

	rs = db.Hmget("myhmset", [][]byte{[]byte("k1"), []byte("k2"), []byte("k3"), []byte("k4")})
	if rs.State == "ok" {
		for _, v := range rs.List() {
			fmt.Println(v.Key.String(), v.Value.String())
		}
	}

	fmt.Println(db.Hincr("num", []byte("k1"), 2))

	k, _ := youdb.DS2b("19822112")
	v := uint64(121211121212233)
	db.Hset("mytestnum", k, youdb.I2b(v))
	r := db.Hget("mytestnum", k)
	if r.State == "ok" {
		fmt.Println(r.Int64(), r.Data[0].Int64(), string(r.Data[0]), youdb.B2i(r.Data[0]))
	} else {
		fmt.Println(r.State, r.Int64())
	}

	// zet

	db.Zset("mytest", []byte("key1"), 100)

	rs2 := db.Zget("mytest", []byte("key1"))
	if rs2.State == "ok" {
		fmt.Println(rs2.Int64())
	}

	fmt.Println(db.Zincr("num", []byte("k1"), 2))
}
```

## Who's using youdb?

- [goyoubbs](https://www.youbbs.org/) - A forum/discussion software written in Go.


