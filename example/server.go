package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	mediaserver "github.com/notedit/media-server-go"
	"github.com/notedit/sdp"
)

type Message struct {
	Cmd string `json:"cmd,omitempty"`
	Sdp string `json:"sdp,omitempty"`
}

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var Capabilities = map[string]*sdp.Capability{
	"audio": &sdp.Capability{
		Codecs: []string{"opus"},
	},
	"video": &sdp.Capability{
		Codecs: []string{"h264"},
		Rtx:    true,
		Rtcpfbs: []*sdp.RtcpFeedback{
			&sdp.RtcpFeedback{
				ID: "goog-remb",
			},
			&sdp.RtcpFeedback{
				ID: "transport-cc",
			},
			&sdp.RtcpFeedback{
				ID:     "ccm",
				Params: []string{"fir"},
			},
			&sdp.RtcpFeedback{
				ID:     "nack",
				Params: []string{"pli"},
			},
		},
		Extensions: []string{
			"urn:3gpp:video-orientation",
			"http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
			"http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			"urn:ietf:params:rtp-hdrext:toffse",
			"urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
			"urn:ietf:params:rtp-hdrext:sdes:mid",
		},
	},
}

func channel(c *gin.Context) {

	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	var transport *mediaserver.Transport
	endpoint := mediaserver.NewEndpoint("127.0.0.1")

	for {
		// read json
		var msg Message
		err = ws.ReadJSON(&msg)
		if err != nil {
			fmt.Println("error: ", err)
			break
		}

		if msg.Cmd != "offer" {
			continue
		}
		offer, err := sdp.Parse(msg.Sdp)
		if err != nil {
			panic(err)
		}
		transport = endpoint.CreateTransport(offer, nil)
		transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

		ice := transport.GetLocalICEInfo()
		dtls := transport.GetLocalDTLSInfo()
		candidates := endpoint.GetLocalCandidates()

		answer := sdp.NewSDPInfo()
		answer.SetICE(ice)
		answer.SetDTLS(dtls)
		answer.AddCandidates(candidates)

		if offer.GetMedia("audio") != nil {
			audioMedia := offer.GetMedia("audio").AnswerCapability(Capabilities["audio"])
			answer.AddMedia(audioMedia)
		}

		if offer.GetMedia("video") != nil {
			videoMedia := offer.GetMedia("video").AnswerCapability(Capabilities["video"])
			answer.AddMedia(videoMedia)
		}

		transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

		for i, stream := range offer.GetStreams() {
			println(i)

			incomingStream := transport.CreateIncomingStream(stream)
			outgoingStream := transport.CreateOutgoingStream(stream.Clone())

			outgoingStream.AttachTo(incomingStream)

			answer.AddStream(outgoingStream.GetStreamInfo())

			if len(incomingStream.GetVideoTracks()) > 0 {
				videoTrack := incomingStream.GetVideoTracks()[0]

				videoTrack.OnMediaFrame(func(frame []byte, timestamp uint) {
					//fmt.Println("media frame ===========")
					if len(frame) <= 4 {
						fmt.Println("media frame short ===========", len(frame))
						fmt.Println(string(frame))
						return
					}
					//appsrc.Push(frame)
				})
			}
		}

		ws.WriteJSON(Message{
			Cmd: "answer",
			Sdp: answer.String(),
		})
	}
}

func index(c *gin.Context) {
	fmt.Println("helloworld")
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func main() {
	godotenv.Load()
	address := ":8000"
	if os.Getenv("port") != "" {
		address = ":" + os.Getenv("port")
	}
	r := gin.Default()
	r.LoadHTMLFiles("./index.html")
	r.GET("/channel", channel)
	r.GET("/", index)
	r.Run(address)
}
