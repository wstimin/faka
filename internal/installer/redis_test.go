package installer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// A minimal RESP peer verifies authentication and SELECT without installing a local Redis service.
func TestRedisCheckValidatesBothDatabasesAndHidesCredentials(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	var mu sync.Mutex
	seen := map[string]bool{}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				reader := bufio.NewReader(conn)
				authorized, database := false, "0"
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					count, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "*")))
					if count < 1 || count > 20 {
						return
					}
					args := make([]string, count)
					for i := range args {
						line, err = reader.ReadString('\n')
						if err != nil {
							return
						}
						length, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "$")))
						if length < 0 || length > 4096 {
							return
						}
						data := make([]byte, length+2)
						if _, err = io.ReadFull(reader, data); err != nil {
							return
						}
						args[i] = string(data[:length])
					}
					reply := "+OK\r\n"
					switch strings.ToUpper(args[0]) {
					case "HELLO":
						reply = "-ERR unknown command 'hello'\r\n"
					case "AUTH":
						authorized = args[len(args)-1] == "private-redis-password"
						if !authorized {
							reply = "-WRONGPASS invalid password\r\n"
						}
					case "SELECT":
						database = args[1]
					case "PING":
						if !authorized {
							reply = "-NOAUTH Authentication required\r\n"
						} else {
							mu.Lock()
							seen[database] = true
							mu.Unlock()
							reply = "+PONG\r\n"
						}
					}
					if _, err = fmt.Fprint(conn, reply); err != nil {
						return
					}
				}
			}()
		}
	}()
	_, portText, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portText)
	in := Input{RedisHost: "127.0.0.1", RedisPort: port, RedisPassword: "private-redis-password"}
	if err := checkRedis(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	both := seen["0"] && seen["1"]
	mu.Unlock()
	if !both {
		t.Fatal("both cache and queue databases must be verified")
	}
	in.RedisPassword = "wrong-private-password"
	err = checkRedis(context.Background(), in)
	if err == nil || err.Error() != "setup.redis_failed" || strings.Contains(err.Error(), in.RedisPassword) {
		t.Fatal("authentication failure was not safely reported", err)
	}
}
