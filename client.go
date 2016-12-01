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

type ProgressCallback func(int64)

func NoopProgressCallback(copied int64) { }

type ProgressReader struct {
	reader io.Reader
	callback ProgressCallback
	size int64
}

func (r *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 {
		r.size += int64(n)
		r.callback(r.size)
	}
	return n, err
}

type Client struct {
    config *ClientConfig
    httpClient *http.Client
	Callback ProgressCallback
	StartTime time.Time
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
    client := &Client{config:config, httpClient:httpclient, Callback: NoopProgressCallback}
	return client
}

func (client *Client) doQueryFileInfo(filename string) (int64, error) {
	queryurlstr := "http://" + client.config.RemoteAddr + "/info?name=" + filename
	resp, err := client.httpClient.Get(queryurlstr)
	log.Println("resp get:", resp, err)
	defer resp.Body.Close()
	// contentLen, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	// log.Println("resp content length:", resp, err, contentLen)
	// if err != nil {
	// 	return -1, err
	// }
	// if contentLen > 16*1024 || contentLen <= 0 {
	// 	return -1, errors.New(log.Sprintf("content-length is invalid: %v", contentLen))
	// }
	info, err := decodeFileInfo(resp.Body)
	log.Println("decodeFileInfo", info, err)
	if info.Code != 0 {
		log.Println("doQueryFileInfo fail", info.Code, info.Message, info, err)
		return 0, errors.New(info.Message)
	}
	return info.Size, nil
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
	puturlstr := "http://" + client.config.RemoteAddr + "/put?name=" + filename
	req, err := http.NewRequest("POST", puturlstr, &ProgressReader{reader:fin, callback:client.Callback, size:0})
    if err != nil {
   		return err
    }
	fi, err := fin.Stat()
	totalsize := fi.Size()
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %v-%v/%v", startpos, totalsize - 1, totalsize))
	req.Header.Set("Content-Length", strconv.FormatInt(totalsize - startpos, 10))
	resp, err := client.httpClient.Do(req)
	log.Println("doUploadFile resp:", resp, err)
	defer resp.Body.Close()
	info, err := decodeFileInfo(resp.Body)
	log.Println("decodeFileInfo", info, err)
	if info.Code != 0 {
		log.Println("decodeFileInfo fail", info.Code, info.Message, info, err)
		return errors.New(info.Message)
	}
	log.Println("file is uploaded", filepath, err)
	return err
}

func (client *Client) SendFile(filepath string) error {
	filename := path.Base(filepath)
	client.StartTime = time.Now()
	filesize, err := client.doQueryFileInfo(filename)
	log.Println("query file info", filepath, filesize, err)
	err = client.doUploadFile(filepath, filename, filesize)
	log.Println("upload file", filepath, err)
	return err
}

func CopyIO(dst io.Writer, src io.Reader, callback ProgressCallback, bufsize int) (written int64, err error) {
	buf := make([]byte, bufsize)
	copied := int64(0)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				copied += int64(nw)
				log.Println("CopyIO", nr, nw, copied)
				callback(copied)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return copied, err
}

/*
func SendFile(config *ClientConfig, block kcp.BlockCrypt, callback ProgressCallback) (*Client, error) {
    log.Println("SendFile", config.Url)

	if config.RemoteAddr == "" {
		log.Println("RemoteAddr is empty", config.Input)
		return nil, errors.New("RemoteAddr is empty")
	}
	var fin *os.File
	var err error
	if config.Input != "" {
		fin, err = os.Open(config.Input)
		if err != nil {
			log.Println("failed to open input file", err)
			return nil, err
		}
	} else {
		log.Println("no input")
		return nil, errors.New("no input")
	}
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
	filename := path.Base(config.Input)
	fi, err := fin.Stat()
	// queryurlstr := "http://" + config.RemoteAddr + "/info?name=" + filename
	puturlstr := "http://" + config.RemoteAddr + "/put?name=" + filename
	httpclient := &http.Client{Transport: transport}
	log.Println("client:", httpclient, config.Url)
	startTime := time.Now()
	resp, err := httpclient.Post(puturlstr, "application/octet-stream", fin)
	log.Println("resp:", resp, err)
	defer resp.Body.Close()
	written, err := CopyIO(os.Stdout, resp.Body, callback, 32*1024)
	usedTime := time.Since(startTime)
	log.Println("get time used", usedTime, written, fi.Size())
	client.httpClient = httpclient
	return client, err
}
*/
