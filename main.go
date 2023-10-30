package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

// tcp 报文前20个是报文头，后面的才是 ICMP 的内容。
// ICMP：组建 ICMP 首部（8 字节） + 我们要传输的内容
// ICMP 首部：type、code、校验和、ID、序号，1 1 2 2 2
// 回显应答：type = 0，code = 0
// 回显请求：type = 8, code = 0

var (
	timeout int64 // 耗时
	size    int   // 大小
	count   int   // 请求次数
	typ     uint8 = 8
	code    uint8 = 0
)

// ICMP 序号不能乱
type ICMP struct {
	Type        uint8
	Code        uint8
	CheckSum    uint16 // 校验和
	ID          uint16 // ID
	SequenceNum uint16 // 序号
}

func main() {
	log.SetFlags(log.Llongfile)
	GetCommandArgs()

	// 获取目标 IP
	desIP := os.Args[len(os.Args)-1]
	//fmt.Println(desIP)
	// 构建连接
	conn, err := net.DialTimeout("ip:icmp", desIP, time.Duration(timeout)*time.Millisecond)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer conn.Close()
	// 远程地址
	remoteaddr := conn.RemoteAddr()
	fmt.Printf("正在 Ping %s [%s] 具有 %d 字节的数据:\n", desIP, remoteaddr, size)
	for i := 0; i < count; i++ {
		// 构建请求
		icmp := &ICMP{
			Type:        typ,
			Code:        code,
			CheckSum:    uint16(0),
			ID:          uint16(i),
			SequenceNum: uint16(i),
		}

		// 将请求转为二进制流
		var buffer bytes.Buffer
		binary.Write(&buffer, binary.BigEndian, icmp)
		// 请求的数据
		data := make([]byte, size)
		// 将请求数据写到 icmp 报文头后
		buffer.Write(data)
		data = buffer.Bytes()
		// ICMP 请求签名（校验和）：相邻两位拼接到一起，拼接成两个字节的数
		checkSum := checkSum(data)
		// 签名赋值到 data 里
		data[2] = byte(checkSum >> 8)
		data[3] = byte(checkSum)
		startTime := time.Now()

		// 设置超时时间
		conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))

		// 将 data 写入连接中，
		n, err := conn.Write(data)
		if err != nil {
			log.Println(err)
			continue
		}

		// 接收响应
		buf := make([]byte, 1024)
		n, err = conn.Read(buf)
		//fmt.Println(data)
		if err != nil {
			log.Println(err)
			continue
		}
		//fmt.Println(n, err) // data：64，ip首部：20，icmp：8个 = 92 个
		// 打印信息
		fmt.Printf("来自 %d.%d.%d.%d 的回复：字节=%d 时间=%d TTL=%d\n", buf[12], buf[13], buf[14], buf[15], n-28, time.Since(startTime).Milliseconds(), buf[8])
		time.Sleep(time.Second)
	}
}

// 求校验和
func checkSum(data []byte) uint16 {
	// 第一步：两两拼接并求和
	length := len(data)
	index := 0
	var sum uint32
	for length > 1 {
		// 拼接且求和
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		length -= 2
		index += 2
	}
	// 奇数情况，还剩下一个，直接求和过去
	if length == 1 {
		sum += uint32(data[index])
	}

	// 第二部：高 16 位，低 16 位 相加，直至高 16 位为 0
	hi := sum >> 16
	for hi != 0 {
		sum = hi + uint32(uint16(sum))
		hi = sum >> 16
	}
	// 返回 sum 值 取反
	return uint16(^sum)
}

// GetCommandArgs 命令行参数
func GetCommandArgs() {
	flag.Int64Var(&timeout, "w", 1000, "请求超时时间")
	flag.IntVar(&size, "l", 32, "发送字节数")
	flag.IntVar(&count, "n", 4, "请求次数")
	flag.Parse()
}
