package main

import (
	"encoding/binary"
//	"flag"
	"sync"
	"os"
	"strings"
	"fmt"
	"log"
	"net"
	"runtime"
	"time"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	
	"github.com/gogf/gf/os/glog"
	"github.com/gogf/gf/os/gproc"
	"github.com/gogf/gf/text/gstr"
)

const ntpEpochOffset = 2208988800

type packet struct {
	Settings       uint8
	Stratum        uint8
	Poll           int8
	Precision      int8
	RootDelay      uint32
	RootDispersion uint32
	ReferenceID    uint32
	RefTimeSec     uint32
	RefTimeFrac    uint32
	OrigTimeSec    uint32
	OrigTimeFrac   uint32
	RxTimeSec      uint32
	RxTimeFrac     uint32
	TxTimeSec      uint32
	TxTimeFrac     uint32
}

func main() {
	var timeLayoutStr = "2006-01-02 15:04:05"
	ch := make(chan time.Time, 3)	
	defer close(ch)
	getremotetime(ch)
	ntime := <- ch
//	fmt.Println(ntime)
//	fmt.Println(<- ch)
//	fmt.Println(<- ch)
	ts := ntime.Format(timeLayoutStr) //time转string
	fmt.Println("------------")
	fmt.Println(ts)
	fmt.Println("------------")
	// 2021-08-29 15:53:35.922579627 +0800 CST
	UpdateSystemDate(ts)
}

type Config struct { 
	Host struct {
		Ip	string `yaml:"ip"`
	}
}

func (c *Config) getConf() *Config {
	yamlFile ,err := ioutil.ReadFile("./host.yaml")
	if err != nil {
		fmt.Println("yamlFile.Get err", err.Error())
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Println("Unmarshal: ", err.Error())
	}
	return c
}

func get_ntp_info(path string) string {
	var c Config
	conf := c.getConf()
	ntp_servers := conf.Host.Ip
	return ntp_servers
}


func udpGather(ip string, port string) bool {
	// 3 秒超时
	var result bool 
	address := net.JoinHostPort(ip ,port)
	fmt.Println(address)
	conn, err := net.DialTimeout("udp", address, 3 * time.Second)
//	addr, err := net.ResolveUDPAddr("udp", address)
//	conn, err := net.Dial("udp", addr)
	if err != nil {
		result = false
		// todo log handler
	} else {
		if conn != nil {
			result = true
			_ = conn.Close()
		} else {
			result = false
		}
	}
	fmt.Println(result)
	return result
}


//func getremotetime() time.Time {
func getremotetime(ch chan time.Time) {
//	var host string
	path := "host.yaml"
	// 182.92.12.11:123 是阿里的ntp服务器，可以换成其他域名的
	hosts := get_ntp_info(path)
//	flag.StringVar(&host, "e", host, "NTP host")
//	flag.Parse()
	var showtime time.Time
	var wg sync.WaitGroup
	fmt.Println(hosts)	
	for _, host := range strings.Split(hosts ,",") {
		hostStr := string(host) + ":123"
		wg.Add(1)
		go func(hostStr string) {
//			if udpGather(host ,"123") {
			addr, err := net.ResolveUDPAddr("udp", hostStr)
			if err != nil {
				fmt.Println("Can't resolve address: %v ", err)
				os.Exit(1)
			}

			conn, err := net.DialUDP("udp" ,nil ,addr)
			if err != nil {
				log.Fatalf("failed to connect: %v", err)
			}

			defer conn.Close()
			if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
				log.Fatalf("failed to set deadline: %v", err)
			}
			
			req := &packet{Settings: 0x1B}
			
			if err := binary.Write(conn, binary.BigEndian, req); err != nil {
				log.Fatalf("failed to send request: %v", err)
			}
			
			rsp := &packet{}
			if err := binary.Read(conn, binary.BigEndian, rsp); err != nil {
			//	log.Fatalf("failed to read server response: %v", err)
				fmt.Println("failed to read server response: %v", err)
			}
			
			secs := float64(rsp.TxTimeSec) - ntpEpochOffset
			nanos := (int64(rsp.TxTimeFrac) * 1e9) >> 32
			
			showtime = time.Unix(int64(secs), nanos)
			ch <- showtime
			wg.Done()
		}(hostStr)
		wg.Wait()
	}
//	return showtime
	return
}

func UpdateSystemDate(dateTime string) bool {
    system := runtime.GOOS
    switch system {
    case "windows":
	{
		_, err1 := gproc.ShellExec(`date  ` + gstr.Split(dateTime, " ")[0])
		_, err2 := gproc.ShellExec(`time  ` + gstr.Split(dateTime, " ")[1])
		if err1 != nil && err2 != nil {
			glog.Info("更新系统时间错误:请用管理员身份启动程序!")
			return false
		}
		return true
	}

    case "linux":
	{
		_, err1 := gproc.ShellExec(`date -s  "` + dateTime + `"`)
		if err1 != nil {
			glog.Info("更新系统时间错误:", err1.Error())
			return false
		}
		return true
	}
    case "darwin":
	{
		_, err1 := gproc.ShellExec(`date -s  "` + dateTime + `"`)
		if err1 != nil {
			glog.Info("更新系统时间错误:", err1.Error())
			return false
		}
		return true
		}
	}
	return false
}
