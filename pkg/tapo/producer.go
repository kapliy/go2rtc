package tapo

import (
	"encoding/json"
	"strings"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/mpegts"
)

func (c *Client) GetMedias() []*core.Media {
	if c.medias == nil {
		// C460 (and likely other 4K-capable battery Tapos) emit H265.
		// Advertise both; #video=h264 / #video=h265 fragment narrows the list.
		videoCodecs := []*core.Codec{
			{Name: core.CodecH264, ClockRate: 90000, PayloadType: core.PayloadTypeRAW},
			{Name: core.CodecH265, ClockRate: 90000, PayloadType: core.PayloadTypeRAW},
		}
		if c.url != nil && c.url.Fragment != "" {
			for _, frag := range strings.Split(c.url.Fragment, "#") {
				if !strings.HasPrefix(frag, "video=") {
					continue
				}
				want := strings.TrimPrefix(frag, "video=")
				var filtered []*core.Codec
				for _, codec := range videoCodecs {
					if strings.EqualFold(codec.Name, want) {
						filtered = append(filtered, codec)
					}
				}
				if len(filtered) > 0 {
					videoCodecs = filtered
				}
			}
		}
		c.medias = []*core.Media{
			{
				Kind:      core.KindVideo,
				Direction: core.DirectionRecvonly,
				Codecs:    videoCodecs,
			},
			{
				Kind:      core.KindAudio,
				Direction: core.DirectionRecvonly,
				Codecs: []*core.Codec{
					{Name: core.CodecPCMA, ClockRate: 8000, PayloadType: 8},
				},
			},
			{
				Kind:      core.KindAudio,
				Direction: core.DirectionSendonly,
				Codecs: []*core.Codec{
					{Name: core.CodecPCMA, ClockRate: 8000, PayloadType: 8},
				},
			},
		}
	}

	return c.medias
}

func (c *Client) GetTrack(media *core.Media, codec *core.Codec) (*core.Receiver, error) {
	for _, track := range c.receivers {
		if track.Codec == codec {
			return track, nil
		}
	}

	if err := c.SetupStream(); err != nil {
		return nil, err
	}

	track := core.NewReceiver(media, codec)
	switch media.Kind {
	case core.KindVideo:
		if codec.Name == core.CodecH265 {
			track.ID = mpegts.StreamTypeH265
		} else {
			track.ID = mpegts.StreamTypeH264
		}
	case core.KindAudio:
		track.ID = mpegts.StreamTypePCMATapo
	}
	c.receivers = append(c.receivers, track)
	return track, nil
}

func (c *Client) Start() error {
	return c.Handle()
}

func (c *Client) Stop() error {
	for _, receiver := range c.receivers {
		receiver.Close()
	}
	if c.sender != nil {
		c.sender.Close()
	}
	return c.Close()
}

func (c *Client) MarshalJSON() ([]byte, error) {
	info := &core.Connection{
		ID:         core.ID(c),
		FormatName: c.url.Scheme,
		Protocol:   "http",
		Medias:     c.medias,
		Recv:       c.recv,
		Receivers:  c.receivers,
		Send:       c.send,
	}
	if c.sender != nil {
		info.Senders = []*core.Sender{c.sender}
	}
	if c.conn1 != nil {
		info.RemoteAddr = c.conn1.RemoteAddr().String()
	}
	return json.Marshal(info)
}
