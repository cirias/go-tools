package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/cenkalti/backoff"
	"gopkg.in/redis.v2"
	"io"
	"log"
	"os"
	"time"
)

var (
	redisKey   string
	redisAddr  string
	maxSetSize int64
)

func init() {
	flag.StringVar(&redisAddr, "redis-addr", ":6379", "the `address:port` of the redis server")
	flag.Int64Var(&maxSetSize, "max-set-size", -1, "the max size of redis set")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <redis-set-key>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	redisKey = flag.Arg(0)
}

func main() {
	var total int
	tStart := time.Now()
	client := redis.NewTCPClient(&redis.Options{
		Addr: redisAddr,
	})

	r := csv.NewReader(os.Stdin)
	header, err := r.Read()
	if err != nil {
		log.Fatalln(err)
	}

	for i := 0; ; i++ {
		if i%5000 == 0 {
			log.Printf("Load %d rows\n", i)
			wait(client)
		}

		row, err := r.Read()
		if err != nil {
			if err != io.EOF {
				log.Fatalln(err)
			}

			total = i
			break
		}

		m := make(map[string]string)

		for i, h := range header {
			m[h] = row[i]
		}

		bytes, err := json.Marshal(m)
		str := string(bytes)
		if err := save(client, str); err != nil {
			log.Fatalln(err)
		}
	}

	tEnd := time.Now()
	log.Printf("Complete, took %v to load %d rows.\n", tEnd.Sub(tStart), total)
}

func save(client *redis.Client, s string) error {
	b := backoff.NewExponentialBackOff()
	ticker := backoff.NewTicker(b)

	var err error

	for range ticker.C {
		if err = client.SAdd(redisKey, s).Err(); err != nil {
			log.Println(err, "will retry...")
			continue
		}

		ticker.Stop()
		break
	}

	return err
}

func wait(client *redis.Client) (err error) {
	if maxSetSize < 0 {
		return
	}

	var num int64
	bo := backoff.NewExponentialBackOff()
	tk := backoff.NewTicker(bo)

	for range tk.C {
		b := backoff.NewExponentialBackOff()
		ticker := backoff.NewTicker(b)

		for range ticker.C {
			num, err = client.SCard(redisKey).Result()
			if err != nil {
				log.Println(err, "will retry...")
				continue
			}

			ticker.Stop()
			break
		}

		if err != nil {
			return err
		}

		if num < maxSetSize {
			break
		}

		log.Println("Reach max set size, waiting...")
	}

	return
}
