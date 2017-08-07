package kdt

import (
	"fmt"
    "log"
    "io"
    "net"
    "os"
    "time"
    "path"
	"errors"
	"strconv"
    "net/http"
)

type ProgressReader struct {
	reader io.Reader
	callback ProgressCallback
	size int64
	offset int64
	total int64
	bandwidth int64
	starttime time.Time
}

func (r *ProgressReader) Read(p []byte) (n int, err error) {
	var delayTime int
	if r.bandwidth > 0 {
		delayTime = int(float64(1000000 * 32 * 1024 * 8) / float64(r.bandwidth))
		usedTime := float64(time.Since(r.starttime)) / float64(time.Second)
		if usedTime == 0 {
			usedTime = 1
		}
		speed := float64(r.size * 8) / usedTime

		var ratio = float64(speed) / float64(r.bandwidth)
		if ratio > 1.05 {
			delayTime += 10000;
		} else if ratio < 0.95 && delayTime >= 10000 {
			delayTime -= 10000;
		} else if ratio <  0.95 {
			delayTime = 0
		}
		time.Sleep(time.Duration(delayTime) * time.Microsecond)
	}
	n, err = r.reader.Read(p)
	if n > 0 {
		r.size += int64(n)
		r.callback(r.starttime, r.offset, r.size, r.total)
	}
	return n, err
}

type Client struct {
    config *ClientConfig
	transport *http.Transport
    httpClient *http.Client
	request *http.Request
	Callback ProgressCallback
}

func CreateClient(config *ClientConfig) *Client {
	block := config.CreateBlockCrypt()
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// DialContext: (&net.Dialer{
		// 	Timeout:   30 * time.Second,
		// 	KeepAlive: 30 * time.Second,
		// }).DialContext,
		Dial: func (network, addr string) (net.Conn, error) {
			return dialKcp(config, block)
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	httpclient := &http.Client{Transport: transport}
    client := &Client{config:config, httpClient:httpclient, Callback: NoopProgressCallback, transport: transport}
	return client
}

func (client *Client) doInitUpload(filename string) (*FileInfoResponse, error) {
	queryurlstr := "http://" + client.config.RemoteAddr + "/api/uploadinit?name=" + filename + "&transferid=" + client.config.TransferID
	resp, err := client.httpClient.Get(queryurlstr)
	log.Println("resp get:", resp, err)
	if err != nil {
		return nil, err 
	}
	defer resp.Body.Close()
	// contentLen, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	// log.Println("resp content length:", resp, err, contentLen)
	// if err != nil {
	// 	return -1, err
	// }
	// if contentLen > 16*1024 || contentLen <= 0 {
	// 	return -1, errors.New(log.Sprintf("content-length is invalid: %v", contentLen))
	// }
	// tempbuf := make([]byte, 32*1024)
	// readsize, err := io.ReadFull(resp.Body, tempbuf)
	// content := string(tempbuf[:readsize])
	// log.Println("decodeFileInfo content", content)
	info, err := decodeFileInfo(resp.Body)
	log.Println("decodeFileInfo", info, err)
	if err != nil || info == nil {
		log.Println("doInitUpload failed with error", info, err)
		if err == nil {
			err = errors.New("failed to get file info")
		}
		return nil, err
	}
	if info.Code != 0 {
		log.Println("doInitUpload failed with info", info.Code, info.Message, info.Name, info.Size, info, err)
		return info, errors.New("doInitUpload failed:" + info.Message)
	}
	return info, nil
}

func (client *Client) doUploadFile(filepath, filename string, startpos int64) error {
	log.Println("doUploadFile", filepath, filename, startpos)
	fin, err := os.Open(filepath)
	if err != nil {
		log.Println("failed to open input file", err)
		return err
	}
	_, err = fin.Seek(startpos, os.SEEK_SET)
	log.Println("doUploadFile seek", filepath, startpos, err)
	if err != nil {
		return err
	}
	puturlstr := "http://" + client.config.RemoteAddr + "/api/upload?name=" + filename + "&transferid=" + client.config.TransferID
	reader := &ProgressReader{reader:fin, callback:client.Callback, size:0, offset:startpos}
	req, err := http.NewRequest("POST", puturlstr, reader)
    if err != nil {
		log.Println("failed to create http request", err, puturlstr)
   		return err
    }
	client.request = req
	fi, err := fin.Stat()
	if err != nil {
		log.Println("failed to get file stat", err, filepath)
		return err
	}
	totalsize := fi.Size()
	reader.total = totalsize
	reader.starttime = time.Now()
	reader.bandwidth = int64(client.config.Bandwidth * 1024)
	contentrange := fmt.Sprintf("bytes %v-%v/%v", startpos, totalsize - 1, totalsize)
	contentlen := strconv.FormatInt(totalsize - startpos, 10)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Range", contentrange)
	req.Header.Set("Content-Length", contentlen)
	log.Println("before upload: contentrange/contentlen", contentrange, contentlen)
	resp, err := client.httpClient.Do(req)
	log.Println("doUploadFile resp:", resp, err)
	if err != nil {
		log.Println("failed to execute http request", err, resp, puturlstr)
		return err
	}
	defer resp.Body.Close()
	info, err := decodeFileInfo(resp.Body)
	log.Println("decodeFileInfo", info, err)
	if info.Code != 0 || err != nil{
		log.Println("decodeFileInfo fail", info.Code, info.Message, info, err)
		return errors.New(info.Message)
	}
	log.Println("file is uploaded", filepath, err)
	return err
}

func (client *Client) Close() {
	client.transport.CancelRequest(client.request)
}

func (client *Client) SendFile(filepath string) error {
	filename := path.Base(filepath)
	info, err := client.doInitUpload(filename)
	log.Println("query file info result:", filepath, info, err)
	// if info == nil || err != nil {
	if info == nil {
		msg := "query file info no info"
		log.Println(msg, filepath, info, err)
		if err != nil {
			msg = msg + ": " + err.Error()
		}
		return errors.New(msg)
	}
	if info.Finished {
		log.Println("file is already uploaded", filepath, info.Size, err)
		return errors.New("file is already uploaded")
	}
	log.Println("start upload", filepath, info, err)
	err = client.doUploadFile(filepath, filename, info.Size)
	log.Println("upload file", filepath, err)
	return err
}
