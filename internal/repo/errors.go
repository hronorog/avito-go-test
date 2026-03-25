package repo

import "errors"

var (
    ErrSlotNotFree  = errors.New("slot not free")
    ErrSlotNotFound = errors.New("slot not found")
)
