package slack

import (
	"time"
	"sort"
)

var id uint = uint(time.Now().Unix()) // Should ensure we don't overlap on application restarts

// ---------------------------------------------------------------------------------------------------------------------

type MsgType string

const (
	hello                MsgType = "hello"
	desktop_notification MsgType = "desktop_notification"
	user_typing          MsgType = "user_typing"
	message              MsgType = "message"
	response             MsgType = "response"
	presence_change      MsgType = "presence_change"
)

// ---------------------------------------------------------------------------------------------------------------------

type MsgSubType string

const (
	changed MsgSubType = "message_changed"
	deleted MsgSubType = "message_deleted"
	replied MsgSubType = "message_replied"
	none    MsgSubType = "<n/a>"
)

// ---------------------------------------------------------------------------------------------------------------------

type Event interface {
	Type() MsgType
}

type event struct {
	Typ    MsgType    `json:"type"`
	Subtyp MsgSubType `json:"subtype"`
}

func (e *event) Type() MsgType {
	return e.Typ
}

func (e *event) SubType() MsgSubType {
	return e.Subtyp
}

// ---------------------------------------------------------------------------------------------------------------------

type Hello struct{}

func (h *Hello) Type() MsgType {
	return hello
}

// ---------------------------------------------------------------------------------------------------------------------

// TODO: Is one "mega" message sufficient?
type SimpleMessage struct {
	Id      uint   `json:"id"`
	ReplyTo uint   `json:"reply_to"`
	Typ string     `json:"type"` // Needed for sends
	Channel string `json:"channel"`
	User    string `json:"user,omitempty"`
	Text    string `json:"text"`
	Ts      string `json:"ts,omitempty"`
	AsUser  bool   `json:"as_user"`
	Edited  struct {
		User string `json:"user"`
		Ts   string `json:"ts"`
	} `json:"edited"`
}

func (m *SimpleMessage) Type() MsgType {
	return message
}

func (m *SimpleMessage) IsReplyTo() bool { return m.ReplyTo != 0 }
func (m *SimpleMessage) IsEdit() bool    { return m.Edited.Ts != "" }

func NewSimpleMessage(channel, text string) Event {
	id++
	return &SimpleMessage{
		Id:      id,
		Typ:     string(message),
		Channel: channel,
		Text:    text,
		AsUser:  true,
	}
}

// ---------------------------------------------------------------------------------------------------------------------

type MessageChanged struct {
	Message struct {
		Type   string `json:"type"`
		User   string `json:"user"`
		Text   string `json:"text"`
		Edited struct {
			User string `json:"user"`
			Ts   string `json:"ts"`
		} `json:"edited"`
		Ts string `json:"ts"`
	} `json:"message"`
	Hidden          bool   `json:"hidden"`
	Channel         string `json:"channel"`
	PreviousMessage struct {
		Type string `json:"type"`
		User string `json:"user"`
		Text string `json:"text"`
		Ts   string `json:"ts"`
	} `json:"previous_message"`
	EventTs string `json:"event_ts"`
	Ts      string `json:"ts"`
}

func (em *MessageChanged) Type() MsgType {
	return message
}

// ---------------------------------------------------------------------------------------------------------------------

type MessageDeleted struct {
	DeletedTs string `json:"deleted_ts"`
	Hidden bool `json:"hidden"`
	Channel string `json:"channel"`
	PreviousMessage struct {
		Type string `json:"type"`
		User string `json:"user"`
		Text string `json:"text"`
		Edited struct {
			User string `json:"user"`
			Ts string `json:"ts"`
		} `json:"edited"`
		Ts string `json:"ts"`
	} `json:"previous_message"`
	EventTs string `json:"event_ts"`
	Ts string `json:"ts"`
}

func (em *MessageDeleted) Type() MsgType {
	return message
}

// ---------------------------------------------------------------------------------------------------------------------

type MessageThreadReply struct {
	Message struct {
		Type string `json:"type"`
		User string `json:"user"`
		Text string `json:"text"`
		ThreadTs string `json:"thread_ts"`
		ReplyCount int `json:"reply_count"`
		Replies []struct {
			User string `json:"user"`
			Ts string `json:"ts"`
		} `json:"replies"`
		Ts string `json:"ts"`
	} `json:"message"`
	Subtype string `json:"subtype"`
	Hidden bool `json:"hidden"`
	Channel string `json:"channel"`
	EventTs string `json:"event_ts"`
	Ts string `json:"ts"`
}

func (em *MessageThreadReply) Type() MsgType {
	return message
}

// ---------------------------------------------------------------------------------------------------------------------

type Response struct {
	Ok      bool   `json:"ok"`
	ReplyTo int    `json:"reply_to"`
	Ts      string `json:"ts,omitempty"`   // Only set in response to chat message
	Text    string `json:"text,omitempty"` // Only set in response to chat message
}

// Not pretty but we fudge this to keep representing everything as an event
func (r *Response) Type() MsgType {
	return response
}

// ---------------------------------------------------------------------------------------------------------------------

type RtmConnect struct {
	Ok   bool   `json:"ok"`
	URL  string `json:"url"`
	Team struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Domain string `json:"domain"`
	} `json:"team"`
	Self struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"self"`
}

// ---------------------------------------------------------------------------------------------------------------------

type MsgHistory struct {
	Ok       bool `json:"ok"`
	Messages []struct {
		Type string `json:"type"`
		User string `json:"user"`
		Text string `json:"text"`
	        Subtype string `json:"channel_join"`
		Ts   string `json:"ts"`
		Edited  struct {
			User string `json:"user"`
			Ts   string `json:"ts"`
		} `json:"edited"`
	} `json:"messages"`
	HasMore bool `json:"has_more"`
}

// ---------------------------------------------------------------------------------------------------------------------

type UserList struct {
	Ok      bool `json:"ok"`
	Members []struct {
		ID       string `json:"id"`
		TeamID   string `json:"team_id"`
		Name     string `json:"name"`
		Deleted  bool   `json:"deleted"`
		Color    string `json:"color"`
		RealName string `json:"real_name"`
		Tz       string `json:"tz"`
		TzLabel  string `json:"tz_label"`
		TzOffset int    `json:"tz_offset"`
		Profile  struct {
			AvatarHash         string `json:"avatar_hash"`
			Image24            string `json:"image_24"`
			Image32            string `json:"image_32"`
			Image48            string `json:"image_48"`
			Image72            string `json:"image_72"`
			Image192           string `json:"image_192"`
			Image512           string `json:"image_512"`
			Image1024          string `json:"image_1024"`
			ImageOriginal      string `json:"image_original"`
			RealName           string `json:"real_name"`
			RealNameNormalized string `json:"real_name_normalized"`
			Email              string `json:"email"`
		} `json:"profile"`
		IsAdmin           bool   `json:"is_admin"`
		IsOwner           bool   `json:"is_owner"`
		IsPrimaryOwner    bool   `json:"is_primary_owner"`
		IsRestricted      bool   `json:"is_restricted"`
		IsUltraRestricted bool   `json:"is_ultra_restricted"`
		IsBot             bool   `json:"is_bot"`
		Updated           int    `json:"updated"`
		Presence          string `json:"presence"`
	} `json:"members"`
	CacheTs int `json:"cache_ts"`
}

func (ul *UserList) GetName(id string) string {
	i := ul.find(id)
	if i == -1 {
		return "<unknown user>"
	}
	return ul.Members[i].Name
}

func (ul *UserList) GetRealName(id string) string {
	i := ul.find(id)
	if i == -1 {
		return "Unknown User"
	}
	return ul.Members[i].RealName
}

func (ul *UserList) GetPresence(id string) string {
	i := ul.find(id)
	if i == -1 {
		return "<?>"
	}
	return ul.Members[i].Presence
}

func (ul *UserList) SetPresence(id, presence string) {
	i := ul.find(id)
	if i != -1 {
		ul.Members[i].Presence = presence
	}
}

func (ul *UserList) IsActive(id string) bool {
	i := ul.find(id)
	if i == -1 {
		return false
	}
	return !ul.Members[i].Deleted
}

func (ul *UserList) find(id string) int {
	i := sort.Search(len(ul.Members), func(i int) bool { return ul.Members[i].ID >= id })
	if i < len(ul.Members) && ul.Members[i].ID == id {
		return i
	}
	return -1
}

type GroupList struct {
	Ok     bool `json:"ok"`
	Groups []struct {
		ID             string   `json:"id"`
		Name           string   `json:"name"`
		IsGroup        bool     `json:"is_group"`
		Created        int      `json:"created"`
		Creator        string   `json:"creator"`
		IsArchived     bool     `json:"is_archived"`
		NameNormalized string   `json:"name_normalized"`
		IsMpim         bool     `json:"is_mpim"`
		Members        []string `json:"members"`
		Topic          struct {
			Value   string `json:"value"`
			Creator string `json:"creator"`
			LastSet int    `json:"last_set"`
		} `json:"topic"`
		Purpose struct {
			Value   string `json:"value"`
			Creator string `json:"creator"`
			LastSet int    `json:"last_set"`
		} `json:"purpose"`
	} `json:"groups"`
}

type ChannelList struct {
	Ok       bool `json:"ok"`
	Channels []struct {
		ID             string   `json:"id"`
		Name           string   `json:"name"`
		IsChannel      bool     `json:"is_channel"`
		Created        int      `json:"created"`
		Creator        string   `json:"creator"`
		IsArchived     bool     `json:"is_archived"`
		IsGeneral      bool     `json:"is_general"`
		NameNormalized string   `json:"name_normalized"`
		IsShared       bool     `json:"is_shared"`
		IsOrgShared    bool     `json:"is_org_shared"`
		IsMember       bool     `json:"is_member"`
		Members        []string `json:"members"`
		Topic          struct {
			Value   string `json:"value"`
			Creator string `json:"creator"`
			LastSet int    `json:"last_set"`
		} `json:"topic"`
		Purpose struct {
			Value   string `json:"value"`
			Creator string `json:"creator"`
			LastSet int    `json:"last_set"`
		} `json:"purpose"`
		PreviousNames []interface{} `json:"previous_names"`
		NumMembers    int           `json:"num_members"`
	} `json:"channels"`
}

type GroupAndChannelList struct {
	Groups   *GroupList
	Channels *ChannelList
	IM       *InstantMessageList
}

func (gcl *GroupAndChannelList) IsChannel(id string) bool {
	for _, cl := range gcl.Channels.Channels {
		if cl.ID == id {
			return true
		}
	}
	return true
}

// ---------------------------------------------------------------------------------------------------------------------

type InstantMessageList struct {
	Ok  bool `json:"ok"`
	Ims []struct {
		ID            string `json:"id"`
		Created       int    `json:"created"`
		IsIm          bool   `json:"is_im"`
		IsOrgShared   bool   `json:"is_org_shared"`
		User          string `json:"user"`
		IsUserDeleted bool   `json:"is_user_deleted"`
	} `json:"ims"`
}

// ---------------------------------------------------------------------------------------------------------------------

type UserTyping struct {
	Channel string `json:"channel"`
	User    string `json:"user"`
}

func (m *UserTyping) Type() MsgType {
	return user_typing
}

// ---------------------------------------------------------------------------------------------------------------------

type DesktopNotification struct {
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	Msg         string `json:"msg"`
	Content     string `json:"content"`
	Channel     string `json:"channel"`
	LaunchURI   string `json:"launchUri"`
	AvatarImage string `json:"avatarImage"`
	SsbFilename string `json:"ssbFilename"`
	ImageURI    string `json:"imageUri"`
	IsShared    bool   `json:"is_shared"`
	EventTs     string `json:"event_ts"`
}

func (m *DesktopNotification) Type() MsgType {
	return desktop_notification
}

// ---------------------------------------------------------------------------------------------------------------------

type GroupInfo struct {
	Ok bool `json:"ok"`
	Group struct {
		ID string `json:"id"`
		Name string `json:"name"`
		IsGroup bool `json:"is_group"`
		Created int `json:"created"`
		Creator string `json:"creator"`
		IsArchived bool `json:"is_archived"`
		NameNormalized string `json:"name_normalized"`
		IsMpim bool `json:"is_mpim"`
		IsOpen bool `json:"is_open"`
		LastRead string `json:"last_read"`
		Latest struct {
			Type string `json:"type"`
			User string `json:"user"`
			Text string `json:"text"`
			Ts string `json:"ts"`
		} `json:"latest"`
		UnreadCount int `json:"unread_count"`
		UnreadCountDisplay int `json:"unread_count_display"`
		Members []string `json:"members"`
		Topic struct {
			Value string `json:"value"`
			Creator string `json:"creator"`
			LastSet int `json:"last_set"`
		} `json:"topic"`
		Purpose struct {
			Value string `json:"value"`
			Creator string `json:"creator"`
			LastSet int `json:"last_set"`
		} `json:"purpose"`
	} `json:"group"`
}

// ---------------------------------------------------------------------------------------------------------------------

type ChannelInfo struct {
	Ok bool `json:"ok"`
	Channel struct {
		ID string `json:"id"`
		Name string `json:"name"`
		IsGroup bool `json:"is_group"`
		Created int `json:"created"`
		Creator string `json:"creator"`
		IsArchived bool `json:"is_archived"`
		NameNormalized string `json:"name_normalized"`
		IsMpim bool `json:"is_mpim"`
		IsOpen bool `json:"is_open"`
		LastRead string `json:"last_read"`
		Latest struct {
			Type string `json:"type"`
			User string `json:"user"`
			Text string `json:"text"`
			Ts string `json:"ts"`
		} `json:"latest"`
		UnreadCount int `json:"unread_count"`
		UnreadCountDisplay int `json:"unread_count_display"`
		Members []string `json:"members"`
		Topic struct {
			Value string `json:"value"`
			Creator string `json:"creator"`
			LastSet int `json:"last_set"`
		} `json:"topic"`
		Purpose struct {
			Value string `json:"value"`
			Creator string `json:"creator"`
			LastSet int `json:"last_set"`
		} `json:"purpose"`
	} `json:"channel"`
}

// ---------------------------------------------------------------------------------------------------------------------

type PresenceChange struct {
	User string `json:"user"`
	Presence string `json:"presence"`
}

func (pc *PresenceChange) Type() MsgType {
	return presence_change
}