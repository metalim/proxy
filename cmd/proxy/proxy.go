package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const addr1 = "localhost:25565"
const TICK_PERIOD = time.Second * 5

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: proxy <addr2>")
		os.Exit(1)
	}

	addr2 := os.Args[1]
	log.Println("Starting proxy server", addr1, addr2)
	ls, err := net.Listen("tcp", addr1)
	catch(err)

	var id int
	for {
		conn, err := ls.Accept()
		catch(err)
		id++
		log.Println(id, "Connected from", conn.RemoteAddr())
		go handle(conn, id, addr2)
	}
}

func catch(err error) {
	if err != nil {
		panic(err)
	}
}

func handle(conn net.Conn, id int, addr2 string) {
	defer conn.Close()

	conn2, err := net.Dial("tcp", addr2)
	catch(err)
	defer conn2.Close()

	var up, down, lastup, lastdown int64

	exit := make(chan struct{})

	go func() {
		ticker := time.NewTicker(TICK_PERIOD)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Printf("%d Up: %d bytes, %d bytes/s, Down: %d bytes, %d bytes/s", id,
					up, time.Duration(up-lastup)*time.Second/TICK_PERIOD,
					down, time.Duration(down-lastdown)*time.Second/TICK_PERIOD,
				)
				lastup = up
				lastdown = down
			case <-exit:
				return
			}
		}
	}()

	go func() {
		var err error
		down, err = copy(conn, conn2, &down)
		log.Println(id, "Downloaded", down, "bytes", err)
	}()
	up, err = copy(conn2, conn, &up)
	log.Println(id, "Uploaded", up, "bytes", err)

	close(exit)

	log.Println(id, "disconnected", conn.RemoteAddr(), conn2.RemoteAddr())
	fmt.Println()
}

var errInvalidWrite = errors.New("invalid write result")

func copy(dst io.Writer, src io.Reader, track *int64) (written int64, err error) {
	size := 32 * 1024
	buf := make([]byte, size)

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			*track = written
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
