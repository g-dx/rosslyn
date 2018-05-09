package slack

import (
	"log"
	"encoding/json"
	"github.com/gorilla/websocket"
	"time"
	"errors"
	"net/http"
	"io/ioutil"
	"fmt"
	"net/url"
	"strconv"
	"os"
	"strings"
	"sort"
)

const (
	apiUrl = "https://slack.com/api/"
)

var debug *log.Logger

func init() {
	f, err := os.Create(os.ExpandEnv("${HOME}/.rosslyn/api.log"))
	if err != nil {
		panic(err)
	}
	debug = log.New(f, "", log.Ldate | log.Ltime)
}


type Apis interface {
	MarkChannel(id string, ts *time.Time)
	GetUserList() *UserList
	GetChannelInfo(channel string) *ChannelInfo
	GetChannelHistory(channel string, start time.Time) *MsgHistory
	GetGroupInfo(channel string) *GroupInfo
	GetGroupAndChannelList() *GroupAndChannelList
	RtmConnect() *websocket.Conn
}

type apis struct {

	token string

	users     *UserList
	grpAndChn *GroupAndChannelList
}

func NewApis(token []byte) Apis {
	return &apis{ token: string(token) }
}

func (api *apis) MarkChannel(id string, time *time.Time) {

	method := ""
	switch id[:1] {
	case "C":
		method = "channels.mark"
	case "D":
		method = "im.mark"
	case "G":
		method = "groups.mark"
	default:
		panic(errors.New(fmt.Sprintf("Unable to determine history API for ID: %v", id)))
	}


	ts := fmt.Sprintf("%v.00000", strconv.FormatInt(time.Unix(), 10))

	// Load & cache users
	var empty struct{}
	err := api.call(method, map[string]string {"channel": id, "ts": ts }, &empty)
	if err != nil {
		panic(err)
	}
}

func (api *apis) GetUserList() *UserList {

	if api.users != nil {
		return api.users
	}

	// Load & cache users
	var users UserList
	err := api.call("users.list", map[string]string {"presence": "true"}, &users)
	if err != nil {
		panic(err)
	}

	// Sort members
	sort.Slice(users.Members, func(i, j int) bool { return users.Members[i].ID < users.Members[j].ID })
	api.users = &users
	return api.users
}

func (api *apis) GetGroupInfo(id string) *GroupInfo {

	// Load info
	var info GroupInfo
	err := api.call("groups.info", map[string]string {"channel": id }, &info)
	if err != nil {
		panic(err)
	}
	return &info
}

func (api *apis) GetChannelInfo(id string) *ChannelInfo {

	// Load info
	var info ChannelInfo
	err := api.call("channels.info", map[string]string {"channel": id }, &info)
	if err != nil {
		panic(err)
	}
	return &info
}

func (api *apis) GetChannelHistory(id string, start time.Time) *MsgHistory {

	method := ""
	switch id[:1] {
	case "C":
		method = "channels.history"
	case "D":
		method = "im.history"
	case "G":
		method = "groups.history"
	default:
		panic(errors.New(fmt.Sprintf("Unable to determine history API for ID: %v", id)))
	}

	ts := fmt.Sprintf("%v.00000", strconv.FormatInt(start.Unix(), 10))

	// Load messages
	var history MsgHistory
	err := api.call(method, map[string]string {"channel": id, "latest": ts, "count": "50"}, &history)
	if err != nil {
		panic(err)
	}
	return &history
}

func (api *apis) GetGroupAndChannelList() *GroupAndChannelList {

	if api.grpAndChn != nil {
		return api.grpAndChn
	}

	// query
	params := map[string]string{"exclude_archived": "true", "exclude_members": "true"}

	// Load public channels
	var chls ChannelList
	err := api.call("channels.list", params, &chls)
	if err != nil {
		panic(err)
	}

	// Load private groups
	var groups GroupList
	err = api.call("groups.list", params, &groups)
	if err != nil {
		panic(err)
	}

	// Load private IM
	var im InstantMessageList
	err = api.call("im.list", map[string]string {}, &im)
	if err != nil {
		panic(err)
	}

	// Join
	api.grpAndChn = &GroupAndChannelList{&groups, &chls, &im}
	return api.grpAndChn
}



func (api *apis) RtmConnect() *websocket.Conn {

	// Connect
	var connect RtmConnect
	err := api.call("rtm.connect", map[string]string{}, &connect)
	if err != nil {
		panic(err)
	}

	// Open websocket
	conn, _, err := websocket.DefaultDialer.Dial(connect.URL, http.Header{})
	if err != nil {
		panic(err)
	}
	return conn
}

func (api *apis) call(method string, params map[string]string, i interface{}) error {

	// Add token
	params["token"] = api.token

	// Build query string
	pairs := make([]string, 0, len(params))
	for k, v := range params {
		pairs = append(pairs, k + "=" + url.QueryEscape(v))
	}

	// Make call
	apiCall := fmt.Sprintf("%v%v?%v", apiUrl, method, strings.Join(pairs, "&"))
	debug.Println(strings.Replace(apiCall, api.token, "<removed>", 1))
	resp, err := http.Get(apiCall)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Unmarshall
	err = json.Unmarshal(data, i)
	if err != nil {
		return err
	}
	return nil
}

type RtmConnection struct {
	logger *log.Logger

	conn *websocket.Conn

	writeQ chan Event
	readQ  chan Event

	close chan chan struct{}
}

func NewRtmConnection(logger *log.Logger, conn *websocket.Conn) *RtmConnection {

	c := &RtmConnection{
		logger: logger,
		conn: conn,
		writeQ: make(chan Event, 5),
		readQ: make(chan Event, 5),
		close: make(chan chan struct{}),
	}
	go c.writeLoop()
	go c.readLoop()
	return c
}

func (c *RtmConnection) ReadEvent() chan Event {
	return c.readQ
}

func (c *RtmConnection) SendEvent(evt Event) error {
	select {
	case c.writeQ <- evt:
		return nil
	case <- time.After(time.Millisecond * 500):
		return errors.New("Write Q is full")
	}
}

func (c *RtmConnection) Close() error {

	// Signal to stop
	done := make(chan struct{})
	c.close <- done

	// Wait a max of 2 seconds
	select {
	case <- done:
	case <- time.After(time.Second * 2):
	}

	// Close underlying connection
	return c.conn.Close()
}

func (c *RtmConnection) readLoop() {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.Printf("Read error: %v", err) // TODO: Writer should set a flag to ignore errors after shutdown initiated
			return
		}

		c.readQ <- c.unmarshalEvent(data)
	}
}

func (rtm *RtmConnection) writeLoop() {

	// Tick every second to send pongs
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-rtm.writeQ:

			err := rtm.conn.WriteMessage(websocket.TextMessage, rtm.marshal(msg))
			if err != nil {
				panic(err) // TODO: Revisit
			}

		case <-ticker.C:

			err := rtm.conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				panic(err) // TODO: Revisit - should reconnect
			}

		case done := <-rtm.close:
			rtm.logger.Println("interrupt")

			// Stop ticker
			ticker.Stop()

			// To cleanly close a connection, a client should send a close
			// frame and wait for the server to close the connection.
			err := rtm.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				rtm.logger.Println("write close:", err)
			}

			// TODO: Should we wait on reader loop finishing?

			// Tell outer we are done
			done <- struct{}{}
			rtm.logger.Println("Writer finished")
		}
	}

}

func (c *RtmConnection) unmarshalEvent(data []byte) Event {

	// Attempt event first as these are more likely
	var e event
	c.unmarshal(data, &e)

	// No type - therefore it is a response
	if e.Type() == "" {
		var resp Response
		c.unmarshal(data, &resp)
		c.logger.Printf("> Response: %v", string(data))
		return &resp
	}

	// Unmarshal to specific type
	c.logger.Printf("> Event   : %v", string(data))
	switch e.Type() {
	case message:
		return c.unmarshalMessage(e.SubType(), data)
	case user_typing:
		var msg UserTyping
		c.unmarshal(data, &msg)
		return &msg
	case desktop_notification:
		var msg DesktopNotification
		c.unmarshal(data, &msg)
		return &msg
	case presence_change:
		var msg PresenceChange
		c.unmarshal(data, &msg)
		return &msg
	default:
		return &e // Can't do anything else with this
	}
}

func (c *RtmConnection) unmarshalMessage(sType MsgSubType, data []byte) Event {
	switch sType {
	case changed:
		var msg MessageChanged
		c.unmarshal(data, &msg)
		return &msg
	case deleted:
		var msg MessageDeleted
		c.unmarshal(data, &msg)
		return &msg
	case replied:
		var msg MessageThreadReply
		c.unmarshal(data, &msg)
		return &msg
	case none:
		fallthrough
	default:
		// return simple
		var msg SimpleMessage
		c.unmarshal(data, &msg)
		return &msg
	}
}


func (c *RtmConnection) marshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		c.logger.Printf("Marshalling Failure: %v", v)
		panic(err)
	}
	return data
}

func (c *RtmConnection) unmarshal(data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	if err != nil {
		c.logger.Printf("Unmarshalling Failure: %v", v)
		panic(err)
	}
}