package ui

import (
	"github.com/nsf/termbox-go"
	"github.com/g-dx/rosslyn/slack"
	"log"
	"fmt"
	"time"
	"github.com/0xAX/notificator"
	"strconv"
	"os"
	"sort"
	"html"
)

// TODO: Put this behind an interface and add to controller
var notify *notificator.Notificator
var ui *log.Logger
var debug *log.Logger

func init() {
	notify = notificator.New(notificator.Options{AppName: "Rosslyn", })

	f, err := os.Create(os.ExpandEnv("${HOME}/.rosslyn/debug.log"))
	if err != nil {
		panic(err)
	}
	debug = log.New(f, "", log.Ldate | log.Ltime)

	f, err = os.Create(os.ExpandEnv("${HOME}/.rosslyn/ui.log"))
	if err != nil {
		panic(err)
	}
	ui = log.New(f, "", 0)
}

type Controller interface {
	SwitchChannel(cl *Channel)
	SelectChannel()
	SendMessage(text string)
	LoadMessages(cl *Channel)
	Redraw()
}

type controller struct {
	logger *log.Logger

	rtm *slack.RtmConnection
	apis slack.Apis

	termEvts chan termbox.Event
	userEvts chan func()

	chls       *ChannelList
	chl        *Channel

	chlsView *ChannelSelectionView
	chlView  *ChannelView
	view     View
}

var userTypingTimer *UserTypingTimer

func NewController(logger *log.Logger, apis slack.Apis) *controller {

	// Open connection
	rtm := slack.NewRtmConnection(logger, apis.RtmConnect())

	// Create controller
	ctrl := &controller{
		logger: logger,
		rtm: rtm,
		apis: apis,
		termEvts: make(chan termbox.Event, 5),
		userEvts: make(chan func(), 5),
		chls: &ChannelList{},
	}

	// Process groups
	grpAndChl := apis.GetGroupAndChannelList()
	for _, grp := range grpAndChl.Groups.Groups {
		// TODO: Skip "multiple person IM" as it's not clear how to render this...
		if !grp.IsMpim {
			cl := &Channel { id: grp.ID, name: grp.NameNormalized }
			go func() {
				info := apis.GetGroupInfo(cl.id)
				ctrl.userEvts <- func() {
					cl.unread = info.Group.UnreadCountDisplay
					if ctrl.isVisible(ctrl.chlsView) {
						ctrl.Redraw()
					}
				}
			}()
			ctrl.chls.add(cl)
		}
	}

	// Process channels
	for _, cl := range grpAndChl.Channels.Channels {
		// Only add channels we are a member of
		if cl.IsMember {
			chl := &Channel { id: cl.ID, name: cl.NameNormalized }
			go func() {
				info := apis.GetChannelInfo(chl.id)
				ctrl.userEvts <- func() {
					chl.unread = info.Channel.UnreadCountDisplay
					if ctrl.isVisible(ctrl.chlsView) {
						ctrl.Redraw()
					}
				}
			}()
			ctrl.chls.add(chl)
		}
	}

	// Process IM
	for _, im := range grpAndChl.IM.Ims {
		// TODO: Remove our user name - not sure why it's here...
		if apis.GetUserList().IsActive(im.User) {
			cl := &Channel{id: im.ID, name: fmt.Sprintf("%-25v (%v)",
				apis.GetUserList().GetRealName(im.User), apis.GetUserList().GetName(im.User)), user: im.User}
			go func() {
				info := apis.GetGroupInfo(cl.id)
				ctrl.userEvts <- func() {
					cl.unread = info.Group.UnreadCountDisplay
					if ctrl.isVisible(ctrl.chlsView) {
						ctrl.Redraw()
					}
				}
			}()
			ctrl.chls.add(cl)
		}
	}

	ctrl.chlsView = NewChannelListView(ctrl, ctrl.chls, apis.GetUserList())


	// TODO: Move me elsewhere
	userTypingTimer = &UserTypingTimer{typingTimeout, make(map[string]*time.Timer), ctrl.userEvts }


	// Find, load and set #edinburgh-ticketing as default
	_, edinburgh := ctrl.chls.find("G0L95H9TL") // #edinburgh-ticketing
	ctrl.SwitchChannel(edinburgh)

	go ctrl.eventLoop()
	return ctrl
}

func (ctrl *controller) eventLoop() {
	for {
		ctrl.termEvts <- termbox.PollEvent()
	}
}

func (ctrl *controller) Run() {

	ctrl.Redraw()
	for {
		select {
		case evt := <-ctrl.rtm.ReadEvent():
			ctrl.onSlackEvent(evt)
		case ev := <- ctrl.termEvts:
			if !ctrl.onTerminalEvent(ev) {
				return // Shutdown
			}
		case f := <- ctrl.userEvts:
			f()
		}
	}
}

func (ctrl *controller) onTerminalEvent(ev termbox.Event) bool {
	switch ev.Type {
	case termbox.EventKey:
		switch ev.Key {
		case termbox.KeyCtrlQ:
			ctrl.rtm.Close()
			return false
		case termbox.KeyCtrlW:
			// Debug
			ctrl.userEvts <- func() {
				ctrl.onSlackEvent(&slack.PresenceChange{ "U0N4UV70Q", "away"})
			}
		default:
			ctrl.view.OnKey(ev.Key, ev.Ch) // Current view
		}
	case termbox.EventResize:
		ctrl.Redraw()
	case termbox.EventError:
		panic(ev.Err)
	}
	return true
}

func (ctrl *controller) Redraw() {
	ctrl.view.Draw(&terminal{})
}


func (ctrl *controller) LoadMessages(cl *Channel) {

	debug.Printf("Loading Messages for Channel: %v\n", cl.name)
	start := time.Now()
	if len(cl.msgs) > 0 {
		start = fromTsToTime(cl.msgs[0].Ts)
	}

	users := ctrl.apis.GetUserList()
	history := ctrl.apis.GetChannelHistory(cl.id, start)

	msgs := make([]*Message, 0, len(history.Messages))

	for i := len(history.Messages)-1; i >= 0; i-- {

		msg := history.Messages[i]

		// TODO: what about other events?
		if (msg.Type == "message") {
			fe := Formatter{lookup: &slackLookup {users }}
			content, styles := fe.Format(html.UnescapeString(msg.Text))
			msgs = append(msgs, &Message{
				Text:     string(content),
				Ts:       msg.Ts,
				T:        tsToTime(msg.Ts),
				User:     users.GetName(msg.User),
				IsEdited: msg.Edited.Ts != "", // TODO: Is there a better way to handle this?
				Formats:  styles,
			})
		}
	}

	// Add to start of message list and correct pos
	cl.msgs = append(msgs, cl.msgs...)
	inc := len(history.Messages)-1
	if inc < 0 {
		inc = 0
	}
	cl.pos += inc
}

func (ctrl *controller) SelectChannel() {
	ctrl.view = ctrl.chlsView
	ctrl.Redraw()
}

func (ctrl *controller) SendMessage(msg string) {
	ctrl.rtm.SendEvent(slack.NewSimpleMessage(ctrl.chl.id, msg))

	now := time.Now()
	ctrl.chl.AddSent(&Message{ Text: msg, Ts: fmt.Sprintf("%v.00000", strconv.FormatInt(now.Unix(), 10)), T: now, User: "garyduprex"})
	ctrl.Redraw()
}

func (ctrl *controller) onSlackEvent(evt slack.Event) {
	switch msg := evt.(type) {
	case *slack.SimpleMessage:
		ctrl.onMessage(msg)
	case *slack.Response:
		ctrl.onResponse(msg)
	case *slack.UserTyping:
		ctrl.onUserTyping(msg)
	case *slack.DesktopNotification:
		ctrl.onDesktopNotification(msg)
	case *slack.MessageChanged:
		ctrl.onChangedMessage(msg)
	case *slack.MessageDeleted:
		ctrl.onDeletedMessage(msg)
	case *slack.MessageThreadReply:
		ctrl.onThreadReplyMessage(msg)
	case *slack.PresenceChange:
		ctrl.onPresenceChangeMessage(msg)
	default:
		ctrl.logger.Println("Unhandled Event: %v", msg)
	}
}
func (ctrl *controller) onPresenceChangeMessage(change *slack.PresenceChange) {
	ctrl.apis.GetUserList().SetPresence(change.User, change.Presence)
	if ctrl.isVisible(ctrl.chlsView) {
		ctrl.Redraw()
	}
}

func (ctrl *controller) onThreadReplyMessage(reply *slack.MessageThreadReply) {
	// Skip until message threads are implemented...
}

func (ctrl *controller) onDesktopNotification(alrt *slack.DesktopNotification) {
	notify.Push(fmt.Sprintf("New message from %v", alrt.Subtitle), alrt.Content, "", notificator.UR_NORMAL)
}

func (ctrl *controller) onMessage(msg *slack.SimpleMessage) {

	_, chl := ctrl.chls.find(msg.Channel)
	if chl == nil {
		ctrl.logger.Printf("Channel '%v' not found for new message - skipping...") // MPIM messages...
		return
	}

	// Don't bother displaying 'reply_to' - it's not exactly clear what they are for...
	if !msg.IsReplyTo() {
		userList := ctrl.apis.GetUserList()

		// Separate formatting from content
		fe := Formatter{ lookup: &slackLookup{userList }}
		content, styles := fe.Format(msg.Text)

		chl.AddReceived(&Message{
			User:    userList.GetName(msg.User),
			Ts:      msg.Ts,
			T:       tsToTime(msg.Ts),
			Text:    string(content),
			Formats: styles,
		})
	}

	if msg.IsEdit() {
		// find message and switch text!

	}

	// Remove them from "typing" monitor
	userTypingTimer.Remove(ctrl.apis.GetUserList().GetRealName(msg.User))
	ctrl.Redraw()
}

func (ctrl *controller) SwitchChannel(cl *Channel) {
	if cl != nil {
		if len(cl.msgs) == 0 {
			ctrl.LoadMessages(cl)
		}
		// Set channel
		ctrl.chl = cl
		ctrl.chlView = NewChannelView(ctrl, ctrl.chl)
		ctrl.view = ctrl.chlView

		// Clear existing "typing users"
		userTypingTimer.Clear()
	} else {
		ctrl.view = ctrl.chlView
	}
	ctrl.Redraw()
}

func (ctrl *controller) onResponse(resp *slack.Response) {

	// Find message with `reply_to` id and mark as ok or failed
	ctrl.logger.Println("Received Response: %v", resp)
}

func (ctrl *controller) findChannel(id string) *Channel {
	// Fast-path
	if ctrl.chl.id == id {
		return ctrl.chl
	}
	_, chl := ctrl.chls.find(id)
	return chl
}

func (ctrl *controller) onUserTyping(typing *slack.UserTyping) {
	if ctrl.chl.id == typing.Channel {
		// TODO: Switch back to User IDs and let the front end render the ID how it wants
		userTypingTimer.Add(ctrl.apis.GetUserList().GetRealName(typing.User), func() {
			userTypingTimer.Remove(ctrl.apis.GetUserList().GetRealName(typing.User))
			if ctrl.isVisible(ctrl.chlView) {
				ctrl.Redraw()
			}
		})
		if ctrl.isVisible(ctrl.chlView)  {
			ctrl.Redraw()
		}
	}
}
func (ctrl *controller) isVisible(view View) bool {
	return ctrl.view == view
}

func (ctrl *controller) onChangedMessage(edit *slack.MessageChanged) {
	chl := ctrl.findChannel(edit.Channel)
	msg := chl.findByTs(edit.PreviousMessage.Ts)
	if msg != nil {
		// Update TS
		msg.Ts = edit.Message.Ts

		// Parse style & update content
		fe := Formatter{ lookup: &slackLookup{ctrl.apis.GetUserList() }}
		content, styles := fe.Format(edit.Message.Text)
		msg.Text = string(content)
		msg.Formats = styles
		msg.IsEdited = true

		// Only redraw if are on screen
		if ctrl.chl.id == chl.id {
			ctrl.Redraw()
		}
	}
}

func (ctrl *controller) onDeletedMessage(delete *slack.MessageDeleted) {
	// TODO: Implement
}


type UserTypingTimer struct {
	timeout    time.Duration
	userTimers map[string]*time.Timer
	onTimeout  chan func()
}

const typingTimeout time.Duration = 5 * time.Second

func (utt *UserTypingTimer) Add(u string, f func()) {
	if t, ok := utt.userTimers[u]; ok {
		t.Stop()
		t.Reset(utt.timeout)
	} else {
		t := time.AfterFunc(utt.timeout, func() {
			utt.onTimeout <- f
		})
		utt.userTimers[u] = t
	}
}

func (utt *UserTypingTimer) Clear() {
	for u, t := range utt.userTimers {
		t.Stop()
		delete(utt.userTimers, u)
	}
}
func (utt *UserTypingTimer) Remove(u string) {
	if t, ok := utt.userTimers[u]; ok {
		t.Stop()
		delete(utt.userTimers, u)
	}
}

func (utt *UserTypingTimer) UsersTyping() []string {
	if len(utt.userTimers) == 0 {
		return nil
	}
	users := make([]string, 0, len(utt.userTimers))
	for u, _ := range utt.userTimers {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool { return users[i] < users[j] }) // Ensure consistent order...
	return users
}

// TODO: Could update *slack.UserList to implement this interface directly....

type slackLookup struct {
	user *slack.UserList
}

func (sl *slackLookup) GetUser(id string) string {
	return sl.user.GetName(id)
}

func (sl *slackLookup) GetChannel(channel string) string {
	// TODO!
	return channel
}
