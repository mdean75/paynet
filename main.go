package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"paynet/pkg/iso8583"
	"strings"
	"sync"
	"time"
)

func main() {

	var mode string
	var ip string
	var port int
	var numConns int
	var workersPerConn int
	var duration int

	flag.StringVar(&mode, "m", "", "Operation mode: client|server")
	flag.StringVar(&mode, "mode", "", "Operation mode: client|server")

	flag.StringVar(&ip, "i", "127.0.0.1", "IP (client mode = dest ip, server mode = listen ip")
	flag.StringVar(&ip, "ip", "127.0.0.1", "IP (client mode = dest ip, server mode = listen ip")

	flag.IntVar(&port, "p", 3000, "Port (client mode = dest port, server mode = listen port")
	flag.IntVar(&port, "port", 3000, "Port (client mode = dest port, server mode = listen port")

	flag.IntVar(&numConns, "c", 1, "Number of client connections to established to backend")
	flag.IntVar(&numConns, "num-connections", 1, "Number of client connections to established to backend")

	flag.IntVar(&workersPerConn, "w", 10, "Number of workers per client connection")
	flag.IntVar(&workersPerConn, "workers-per-connection", 10, "Number of workers per client connection")

	flag.IntVar(&duration, "d", 10, "Duration in seconds to run the sim")
	flag.IntVar(&duration, "duration", 10, "Duration in seconds to run the sim")

	flag.Usage = usage

	flag.Parse()

	switch strings.ToUpper(mode) {
	case "CLIENT":
		runClient(ip, port, numConns, workersPerConn, duration)

	case "SERVER":
		runServer(ip, port)

	default:
		fmt.Println("mode not provided or invalid value")
		flag.Usage()
		os.Exit(1)
	}

	//fmt.Println("payload only:\t\t", len(iso8583.Encode(msg)))
	//fullPayload := iso8583.EncodeWithHeader(msg)
	//fmt.Println("payload with header:\t", len(fullPayload))
}

// var msgCache = make(map[int]struct{})
type msgCache struct {
	cache map[int]struct{}
	mu    *sync.Mutex
}

func newMsgCache() *msgCache {
	return &msgCache{
		cache: make(map[int]struct{}),
		mu:    new(sync.Mutex),
	}
}

func (m *msgCache) store(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[n] = struct{}{}
}

func (m *msgCache) delete(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, n)
}

func (m *msgCache) len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.cache)
}

func runClient(ip string, port, numConns, workersPerConn, duration int) {
	fmt.Println("client mode")
	var connCache []*net.TCPConn

	for i := 0; i < numConns; i++ {
		conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		})
		if err != nil {
			fmt.Println(err)
			return
		}
		connCache = append(connCache, conn)
	}

	defer func() {
		for _, conn := range connCache {
			conn.Close()
		}
	}() //conn.Close()

	cache := newMsgCache()

	var start time.Time
	wg := sync.WaitGroup{}
	//wg.Add(1)
	//exitTimer := time.Time{}
	exitCtx, exitCancel := context.WithCancel(context.Background())
	defer exitCancel()
	for _, c := range connCache {
		wg.Add(1)
		go func(conn *net.TCPConn) {
			defer wg.Done()
			time.Sleep(1 * time.Second) // delay until at least 1 has been added to cache
			for {
				if cache.len() == 0 {
					break
				}

				select {
				case <-exitCtx.Done():
					fmt.Println("exit ctx done")
					return
				default:
					buf := make([]byte, 133)
					_, err := conn.Read(buf)
					if err != nil {
						fmt.Println(err)
						return
					}

					//fmt.Printf("read %d bytes\n", n)

					readPayload := iso8583.Decode(buf[2:])
					//fmt.Printf("received response from %+v\n", readPayload.SystemTraceNo)
					cache.delete(int(readPayload.SystemTraceNo))
				}

			}
		}(c)
	}

	numgenFunc := nextNum()
	start = time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("Starting send with %d connections and %d workers\n", numConns, workersPerConn)

	for _, conn := range connCache {
		go func() {
			for i := 0; i < workersPerConn; i++ {
				wg.Add(1)
				go func(i int) {

					defer wg.Done()
					defer func() {
						fmt.Println("exiting client write loop num", i)
					}()

					for {
						select {
						case <-ctx.Done():
							return

						default:
							num := <-numgenFunc
							msg := genMsg(num)

							_, err := conn.Write(iso8583.EncodeWithHeader(msg))
							if err != nil {
								fmt.Println(err)
								return
							}
							cache.store(num)
							//fmt.Printf("wrote %d bytes to backend\n", n)
							time.Sleep(100 * time.Millisecond)

						}
					}
				}(i)
			}
		}()
	}

	timer := time.NewTicker(time.Duration(duration) * time.Second)

	var dur time.Duration
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			fmt.Println("exit monitor")
		}()
		<-timer.C
		cancel()
		for i := 0; i < 10; i++ {
			<-time.Tick(1 * time.Second)
			if cache.len() == 0 {
				break
			}
			fmt.Println("cache len:", cache.len())
		}
		exitCancel()
		for _, conn := range connCache {
			err := conn.CloseRead()
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("exit called")
		}

	}()

	wg.Wait()
	dur = time.Since(start)

	ttl := <-numgenFunc

	fmt.Printf("run completed in %.2fs, processed %d msgs\n", dur.Seconds(), ttl-1)
	if cache.len() != 0 {
		fmt.Println("lost tx in flight:", cache.len())
	}

	fmt.Printf("requests per sec: %.2f\n", float64(ttl-1)/dur.Seconds())
}

func runServer(ip string, port int) {
	fmt.Println("server mode listening on", ip, port)
	lis, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	defer lis.Close()

	for {
		conn, err := lis.AcceptTCP()
		if err != nil {
			fmt.Println(err)
			return
		}

		go handleConn(conn)
	}
}

func handleConn(conn *net.TCPConn) {
	defer conn.Close()
	rdr := bufio.NewReader(conn)
	for {
		header, err := rdr.Peek(2)
		if err != nil {
			fmt.Println(err)
			return
		}

		lenPayload := binary.BigEndian.Uint16(header)

		buf := make([]byte, lenPayload+2)

		_, err = rdr.Read(buf)
		if err != nil {
			var timeoutErr *net.OpError
			if !errors.As(err, &timeoutErr) {
				if err.Error() != "i/o timeout" {
					fmt.Println(err)
					return
				}
			}

		}

		//fmt.Printf("read %d bytes\n", n)

		go func() {
			time.Sleep(time.Duration(rand.Intn(5+1)) * time.Second)

			_, err = conn.Write(buf)
			if err != nil {
				fmt.Println("write error:", err)
				return
			}

			//fmt.Printf("wrote %d bytes to client\n", n)
		}()
	}
}

func genMsg(n int) iso8583.Request {
	msg := iso8583.Request{
		MessageType:     "0100",
		PrimaryBitmap:   "B23AE4012AE08034",
		SecondaryBitmap: "0000000004000020",
		ProcessingCode:  "072000",
		TransactionAmt:  "000000001860",
		TransmissionDt:  "0724200501",
		//SystemTraceNo:   "012190",
		SystemTraceNo:  uint64(n),
		LocalTime:      "160501",
		LocalDate:      "0724",
		ExpiryDate:     "2512",
		MerchantType:   "5411",
		AcquiringId:    "1042000314",
		RetrievalRefNo: "020600023074",
		CardNo:         "400000******0000",
		CardSeqNo:      "001",
		// Initialize other fields as needed
	}

	return msg
}

func nextNum() <-chan int {
	ch := make(chan int)
	go func() {
		i := 1
		defer close(ch)
		for {
			ch <- i // Yield data to the consumer
			i++
		}
	}()
	return ch
}

func usage() {
	fmt.Println(`Usage of paynet:
  -m --mode client|server
	Operation run mode
  -i --ip
	IP (client mode = dest ip, server mode = listen ip (default "127.0.0.1")
  -p --port
	Port (client mode = dest port, server mode = listen port (default 3000)
  -c --num-connections int
        Number of client connections to established to backend (default 1)
  -w --workers-per-connection int
        Number of workers per client connection (default 10)
  -d --duration int
        Duration in seconds to run the sim (default 10)`)
}
