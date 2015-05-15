pixiv-crawler
----

Simple tool for downloading pixiv images by author's id.

### Usage

```
Usage: pixiv-crawler [options] <id>
  -dir="/Users/sirius/go/src/github.com/cirias/go-tools/pixiv-crawler": the directory to save the images
  -dir-format="{{Author.Name}}-{{Author.Id}}": the format of the directory name
  -file-format="pixiv-{{Illust.Id}}-{{Illust.Name}}-{{Author.Name}}-{{Image.Id}}": the format of the image name
  -pass="": the password of the login user
  -user="": the user name to login
  -work-size=10: the max count of concurreny working jobs
```
