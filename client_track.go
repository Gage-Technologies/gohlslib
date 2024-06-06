package gohlslib

import (
	"context"
	"fmt"
	"time"
)

type clientTrack struct {
	track                  *Track
	onData                 clientOnDataFunc
	lastAbsoluteTime       time.Time
	startRTC               time.Time
	restrictToViewingSpeed bool
}

func (t *clientTrack) absoluteTime() (time.Time, bool) {
	if t.lastAbsoluteTime == zero {
		return zero, false
	}
	return t.lastAbsoluteTime, true
}

func (t *clientTrack) handleData(
	ctx context.Context,
	pts time.Duration,
	dts time.Duration,
	ntp time.Time,
	data [][]byte,
) error {
	// silently discard packets prior to the first packet of the leading track
	if pts < 0 {
		return nil
	}

	// conditionally synchronize time
	if t.restrictToViewingSpeed {
		elapsed := time.Since(t.startRTC)
		if dts > elapsed {
			diff := dts - elapsed
			if diff > clientMaxDTSRTCDiff {
				return fmt.Errorf("difference between DTS and RTC is too big")
			}

			select {
			case <-time.After(diff):
			case <-ctx.Done():
				return fmt.Errorf("terminated")
			}
		}
	}

	if t.restrictToViewingSpeed {
		t.lastAbsoluteTime = ntp
	} else {
		// handle absolute time if we are decoding faster that viewing speed by
		// accelerating time at the deconding rate from the original ntp provided
		if t.lastAbsoluteTime == zero {
			t.lastAbsoluteTime = ntp
		}
		t.lastAbsoluteTime = t.lastAbsoluteTime.Add(dts)
	}
	t.onData(pts, dts, data)
	return nil
}
