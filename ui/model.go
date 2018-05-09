package ui

import (
	"time"
	"strconv"
	"strings"
	"sort"
)


type Message struct {
	User     string
	Text     string
	Ts       string
	T        time.Time
	Formats  []format
	IsEdited bool
}

// ---------------------------------------------------------------------------------------------------------------------

type ChannelList struct {
	// TODO: Split into groups, channels, ims, mpims?
	chls []*Channel
}

func (cs *ChannelList) add(cl *Channel) {
	// Add and keep sorted
	cs.chls = append(cs.chls, cl)
	sort.Slice(cs.chls, func(i, j int) bool {
		return cs.chls[i].id < cs.chls[j].id
	})
}

func (cs *ChannelList) find(id string) (int, *Channel) {
	i := sort.Search(len(cs.chls), func(i int) bool { return cs.chls[i].id >= id })
	if i < len(cs.chls) && cs.chls[i].id == id {
		return i, cs.chls[i]
	}
	return -1, nil
}

func (cl *ChannelList) Size() int {
	return len(cl.chls)
}


// ---------------------------------------------------------------------------------------------------------------------

type Channel struct {
	id, name string
	msgs []*Message
	pos int
	unread int
	user string // IM channels only...
}

func (cl *Channel) AddSent(msg *Message) {

	cl.msgs = append(cl.msgs, msg)
	cl.pos = len(cl.msgs)-1 // When user added message!
}


func (cl *Channel) AddReceived(msg *Message) {
	updatePos := cl.id == cl.id && cl.pos == len(cl.msgs)-1
	cl.msgs = append(cl.msgs, msg)
	if updatePos {
		cl.pos = len(cl.msgs)-1
	}
}

func (cl *Channel) findByTs(ts string) *Message {

	// TODO: This is too slow! Find a better way to keep messages sorted...
	// Channels should be sorted by 'Ts'
	for _, msg := range cl.msgs {
		if msg.Ts == ts {
			return msg
		}
	}
	return nil
}

func fromTsToTime(ts string) time.Time {
	i, err := strconv.ParseInt(ts[:strings.Index(ts, ".")], 10, 64)
	if err != nil {
		panic(err)
	}
	return time.Unix(i, 0)
}
