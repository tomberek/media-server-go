package mediaserver

import (
	"fmt"

	"github.com/chuckpreslar/emission"
	"github.com/notedit/media-server-go/sdp"
	uuid "github.com/satori/go.uuid"
)

type Transport struct {
	localIce         *sdp.ICEInfo
	localDtls        *sdp.DTLSInfo
	localCandidates  []*sdp.CandidateInfo
	remoteIce        *sdp.ICEInfo
	remoteDtls       *sdp.DTLSInfo
	remoteCandidates []*sdp.CandidateInfo
	bundle           RTPBundleTransport
	transport        DTLSICETransport
	username         StringFacade
	incomingStreams  map[string]*IncomingStream
	outgoingStreams  map[string]*OutgoingStream
	*emission.Emitter
}

func NewTransport(bundle RTPBundleTransport, remoteIce *sdp.ICEInfo, remoteDtls *sdp.DTLSInfo, remoteCandidates []*sdp.CandidateInfo,
	localIce *sdp.ICEInfo, localDtls *sdp.DTLSInfo, localCandidates []*sdp.CandidateInfo, disableSTUNKeepAlive bool) *Transport {

	transport := new(Transport)
	transport.remoteIce = remoteIce
	transport.remoteDtls = remoteDtls
	transport.localIce = localIce
	transport.localDtls = localDtls
	transport.bundle = bundle
	transport.Emitter = emission.NewEmitter()

	properties := NewProperties()

	properties.SetProperty("ice.localUsername", localIce.GetUfrag())
	properties.SetProperty("ice.localPassword", localIce.GetPassword())
	properties.SetProperty("ice.remoteUsername", remoteIce.GetUfrag())
	properties.SetProperty("ice.remotePassword", remoteIce.GetPassword())

	properties.SetProperty("dtls.setup", remoteDtls.GetSetup().String())
	properties.SetProperty("dtls.hash", remoteDtls.GetHash())
	properties.SetProperty("dtls.fingerprint", remoteDtls.GetFingerprint())

	properties.SetProperty("disableSTUNKeepAlive", disableSTUNKeepAlive)

	transport.username = NewStringFacade(localIce.GetUfrag() + ":" + remoteIce.GetUfrag())
	transport.transport = bundle.AddICETransport(transport.username, properties)

	// todo ontargetbitrate callback
	// SenderSideEstimatorListener

	var address string
	var port int
	for _, candidate := range remoteCandidates {
		if candidate.GetType() == "relay" {
			address = candidate.GetRelAddr()
			port = candidate.GetRelPort()
		} else {
			address = candidate.GetAddress()
			port = candidate.GetPort()
		}
		bundle.AddRemoteCandidate(transport.username, address, uint16(port))
	}

	transport.localCandidates = localCandidates
	transport.remoteCandidates = remoteCandidates

	transport.incomingStreams = make(map[string]*IncomingStream)
	transport.outgoingStreams = make(map[string]*OutgoingStream)

	return transport
}

func (t *Transport) SetBandwidthProbing() {

	// todo
}

func (t *Transport) SetMaxProbingBitrate() {

	// todo
}

func (t *Transport) SetRemoteProperties(audio *sdp.MediaInfo, video *sdp.MediaInfo) {

	properties := NewProperties()

	if audio != nil {
		num := 0
		for _, codec := range audio.GetCodecs() {
			item := fmt.Sprintf("audio.codecs.%d", num)
			properties.SetProperty(item+".codec", codec.GetCodec())
			properties.SetProperty(item+".pt", codec.GetType())
			if codec.HasRTX() {
				properties.SetProperty(item+".rtx", codec.GetRTX())
			}
			num = num + 1
		}
		properties.SetProperty("audio.codecs.length", num)

		num = 0
		for id, uri := range audio.GetExtensions() {
			item := fmt.Sprintf("audio.ext.%d", num)
			properties.SetProperty(item+".id", id)
			properties.SetProperty(item+".uri", uri)
			num = num + 1
		}
		properties.SetProperty("audio.ext.length", num)
	}

	if video != nil {
		num := 0
		for _, codec := range video.GetCodecs() {
			item := fmt.Sprintf("video.codecs.%d", num)
			properties.SetProperty(item+".codec", codec.GetCodec())
			properties.SetProperty(item+".pt", codec.GetType())
			if codec.HasRTX() {
				properties.SetProperty(item+".rtx", codec.GetRTX())
			}
			num = num + 1
		}
		properties.SetProperty("video.codecs.length", num)

		num = 0
		for id, uri := range video.GetExtensions() {
			item := fmt.Sprintf("video.ext.%d", num)
			properties.SetProperty(item+".id", id)
			properties.SetProperty(item+".uri", uri)
			num = num + 1
		}
		properties.SetProperty("video.ext.length", num)
	}

	t.transport.SetRemoteProperties(properties)
}

func (t *Transport) GetLocalDTLSInfo() *sdp.DTLSInfo {

	return t.localDtls
}

func (t *Transport) GetLocalICEInfo() *sdp.ICEInfo {

	return t.localIce
}

func (t *Transport) GetLocalCandidates() []*sdp.CandidateInfo {

	return t.localCandidates
}

func (t *Transport) GetRemoteCandidates() []*sdp.CandidateInfo {

	return t.remoteCandidates
}

func (t *Transport) AddRemoteCandidate(candidate *sdp.CandidateInfo) {

	var address string
	var port int

	if candidate.GetType() == "relay" {
		address = candidate.GetRelAddr()
		port = candidate.GetRelPort()
	} else {
		address = candidate.GetAddress()
		port = candidate.GetPort()
	}

	t.bundle.AddRemoteCandidate(t.username, address, uint16(port))

	t.remoteCandidates = append(t.remoteCandidates, candidate)

}

func (t *Transport) CreateOutgoingStream(streamInfo *sdp.StreamInfo) *OutgoingStream {

	info := streamInfo.Clone()
	outgoingStream := NewOutgoingStream(t.transport, info)

	outgoingStream.Once("stopped", func() {
		delete(t.outgoingStreams, outgoingStream.GetID())
	})

	t.outgoingStreams[outgoingStream.GetID()] = outgoingStream

	return outgoingStream
}

func (t *Transport) CreateOutgoingStreamTrack(media string, trackId string, ssrcs map[string]uint) *OutgoingStreamTrack {

	var mediaType MediaFrameType = 0
	if media == "video" {
		mediaType = 1
	}

	if trackId == "" {
		trackId = uuid.Must(uuid.NewV4()).String()
	}

	source := NewRTPOutgoingSourceGroup(mediaType)

	if ssrc, ok := ssrcs["media"]; ok {
		source.GetMedia().SetSsrc(ssrc)
	} else {
		source.GetMedia().SetSsrc(GenerateSSRC())
	}

	if ssrc, ok := ssrcs["rtx"]; ok {
		source.GetRtx().SetSsrc(ssrc)
	} else {
		source.GetRtx().SetSsrc(GenerateSSRC())
	}

	if ssrc, ok := ssrcs["fec"]; ok {
		source.GetFec().SetSsrc(ssrc)
	} else {
		source.GetFec().SetSsrc(GenerateSSRC())
	}

	// todo error handle
	t.transport.AddOutgoingSourceGroup(source)

	outgoingTrack := newOutgoingStreamTrack(media, trackId, TransportToSender(t.transport), source)

	outgoingTrack.Once("stopped", func() {
		t.transport.RemoveOutgoingSourceGroup(source)
	})

	return outgoingTrack
}

func (t *Transport) CreateIncomingStream(streamInfo *sdp.StreamInfo) *IncomingStream {

	incomingStream := newIncomingStream(t.transport, TransportToReceiver(t.transport), streamInfo)

	t.incomingStreams[incomingStream.GetID()] = incomingStream

	incomingStream.Once("stopped", func() {
		delete(t.incomingStreams, incomingStream.GetID())
	})

	return incomingStream
}

func (t *Transport) CreateIncomingStreamTrack(media string, trackId string, ssrcs map[string]uint) *IncomingStreamTrack {

	var mediaType MediaFrameType = 0
	if media == "video" {
		mediaType = 1
	}

	if trackId == "" {
		trackId = uuid.Must(uuid.NewV4()).String()
	}

	source := NewRTPIncomingSourceGroup(mediaType)

	if ssrc, ok := ssrcs["media"]; ok {
		source.GetMedia().SetSsrc(ssrc)
	} else {
		source.GetMedia().SetSsrc(GenerateSSRC())
	}

	if ssrc, ok := ssrcs["rtx"]; ok {
		source.GetRtx().SetSsrc(ssrc)
	} else {
		source.GetRtx().SetSsrc(GenerateSSRC())
	}

	if ssrc, ok := ssrcs["fec"]; ok {
		source.GetFec().SetSsrc(ssrc)
	} else {
		source.GetFec().SetSsrc(GenerateSSRC())
	}

	t.transport.AddIncomingSourceGroup(source)

	sources := []RTPIncomingSourceGroup{source}

	incomingTrack := newIncomingStreamTrack(media, trackId, TransportToReceiver(t.transport), sources)

	incomingTrack.Once("stopped", func() {
		for _, item := range sources {
			t.transport.RemoveIncomingSourceGroup(item)
		}
	})

	return incomingTrack
}

// todo create simple outgoing stream

// Stop stop this transport
func (t *Transport) Stop() {

	if t.bundle == nil {
		return
	}

	for _, incoming := range t.incomingStreams {
		incoming.Stop()
	}

	for _, outgoing := range t.outgoingStreams {
		outgoing.Stop()
	}

	t.incomingStreams = nil
	t.outgoingStreams = nil

	t.bundle.RemoveICETransport(t.username)

	t.Emit("stopped", t)

	t.username = nil
	t.bundle = nil

}
