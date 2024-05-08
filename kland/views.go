package kland

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"time"
)

const (
	AnonymousUser = "Anonymous"
)

type PostView struct {
	Pid          int64     `json:"pid"`
	Tid          int64     `json:"tid"`
	CreatedOn    time.Time `json:"createdOn"`
	Content      string    `json:"content"`
	RealUsername string    `json:"realUsername,omitempty"`
	Trip         string    `json:"trip,omitempty"`
	HasImage     bool      `json:"hasImage"`
	IsBanned     bool      `json:"isBanned"`
	ImageLink    string    `json:"imageLink,omitempty"`
	Link         string    `json:"link,omitempty"`
	IPAddress    string    `json:"ipAddress"`
}

type ThreadView struct {
	Tid        int64     `json:"tid"`
	Link       string    `json:"link"`
	Subject    string    `json:"subject"`
	LastPostOn time.Time `json:"lastPostOn"`
	CreatedOn  time.Time `json:"createdOn"`
	PostCount  int       `json:"postCount"`
}

// Convert db post to view
func ConvertPost(post Post, config *Config) PostView {

	trip := post.Tripraw
	if trip != "" {
		hashed := sha512.Sum512([]byte(trip))
		trip = base64.StdEncoding.EncodeToString(hashed[:])[:10]
	}

	realUsername := post.Username
	if realUsername == "" {
		realUsername = AnonymousUser
	}

	image := post.Image
	if image == "" {
		image = "UNDEFINED"
	}

	link := fmt.Sprintf("%s/thread/%d#p%d", config.RootPath, post.Tid, post.Pid)
	imageLink := fmt.Sprintf("%s%s/%s", config.RootPath, ImageEndpoint, image)

	return PostView{
		Tid:          post.Tid,
		Pid:          post.Pid,
		Content:      post.Content,
		CreatedOn:    parseTime(post.Created),
		IPAddress:    post.Ipaddress,
		Trip:         trip,
		RealUsername: realUsername,
		Link:         link,
		ImageLink:    imageLink,
		IsBanned:     false, // TODO: GET BANS
		HasImage:     post.Image != "",
	}
}

// Convert db thread to ThreadView. ALL fields are set
func ConvertThread(thread Thread, config *Config) ThreadView {
	return ThreadView{
		Tid:        thread.Tid,
		Subject:    thread.Subject,
		CreatedOn:  parseTime(thread.Created),
		PostCount:  thread.PostCount,             //x.Posts.Count(),
		LastPostOn: parseTime(thread.LastPostOn), //LastPostOn : x.Posts.Max(x => (DateTime?)x.created) ?? new DateTime(0),
		Link:       fmt.Sprintf("%s/thread/%d", config.RootPath, thread.Tid),
	}
}
