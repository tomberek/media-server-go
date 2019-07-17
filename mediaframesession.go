package mediaserver

import (
	"fmt"
	"strings"

	native "github.com/notedit/media-server-go/wrapper"
	"github.com/notedit/sdp"
)

type MediaFrameSession struct {
	incoming *IncomingStreamTrack
	session  native.MediaFrameSessionFacade
}

// NewMediaFrameSession create media frame session
func NewMediaFrameSession(media *sdp.MediaInfo) *MediaFrameSession {

	mediaSession := &MediaFrameSession{}
	var mediaType native.MediaFrameType = 0
	if strings.ToLower(media.GetType()) == "video" {
		mediaType = 1
	}

	session := native.NewMediaFrameSessionFacade(mediaType)

	properties := native.NewProperties()
	if media != nil {
		num := 0
		for _, codec := range media.GetCodecs() {
			item := fmt.Sprintf("codecs.%d", num)
			properties.SetProperty(item+".codec", codec.GetCodec())
			properties.SetProperty(item+".pt", codec.GetType())
			if codec.HasRTX() {
				properties.SetProperty(item+".rtx", codec.GetRTX())
			}
			num = num + 1
		}
		properties.SetProperty("codecs.length", num)
	}

	session.Init(properties)
	native.DeleteProperties(properties)
	mediaSession.session = session
	mediaSession.incoming = NewIncomingStreamTrack(media.GetType(), media.GetType(), native.RTPSessionToReceiver(session), map[string]native.RTPIncomingSourceGroup{"": session.GetIncomingSourceGroup()})

	return mediaSession
}

// GetIncomingStreamTrack get incoming stream track
func (s *MediaFrameSession) GetIncomingStreamTrack() *IncomingStreamTrack {
	return s.incoming
}

// Push push raw media frame
func (s *MediaFrameSession) Push(rtp []byte) {
	if rtp == nil || len(rtp) == 0 {
		return
	}
	s.session.OnRTPPacket(&rtp[0], len(rtp))
}

// Stop stop this
func (s *MediaFrameSession) Stop() {

	if s.session == nil {
		return
	}

	if s.incoming != nil {
		s.incoming.Stop()
	}

	s.session.End()

	native.DeleteMediaFrameSessionFacade(s.session)

	s.session = nil
}
