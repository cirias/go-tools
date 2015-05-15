package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"sync"
)

const (
	PixivHost = "http://www.pixiv.net"
	AuthorUrl = PixivHost + "/member_illust.php"
	IllustUrl = PixivHost + "/member_illust.php"
)

var (
	memberId string
	user     string
	pass     string
)

func init() {
	flag.StringVar(&user, "user", "", "the user name to login")
	flag.StringVar(&pass, "pass", "", "the password of the login user")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <id>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	memberId = flag.Arg(0)
}

func main() {
	//id := "984442"  リンファミ
	//user := "fhc023"
	//pass := "f32645664"

	queue := make(chan Job, 200)
	wc := make(chan struct{}, 10)
	wg := new(sync.WaitGroup)

	wait := func() {
		go func() {
			wg.Wait()
			close(queue)
		}()
	}

	addJob := func(j Job) {
		wg.Add(1)
		queue <- j
	}

	cookieJar, _ := cookiejar.New(nil)
	c := Client{
		&http.Client{
			Jar: cookieJar,
		},
		reflect.ValueOf(addJob),
	}

	c.login(user, pass)

	addJob(Job{Route: Route{getUrl(), "GetAuthor"}})

	once := new(sync.Once)

	for j := range queue {
		once.Do(wait)
		go func(j Job) {
			wc <- struct{}{}
			defer func() {
				<-wc
			}()
			defer wg.Done()

			log.Println(j)
			reflect.ValueOf(&c).MethodByName(j.Route.Method).
				Call([]reflect.Value{reflect.ValueOf(j)})
		}(j)
	}

	log.Printf("ok\n")
}

func getUrl() string {
	u, _ := url.Parse(AuthorUrl)
	q := u.Query()
	q.Set("id", memberId)
	u.RawQuery = q.Encode()
	return u.String()
}
