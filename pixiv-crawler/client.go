package main

import (
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Route struct {
	Url    string
	Method string
}

type Job struct {
	Route
	Data interface{}
}

type Client struct {
	*http.Client
	addJob reflect.Value
}

const (
	IllustsSize = 20
)

func (c *Client) AddJob(j Job) {
	c.addJob.Call([]reflect.Value{reflect.ValueOf(j)})
}

func (c *Client) login(id string, pass string) {
	_, err := c.Get(PixivHost)
	if err != nil {
		log.Fatalln(err)
	}
	form := url.Values{
		"mode":     {"login"},
		"pixiv_id": {id},
		"pass":     {pass},
		"skip":     {"1"},
	}
	req, _ := http.NewRequest(
		"POST",
		"https://www.secure.pixiv.net/login.php",
		strings.NewReader(form.Encode()))
	//req, _ := http.NewRequest("POST", "http://127.0.0.1:9000/login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html, application/xhtml+xml, */*")
	req.Header.Set("Accept-Language", "zh-CN")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Host", "www.secure.pixiv.net")
	req.Header.Set("Referer", "http://www.pixiv.net/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/42.0.2311.135 Safari/537.36")

	res, err := c.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()
	ur, _ := url.Parse(PixivHost)
	log.Println(c.Jar.Cookies(ur))

	file, err := os.Create("tmp2.html")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		log.Fatalln(err)
	}
}

func (c *Client) GetAuthor(j Job) {
	res, err := c.Client.Get(j.Route.Url)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	dom, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Fatalln(err)
	}

	// Get author's name
	name := dom.Find("h1.user").First().Text()
	log.Println(dom.Find("h1.user").Length())
	log.Println(name)

	// Get author's id
	u, err := url.Parse(j.Route.Url)
	if err != nil {
		log.Fatalln(err)
	}
	id := u.Query().Get("id")

	// Get works count
	text := dom.Find("span.count-badge").First().Text()
	log.Println(text)
	re := regexp.MustCompile("[0-9]+")
	counts := re.FindAllString(text, -1)
	log.Println(counts)
	count, err := strconv.Atoi(counts[0])
	if err != nil {
		log.Fatalln(err)
	}
	pages := count/IllustsSize + 1

	for i := 1; i <= pages; i++ {
		u, err := url.Parse(IllustUrl)
		if err != nil {
			log.Fatalln(err)
		}

		q := u.Query()
		q.Set("id", id)
		q.Set("p", strconv.Itoa(i))
		u.RawQuery = q.Encode()
		url := u.String()

		c.AddJob(Job{Route{url, "GetIllusts"}, Author{id, name}})
	}
}

func (c *Client) GetIllusts(j Job) {
	res, err := c.Get(j.Route.Url)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	dom, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Fatalln(err)
	}

	items := dom.Find("ul._image-items>li.image-item")

	items.Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Find("a.work").Attr("href")

		if !exists {
			log.Println("Attribute 'href' dose not exist")
		} else {
			url := PixivHost + href

			c.AddJob(Job{Route{url, "GetIllust"}, j.Data})
		}
	})
}

func (c *Client) GetIllust(j Job) {
	res, err := c.Get(j.Route.Url)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	dom, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Fatalln(err)
	}

	// Parse url, find illust id
	u, err := url.Parse(j.Route.Url)
	if err != nil {
		log.Fatalln(err)
	}
	id := u.Query().Get("illust_id")

	// Find illust name
	name := dom.Find("div.ui-expander-target>h1.title").First().Text()

	illust := Illust{id, name, j.Data.(Author)}

	if dom.Find("div.multiple").Length() != 0 {
		// Find next url
		path, exists := dom.Find("div.works_display>a").First().Attr("href")
		if !exists {
			log.Fatalln("href not found")
		}

		url := PixivHost + "/" + path

		c.AddJob(Job{Route{url, "GetMulti"}, illust})
	} else {
		src, exists := dom.Find("img.original-image").Attr("data-src")
		if !exists {
			log.Fatalln(j, "Attribute 'data-src' dose not exist")
		} else {
			image := Image{Id: -1, Path: src, Illust: illust, Referer: j.Route.Url}

			c.AddJob(Job{Route{Url: src, Method: "Download"}, image})
		}
	}
}

func (c *Client) GetMulti(j Job) {
	res, err := c.Get(j.Route.Url)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	dom, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Fatalln(err)
	}

	dom.Find("div.item-container").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Find("a").Attr("href")
		if !exists {
			log.Println("Attribute 'href' dose not exists")
		} else {
			image := Image{Id: i, Illust: j.Data.(Illust)}

			url := PixivHost + href

			c.AddJob(Job{Route{url, "GetMultiFurther"}, image})
		}
	})
}

func (c *Client) GetMultiFurther(j Job) {
	res, err := c.Get(j.Route.Url)
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
			image := j.Data.(Image)
			image.Path = src
			image.Referer = j.Route.Url

			c.AddJob(Job{Route{Url: src, Method: "Download"}, image})
		}
	})
}

func (c *Client) Download(j Job) {
	image := j.Data.(Image)

	req, err := http.NewRequest("GET", j.Route.Url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("Accept", "image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Host", "i2.pixiv.net")
	req.Header.Set("Referer", image.Referer)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/42.0.2311.135 Safari/537.36")

	res, err := c.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	dir, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	filename := path.Join(dir, "images", path.Base(j.Route.Url))
	//filename := path.Join(dir, "tmp3.html")

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	size, err := io.Copy(file, res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(size)
}
