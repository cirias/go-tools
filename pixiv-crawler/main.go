package main

import (
	"github.com/PuerkitoBio/goquery"
	"os"
	"path"
	"strconv"
	"sync"
	//"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type Actor struct {
	id   string
	name string
}

type Image struct {
	id    string
	name  string
	url   string
	actor *Actor
}

const (
	Pixiv        = "http://www.pixiv.net"
	MemberIllust = Pixiv + "/member_illust.php"
)

func main() {
	id := "12345"
	user := ""
	pass := ""
	cookieJar, _ := cookiejar.New(nil)

	client := &http.Client{
		Jar: cookieJar,
	}

	login(client, user, pass)

	done := make(chan struct{})

	chanlisturl := generateIllustListUrl(done, id)

	chanillusturl := getIllustUrl(client, done, chanlisturl)

	chanimageurls, chanmultiurl := getSingleImageUrlAndMultiPreviewUrl(client, chanillusturl)

	chanimageurlm := getImageUrl(client, chanmultiurl)

	chanimageurl := merge(chanimageurls, chanimageurlm)

	download(client, chanimageurl)

	//log.Print(cookieJar.Cookies(url))

	//body, err := ioutil.ReadAll(res.Body)
	//log.Println(string(body))

	//_, err = goquery.NewDocumentFromResponse(res)
	//if err != nil {
	//log.Fatalln(err)
	//}

	log.Printf("ok\n")
}

func merge(cs ...<-chan string) <-chan string {
	var wg sync.WaitGroup
	out := make(chan string)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan string) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func duplicate(in <-chan interface{}, n int) []chan interface{} {
	outs := make([]chan interface{}, n)
	go func() {
		for _, out := range outs {
			defer close(out)
		}

		// Send each value from in to each out
		for i := range in {
			for _, out := range outs {
				out <- i
			}
		}
	}()
	return outs
}

func dispatch(in <-chan interface{}, do func(interface{}) interface{}) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for i := range in {
			out <- do(i)
		}
	}()
	return out
}

func login(client *http.Client, id string, pass string) {
	res, err := client.PostForm("https://www.secure.pixiv.net/login.php",
		url.Values{
			"mode":      {"login"},
			"return_to": {"/"},
			"pixiv_id":  {id},
			"pass":      {pass},
			"skip":      {"1"},
		})
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()
}

func generateIllustListUrl(done <-chan struct{}, id string) <-chan string {
	out := make(chan string, 3)
	go func() {
		defer close(out)
		for i := 1; true; i++ {
			select {
			case <-done:
				break
			default:
				u, err := url.Parse(MemberIllust)
				if err != nil {
					log.Fatalln(err)
				}

				q := u.Query()
				q.Set("id", id)
				q.Set("p", strconv.Itoa(i))
				u.RawQuery = q.Encode()
				out <- u.String()
			}
		}
	}()
	return out
}

func builder(client *http.Client, done chan<- struct{}) func(<-chan interface{}) <-chan interface{} {

}

func getIllustUrl(client *http.Client, done chan<- struct{}, in <-chan string) <-chan string {
	out := make(chan string, 20)
	go func() {
		defer close(out)
		for u := range in {
			res, err := client.Get(u)
			if err != nil {
				log.Fatalln(err)
			}
			defer res.Body.Close()

			dom, err := goquery.NewDocumentFromResponse(res)
			if err != nil {
				log.Fatalln(err)
			}

			items := dom.Find("ul._image-items>li.image-item")
			if items.Length() == 0 {
				done <- struct{}{}
			}

			items.Each(func(_ int, s *goquery.Selection) {
				href, exists := s.Find("a.work").Attr("href")
				if !exists {
					log.Println("Attribute 'href' dose not exist")
				} else {
					illustUrl := Pixiv + href
					out <- illustUrl
				}
			})
		}
	}()
	return out
}

func getSingleImageUrlAndMultiPreviewUrl(client *http.Client, in <-chan string) (<-chan string, <-chan string) {
	out := make(chan string, 20)
	outMulti := make(chan string, 20)
	go func() {
		defer close(out)
		defer close(outMulti)
		for u := range in {
			res, err := client.Get(u)
			if err != nil {
				log.Fatalln(err)
			}
			defer res.Body.Close()

			dom, err := goquery.NewDocumentFromResponse(res)
			if err != nil {
				log.Fatalln(err)
			}

			if isMutiple(dom) {
				dom.Find("div.item-container").Each(func(_ int, s *goquery.Selection) {
					href, exists := s.Find("a").Attr("href")
					if !exists {
						log.Println("Attribute 'href' dose not exists")
					} else {
						outMulti <- Pixiv + href
					}
				})
			} else {
				src, exists := dom.Find("img.original-image").Attr("src")
				if !exists {
					log.Println("Attribute 'src' dose not exist")
					break
				}
				out <- src
			}
		}
	}()
	return out, outMulti
}

func isMutiple(dom *goquery.Document) bool {
	return dom.Find("div.item-container").Length() != 0
}

func getImageUrl(client *http.Client, in <-chan string) <-chan string {
	out := make(chan string, 20)
	defer close(out)
	go func() {
		for u := range in {
			res, err := client.Get(u)
			if err != nil {
				log.Fatalln(err)
			}
			defer res.Body.Close()

			dom, err := goquery.NewDocumentFromResponse(res)
			if err != nil {
				log.Fatalln(err)
			}

			dom.Find("img").Each(func(_ int, s *goquery.Selection) {
				src, exists := s.Attr("src")
				if !exists {
					log.Println("Attribute 'src' dose not exist")
				} else {
					out <- src
				}
			})
		}
	}()
	return out
}

func download(client *http.Client, in <-chan Image) {
	go func() {
		for image := range in {
			save(client, &image)
		}
	}()
}

func save(client *http.Client, img *Image) {
	res, err := client.Get(img.url)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	dir, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	filename := path.Join(dir, path.Base(img.url))

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
}
