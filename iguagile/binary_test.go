package iguagile

import (
	"reflect"
	"testing"
)

func TestInbound(t *testing.T) {
	testData := []struct {
		send []byte
		want BinaryData
	}{
		{append([]byte{1, 2}, []byte("hello")...),
			BinaryData{
				Traffic:     Inbound,
				Target:      byte(1),
				MessageType: byte(2),
				Payload:     []byte("hello")},
		},
		{append([]byte{1, 3}, []byte("MSG")...), BinaryData{
			Traffic:     Inbound,
			Target:      byte(1),
			MessageType: byte(3),
			Payload:     []byte("MSG")},
		},
		{append([]byte{1, 4}, []byte("HOGE")...), BinaryData{
			Traffic:     Inbound,
			Target:      byte(1),
			MessageType: byte(4),
			Payload:     []byte("HOGE")},
		},
	}
	for _, v := range testData {
		d, err := NewInBoundData(v.send)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(d.Payload, v.want.Payload) {
			t.Errorf("missmatch Payload get: %v , want: %v", d.Payload, v.want.Payload)
		}
		if !reflect.DeepEqual(d.MessageType, v.want.MessageType) {
			t.Errorf("missmatch MessageType get: %v , want: %v", d.Payload, v.want.Payload)
		}
		if !reflect.DeepEqual(d.Target, v.want.Target) {
			t.Errorf("missmatch Target get: %v , want: %v", d.Target, v.want.Target)
		}
		if !reflect.DeepEqual(d.Traffic, v.want.Traffic) {
			t.Errorf("missmatch Traffic get: %v , want: %v", d.Traffic, v.want.Traffic)
		}
	}
}

func TestOutbound(t *testing.T) {
	cid := make([]byte, 2)
	cid[0] = 1
	cid[1] = 2

	testData := []struct {
		send []byte
		want BinaryData
	}{
		{append(cid, append([]byte{2}, []byte("hello")...)...),
			BinaryData{
				Traffic:     Outbound,
				ID:          cid,
				MessageType: byte(2),
				Payload:     []byte("hello")},
		},
		{append(cid, append([]byte{NewConnect}, []byte("MSG")...)...), BinaryData{
			Traffic:     Outbound,
			ID:          cid,
			MessageType: byte(NewConnect),
			Payload:     []byte("MSG")},
		},
		{append(cid, append([]byte{ExitConnect}, []byte("HOGE")...)...), BinaryData{
			Traffic:     Outbound,
			ID:          cid,
			MessageType: byte(ExitConnect),
			Payload:     []byte("HOGE")},
		},
	}
	for _, v := range testData {
		d, err := NewOutBoundData(v.send)
		if err != nil {
			t.Error(err)
		}

		if !reflect.DeepEqual(d.Payload, v.want.Payload) {
			t.Errorf("missmatch Payload get: %v , want: %v", d.Payload, v.want.Payload)
		}
		if !reflect.DeepEqual(d.MessageType, v.want.MessageType) {
			t.Errorf("missmatch MessageType get: %v , want: %v", d.Payload, v.want.Payload)
		}
		if !reflect.DeepEqual(d.ID, v.want.ID) {
			t.Errorf("missmatch UUID get: %v , want: %v", d.Target, v.want.Target)
		}
		if !reflect.DeepEqual(d.Traffic, v.want.Traffic) {
			t.Errorf("missmatch Traffic get: %v , want: %v", d.Traffic, v.want.Traffic)
		}
	}
}
