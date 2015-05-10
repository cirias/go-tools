package main

import (
	"github.com/PuerkitoBio/goquery"
	"strconv"
	//"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

const (
	Pixiv        = "http://www.pixiv.net"
	MemberIllust = Pixiv + "/member_illust.php"
)

func main() {
	cookieJar, _ := cookiejar.New(nil)

	client := &http.Client{
		Jar: cookieJar,
	}

	//res, err := client.Get("http://www.google.com")
	res, err := client.PostForm("https://www.secure.pixiv.net/login.php",
		url.Values{
			"mode":      {"login"},
			"return_to": {"/"},
			"pixiv_id":  {"fhc023"},
			"pass":      {"f32645664"},
			"skip":      {"1"},
		})
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	url, err := url.Parse("http://www.pixiv.net/")
	if err != nil {
		log.Fatalln(err)
	}

	log.Print(cookieJar.Cookies(url))

	//body, err := ioutil.ReadAll(res.Body)
	//log.Println(string(body))

	_, err = goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("ok\n")
}

func produceListUrl(id string, out chan string, done *bool) {
	defer close(out)
	for i := 1; !*done; i++ {
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

func produceIllustUrl(in chan string, out chan string, done *bool, client http.Client) {
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
			*done = false
			break
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
}

func produceSingleImageUrl(in chan string, img chan string, out chan string, client http.Client) {
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

		if isMutiple(dom) {
			dom.Find("div.item-container").Each(func(_ int, s *goquery.Selection) {
				href, exists := s.Find("a").Attr("href")
				if !exists {
					log.Println("Attribute 'href' dose not exists")
				} else {
					out <- Pixiv + href
				}
			})
		} else {
			src, exists := dom.Find("img.original-image").Attr("src")
			if !exists {
				log.Println("Attribute 'src' dose not exist")
				break
			}
			img <- src
		}
	}
}

func isMutiple(dom *goquery.Document) bool {
	return dom.Find("div.item-container").Length() != 0
}

func produceImageUrl(in chan string, out chan string, client *http.Client) {
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

		dom.Find("img").Each(func(_ int, s *goquery.Selection) {
			src, exists := s.Attr("src")
			if !exists {
				log.Println("Attribute 'src' dose not exist")
			} else {
				out <- src
			}
		})
	}
}

func download(in chan string, client *http.Client) {

}
