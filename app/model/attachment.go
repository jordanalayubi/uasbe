package model

type Attachment struct {
	FileName string `bson:"file_name" json:"file_name"`
	FileURL  string `bson:"file_url" json:"file_url"`
	FileType string `bson:"file_type" json:"file_type"`
	FileSize int64  `bson:"file_size" json:"file_size"`
}