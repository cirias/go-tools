package main

import ()

type Author struct {
	Id   string
	Name string
}

type Illust struct {
	Id   string
	Name string
	Author
}

type Image struct {
	Id      int
	Path    string
	Referer string
	Illust
}

func (img Image) GetName(format string) string {

	return ""
}
