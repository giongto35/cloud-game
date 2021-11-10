package webrtc

import "github.com/pion/webrtc/v3"

// handleInputChannel creates a new WebRTC data channel for user input.
// Default params -- ordered: true, negotiated: false.
func (w *WebRTC) handleInputChannel() error {
	channel, err := w.connection.CreateDataChannel("game-input", nil)
	if err != nil {
		return err
	}

	channel.OnOpen(func() {
		w.log.Debug().
			Str("label", channel.Label()).
			Uint16("id", *channel.ID()).
			Msg("Data channel [input] opened")
	})

	channel.OnError(func(err error) { w.log.Error().Err(err).Msg("Data channel [input]") })

	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		if msg.IsString {
			_ = channel.Send([]byte{0x42})
			return
		}
		// TODO: Can add recover here
		w.InputChannel <- msg.Data
	})

	channel.OnClose(func() { w.log.Debug().Msg("Data channel [input] has been closed") })
	return nil
}
