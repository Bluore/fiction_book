package parameter

import (
	"mime/multipart"
)

type UploadImageIn struct {
	Avatar     *multipart.FileHeader
	ImageId    string
	ImagePath  string
	SizeWidth  int
	SizeHeight int
	SizeFile   int64
	AcceptType map[string]bool
}

type UploadImageOut struct {
	FilePath string
}
