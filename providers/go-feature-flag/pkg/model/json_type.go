package model

type JsonType interface {
	float64 | int64 | string | bool | interface{}
}
