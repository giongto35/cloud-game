package room

import "github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"

type MediaPipe interface {
	// Destroy frees all allocated resources.
	Destroy()
	// Init initializes the pipe: allocates needed resources.
	Init() error
	// PushAudio pushes the 16bit PCM audio frames into an encoder.
	// Because we need to fill the buffer, the SetAudioCb should be
	// used in order to get the result.
	PushAudio([]int16)
	// ProcessVideo returns encoded video frame.
	ProcessVideo(app.Video) []byte
	// SetAudioCb sets a callback for encoded audio data with its frame duration (ns).
	SetAudioCb(func(data []byte, duration int32))
}

type SessionManager[T Session] interface {
	Add(T) bool
	Find(string) (T, bool)
	ForEach(func(T))
	Len() int
	Remove(T)
	// Reset used for proper cleanup of the resources if needed.
	Reset()
}

type Session interface {
	Disconnect()
	SendAudio([]byte, int32)
	SendVideo([]byte, int32)
	SendData([]byte)
}

type Uid interface {
	Id() string
}

type Room[T Session] struct {
	app   app.App
	id    string
	media MediaPipe
	users SessionManager[T]

	closed      bool
	HandleClose func()
}

func NewRoom[T Session](id string, app app.App, um SessionManager[T], media MediaPipe) *Room[T] {
	room := &Room[T]{id: id, app: app, users: um, media: media}
	room.InitVideo()
	room.InitAudio()
	return room
}

func (r *Room[T]) InitAudio() {
	r.app.SetAudioCb(func(a app.Audio) { r.media.PushAudio(a.Data) })
	r.media.SetAudioCb(func(d []byte, l int32) { r.users.ForEach(func(u T) { u.SendAudio(d, l) }) })
}

func (r *Room[T]) InitVideo() {
	r.app.SetVideoCb(func(v app.Video) {
		data := r.media.ProcessVideo(v)
		r.users.ForEach(func(u T) { u.SendVideo(data, v.Duration) })
	})
}

func (r *Room[T]) App() app.App { return r.app }
func (r *Room[T]) Id() string   { return r.id }
func (r *Room[T]) StartApp()    { r.app.Start() }

func (r *Room[T]) Close() {
	if r.closed {
		return
	}
	r.closed = true

	if r.app != nil {
		r.app.Close()
	}
	if r.media != nil {
		r.media.Destroy()
	}
	if r.HandleClose != nil {
		r.HandleClose()
	}
}

// Router tracks and routes freshly connected users to an app room.
// Rooms and users has 1-to-n relationship.
type Router[T Session] struct {
	room  *Room[T]
	users SessionManager[T]
}

func (r *Router[T]) AddUser(user T) { r.users.Add(user) }
func (r *Router[T]) Close() {
	if r.room != nil {
		r.room.Close()
		r.room = nil
	}
}
func (r *Router[T]) FindRoom(id string) *Room[T] {
	if r.room != nil && r.room.Id() == id {
		return r.room
	}
	return nil
}
func (r *Router[T]) FindUser(uid Uid) T { sess, _ := r.users.Find(uid.Id()); return sess }
func (r *Router[T]) Remove(user T) {
	r.users.Remove(user)
	if r.users.Len() == 0 {
		if r.room != nil {
			r.room.Close()
		}
		r.users.Reset()
	}
}
func (r *Router[T]) SetRoom(room *Room[T])    { r.room = room }
func (r *Router[T]) Users() SessionManager[T] { return r.users }

type AppSession struct {
	Uid
	Session
	uid string
}

func (p AppSession) Id() string { return p.uid }

type GameSession struct {
	AppSession
	Index int // track user Index (i.e. player 1,2,3,4 select)
}

func NewGameSession(id Uid, s Session) *GameSession {
	return &GameSession{AppSession: AppSession{uid: id.Id(), Session: s}}
}
