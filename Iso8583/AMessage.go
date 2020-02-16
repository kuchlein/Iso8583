package Iso8583

import (
	"bytes"
	"errors"
	"fmt"
)

type CreateFieldFunc func(int) IField

type AMessage struct {
	MTI                 *string
	Bitmap              *Bitmap
	MsgTemplate         *Template
	Fields              map[int]IField
	CreateFieldCallback CreateFieldFunc
}

func NewAMessage(mti *string, tmpl *Template) *AMessage {
	msg := &AMessage{MsgTemplate: tmpl, MTI: mti, Fields: make(map[int]IField), Bitmap: NewBitmap(tmpl.BitmapFormatter)}
	msg.CreateFieldCallback = msg.CreateField

	return msg
}

func (msg *AMessage) PackedLength() int {
	length := 0
	if msg.MTI != nil {
		length += len(*msg.MTI)
	}
	length += msg.Bitmap.PackedLength()
	for i := 2; i < 128; i++ {
		if msg.Bitmap.IsFieldSet(i) {
			length += msg.Fields[i].PackedLength()
		}
	}

	return length
}

func (msg *AMessage) ClearField(field int) {
	msg.Bitmap.SetField(field, false)
	delete(msg.Fields, field)
}

func (msg *AMessage) IsFieldSet(field int) bool {
	return msg.Bitmap.IsFieldSet(field)
}

func (msg *AMessage) ToMsg() []byte {

	packedLength := msg.PackedLength()
	data := make([]byte, packedLength)
	offset := 0
	if msg.MTI != nil {
		copy(data[offset:], *msg.MTI)
		offset += len(*msg.MTI)
	}
	bmap := msg.Bitmap.ToMsg()
	copy(data[offset:], bmap)
	offset += msg.Bitmap.PackedLength()
	for i := 2; i < 128; i++ {
		if msg.Bitmap.IsFieldSet(i) {
			field := msg.Fields[i]
			copy(data[offset:], field.ToMsg())
			offset += field.PackedLength()
		}
	}

	return data
}

func (msg *AMessage) String() string {
	return msg.ToString("   ")
}

func (msg *AMessage) ToString(prefix string) string {
	var buffer bytes.Buffer
	if msg.MTI != nil {
		buffer.WriteString(prefix + "MTI" + *msg.MTI + "\n")
	}
	for i := 2; i < 128; i++ {
		if msg.Bitmap.IsFieldSet(i) {
			buffer.WriteString(msg.FieldsToString(i, prefix) + "\n")
		}
	}

	return buffer.String()
}

func (msg *AMessage) FieldsToString(field int, prefix string) string {
	return msg.Fields[field].ToString(prefix)
}

func (msg *AMessage) CreateField(field int) IField {

	if _, ok := msg.MsgTemplate.templateDefinition[field]; ok {
		return NewField(field, msg.MsgTemplate.templateDefinition[field])
	}

	return nil
}

func (msg *AMessage) GetField(field int) (IField, error) {

	_, ok := msg.Fields[field]
	if (!msg.Bitmap.IsFieldSet(field)) || (! ok) {
		if msg.Fields[field] = msg.CreateFieldCallback(field); msg.Fields[field] != nil {
			msg.Bitmap.SetField(field, true)
		} else {
			return nil, errors.New(fmt.Sprintf("Unable to create field number %d. Possibly because template does not have a defination for the field",
				field))
		}
	}

	return msg.Fields[field], nil
}

func (msg *AMessage) Unpack(data []byte, startingOffset int) (int, error) {

	offset := msg.Bitmap.Unpack(data, startingOffset)
	for i := 2; i < 128; i++ {
		if msg.Bitmap.IsFieldSet(i) {
			field, err := msg.GetField(i)
			if err != nil {
				return 0, err
			}
			offset, err = field.Unpack(data, offset)
			if err != nil {
				return 0, err
			}
		}
	}

	return offset, nil
}

func (msg *AMessage) GetFieldValue(field int) string {

	if msg.Bitmap.IsFieldSet(field) {
		return msg.Fields[field].Value()
	}

	return ""
}

func (msg *AMessage) SetFieldValue(field int, value string) error {

	if value == "" {
		msg.ClearField(field)
		return nil
	}

	fld, err := msg.GetField(field)
	if err != nil {
		return err
	}

	fld.SetValue(value)
	return nil
}
