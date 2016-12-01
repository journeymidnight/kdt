package kdt

import (
    "encoding/json"
    "io"
)

type FileInfoResponse struct {
	Code     int       `json:"code"`
	Message  string    `json:"msg"`
    Finished bool      `json:"finished"`
    Name     string    `json:"name"`
    Size     int64     `json:"size"`
}

func decodeFileInfo(r io.Reader) (*FileInfoResponse, error) {
    fi := &FileInfoResponse{}
    decoder := json.NewDecoder(r)
    err := decoder.Decode(fi)
    if err != nil {
        return nil, err
    }
    return fi, nil
}

func (fi *FileInfoResponse) Encode() string {
    buf, _ := json.Marshal(fi)
    return string(buf)
}
