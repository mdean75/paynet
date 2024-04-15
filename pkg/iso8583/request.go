package iso8583

import (
	"bytes"
	"encoding/binary"
)

type Request struct {
	MessageType     string
	PrimaryBitmap   string
	SecondaryBitmap string
	ProcessingCode  string
	TransactionAmt  string
	TransmissionDt  string
	//SystemTraceNo   string
	SystemTraceNo  uint64
	LocalTime      string
	LocalDate      string
	ExpiryDate     string
	MerchantType   string
	AcquiringId    string
	RetrievalRefNo string
	CardNo         string
	CardSeqNo      string
	// Add other fields as needed
}

func Encode(msg Request) []byte {
	buf := new(bytes.Buffer)

	// Convert each field to bytes and write to buffer
	binary.Write(buf, binary.BigEndian, []byte(msg.MessageType))
	binary.Write(buf, binary.BigEndian, []byte(msg.PrimaryBitmap))
	binary.Write(buf, binary.BigEndian, []byte(msg.SecondaryBitmap))
	binary.Write(buf, binary.BigEndian, []byte(msg.ProcessingCode))
	binary.Write(buf, binary.BigEndian, []byte(msg.TransactionAmt))
	binary.Write(buf, binary.BigEndian, []byte(msg.TransmissionDt))
	//binary.Write(buf, binary.BigEndian, []byte(msg.SystemTraceNo))
	binary.Write(buf, binary.BigEndian, msg.SystemTraceNo)
	binary.Write(buf, binary.BigEndian, []byte(msg.LocalTime))
	binary.Write(buf, binary.BigEndian, []byte(msg.LocalDate))
	binary.Write(buf, binary.BigEndian, []byte(msg.ExpiryDate))
	binary.Write(buf, binary.BigEndian, []byte(msg.MerchantType))
	binary.Write(buf, binary.BigEndian, []byte(msg.AcquiringId))
	binary.Write(buf, binary.BigEndian, []byte(msg.RetrievalRefNo))
	binary.Write(buf, binary.BigEndian, []byte(msg.CardNo))
	binary.Write(buf, binary.BigEndian, []byte(msg.CardSeqNo))
	// Write other fields as needed

	return buf.Bytes()
}

func EncodeWithHeader(msg Request) []byte {
	payload := Encode(msg)

	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(payload)))
	buf = append(buf, payload...)

	return buf
}

func Decode(buf []byte) Request {
	var msg Request

	msg.MessageType = string(buf[:4])
	msg.PrimaryBitmap = string(buf[4:20])
	msg.SecondaryBitmap = string(buf[20:36])
	msg.ProcessingCode = string(buf[36:42])
	msg.TransactionAmt = string(buf[42:54])
	msg.TransmissionDt = string(buf[54:64])
	msg.SystemTraceNo = binary.BigEndian.Uint64(buf[64:72])
	msg.LocalTime = string(buf[72:78])
	msg.LocalDate = string(buf[78:82])
	msg.ExpiryDate = string(buf[82:86])
	msg.MerchantType = string(buf[86:90])
	msg.AcquiringId = string(buf[90:100])
	msg.RetrievalRefNo = string(buf[100:112])
	msg.CardNo = string(buf[112:128])
	msg.CardSeqNo = string(buf[128:131])

	return msg
}
