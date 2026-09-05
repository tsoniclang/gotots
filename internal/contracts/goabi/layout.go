package goabi

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/contracts/environment"
)

const Module = "@gotots/abi/layout.js"

type Layout uint8

const (
	Invalid Layout = iota
	Little32
	Little64
	Big32
	Big64
)

func Select(order environment.ByteOrder, pointerBytes int64) (Layout, error) {
	if order == environment.ByteOrderLittleEndian {
		switch pointerBytes {
		case 4:
			return Little32, nil
		case 8:
			return Little64, nil
		}
	}
	if order == environment.ByteOrderBigEndian {
		switch pointerBytes {
		case 4:
			return Big32, nil
		case 8:
			return Big64, nil
		}
	}
	return Invalid, fmt.Errorf("unsupported explicit Go ABI: byte order %d, pointer bytes %d", order, pointerBytes)
}

func (layout Layout) Export() (string, error) {
	switch layout {
	case Little32:
		return "little32", nil
	case Little64:
		return "little64", nil
	case Big32:
		return "big32", nil
	case Big64:
		return "big64", nil
	default:
		return "", fmt.Errorf("invalid Go ABI layout %d", layout)
	}
}
