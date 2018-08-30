package main

import (
	"bytes"
	"encoding/binary"
	"log"
)

// 把 int64的数据 转为 byte 数组
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
