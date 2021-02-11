package main

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "192.168.1.102:1234")
	if err != nil {
		logrus.Errorf("connection failed. err=%v", err)
		return
	}
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			fmt.Fprintln(conn, text)
		}
	}()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		fmt.Fprintln(os.Stdout, text)
	}
}
