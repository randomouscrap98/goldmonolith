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
func ConvertPost(post Post) PostView {
	sha512 := sha512.New()

	trip := utils.StrGetOrDefault(post.tripraw, "")
	if trip != "" {
		hashed := sha512.Sum([]byte(trip))
		trip = base64.StdEncoding.EncodeToString(hashed)[:10]
	}

	realUsername := utils.StrGetOrDefault(post.username, "Anonymous")

	link := fmt.Sprintf("/thread/%d#p%d", post.tid, post.pid)
	imageLink := fmt.Sprintf("/i/%s", utils.StrGetOrDefault(post.image, "UNDEFINED"))

	return PostView{
		Tid:          post.tid,
		Pid:          post.pid,
		Content:      post.content,
		CreatedOn:    post.created,
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
func ConvertThread(thread Thread) ThreadView {
	return ThreadView{
		Tid:        thread.tid,
		Subject:    thread.subject,
		CreatedOn:  thread.created,
		PostCount:  thread.postCount,  //x.Posts.Count(),
		LastPostOn: thread.lastPostOn, //LastPostOn : x.Posts.Max(x => (DateTime?)x.created) ?? new DateTime(0),
		Link:       fmt.Sprintf("/thread/%d", thread.tid),
	}
}