package api

import "github.com/tsoniclang/gotots/internal/contracts/goabi"

type SourceLayoutNames interface {
	SourceDataLayout(goabi.Layout) (NameReference, error)
}
