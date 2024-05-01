package kland

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"
)

type PostView struct {
	Pid          int       `json:"pid"`
	Tid          int       `json:"tid"`
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
	Tid        int       `json:"tid"`
	Link       string    `json:"link"`
	Subject    string    `json:"subject"`
	LastPostOn time.Time `json:"lastPostOn"`
	CreatedOn  time.Time `json:"createdOn"`
	PostCount  int       `json:"postCount"`
}

// Convert db post to view
func ConvertPost(post Post, config *Config) PostView {

	trip := utils.StrGetOrDefault(post.tripraw, "")
	if trip != "" {
		hashed := sha512.Sum512([]byte(trip))
		trip = base64.StdEncoding.EncodeToString(hashed[:])[:10]
	}

	realUsername := utils.StrGetOrDefault(post.username, "Anonymous")

	link := fmt.Sprintf("%s/thread/%d#p%d", config.RootPath, post.tid, post.pid)
	imageLink := fmt.Sprintf("%s/i/%s", config.RootPath, utils.StrGetOrDefault(post.image, "UNDEFINED"))

	return PostView{
		Tid:          post.tid,
		Pid:          post.pid,
		Content:      post.content,
		CreatedOn:    parseTime(post.created),
		IPAddress:    post.ipaddress,
		Trip:         trip,
		RealUsername: realUsername,
		Link:         link,
		ImageLink:    imageLink,
		IsBanned:     false, // TODO: GET BANS
		HasImage:     !utils.IsNilOrEmpty(post.image),
	}
}

// Convert db thread to ThreadView. ALL fields are set
func ConvertThread(thread Thread, config *Config) ThreadView {
	return ThreadView{
		Tid:        thread.tid,
		Subject:    thread.subject,
		CreatedOn:  parseTime(thread.created),
		PostCount:  thread.postCount,                //x.Posts.Count(),
		LastPostOn: parseTimePtr(thread.lastPostOn), //LastPostOn : x.Posts.Max(x => (DateTime?)x.created) ?? new DateTime(0),
		Link:       fmt.Sprintf("%s/thread/%d", config.RootPath, thread.tid),
	}
}
