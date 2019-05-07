package data

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
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
		d, err := NewBinaryData(v.send, Inbound)
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
	tUUID, err := uuid.MustParse("dabc30c6-440d-43ee-86a6-7c7e374fdd19").MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	testData := []struct {
		send []byte
		want BinaryData
	}{
		{append(tUUID, append([]byte{2}, []byte("hello")...)...),
			BinaryData{
				Traffic:     Outbound,
				UUID:        tUUID,
				MessageType: byte(2),
				Payload:     []byte("hello")},
		},
		{append(tUUID, append([]byte{NewConnect}, []byte("MSG")...)...), BinaryData{
			Traffic:     Outbound,
			UUID:        tUUID,
			MessageType: byte(NewConnect),
			Payload:     []byte("MSG")},
		},
		{append(tUUID, append([]byte{ExitConnect}, []byte("HOGE")...)...), BinaryData{
			Traffic:     Outbound,
			UUID:        tUUID,
			MessageType: byte(ExitConnect),
			Payload:     []byte("HOGE")},
		},
	}
	for _, v := range testData {
		d, err := NewBinaryData(v.send, Outbound)
		if err != nil {
			t.Error(err)
		}

		if !reflect.DeepEqual(d.Payload, v.want.Payload) {
			t.Errorf("missmatch Payload get: %v , want: %v", d.Payload, v.want.Payload)
		}
		if !reflect.DeepEqual(d.MessageType, v.want.MessageType) {
			t.Errorf("missmatch MessageType get: %v , want: %v", d.Payload, v.want.Payload)
		}
		if !reflect.DeepEqual(d.UUID, v.want.UUID) {
			t.Errorf("missmatch UUID get: %v , want: %v", d.Target, v.want.Target)
		}
		if !reflect.DeepEqual(d.Traffic, v.want.Traffic) {
			t.Errorf("missmatch Traffic get: %v , want: %v", d.Traffic, v.want.Traffic)
		}
	}
}
