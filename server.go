package kdt

import (
    "log"
	"fmt"
    "io"
    "net"
    "os"
    "time"
    "path"
    "net/http"
    "net/url"
	"encoding/json"
	"strconv"
)

type Server struct {
    config *ServerConfig
    kcpListener *KCPStreamListener
    tcpListener net.Listener
	root string
}

func sendResponse(writer http.ResponseWriter, body interface{}) {
	buf, _ := json.Marshal(body)
	bodystr := string(buf)
	log.Println("sendResponse", bodystr)
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Length", strconv.Itoa(len(bodystr)))
	writer.WriteHeader(200)
	io.WriteString(writer, bodystr)
}

func (server *Server) Close() {
	err := server.kcpListener.Close()
	log.Println("Server.Close", err)
}

func (server *Server) handleUploadInit(writer http.ResponseWriter, req *http.Request) {
    log.Println("handleUploadInit", req)
	params, err := url.ParseQuery(req.URL.RawQuery)
	filename := params.Get("name")
	transferid := params.Get("transferid")
	if filename == "" || transferid != server.config.TransferID {
    	log.Println("handleUploadInit invalid params", filename, transferid, server.config.TransferID)
		writer.WriteHeader(500)
		return
	}
    log.Println("handleUploadInit file", filename, err)
	filepath := path.Join(server.config.Root, filename)
	fi, err := os.Stat(filepath)
	if fi != nil {
		if fi.IsDir() {
    		log.Println("handleUploadInit filename is dir", filepath, err, fi.Size())
			body := &FileInfoResponse{Code:1, Message:"file is dir", Finished:false, Name:filename, Size:0}
			sendResponse(writer, body)
			return
		}
    	log.Println("handleUploadInit file is finished", filepath, err, fi.Size())
		body := &FileInfoResponse{Code:1, Message:"file is already uploaded", Finished:true, Name:filename, Size:fi.Size()}
		sendResponse(writer, body)
		return
	}
	pendingFilepath := filepath + ".kdt!"
	fi, err = os.Stat(pendingFilepath)
	if fi != nil {
		if fi.IsDir() {
			log.Println("handleUploadInit pending filename is dir", pendingFilepath, err, fi.Size())
			body := &FileInfoResponse{Code:1, Message:"pending file is dir", Finished:false, Name:filename, Size:0}
			sendResponse(writer, body)
			return
		}
		log.Println("handleUploadInit file is pending", filepath, err, fi.Size())
		body := &FileInfoResponse{Code:0, Message:"file is partially uploaded", Finished:false, Name:filename, Size:fi.Size()}
		sendResponse(writer, body)
		return
	}
	log.Println("handleUploadInit file is pending", filepath, err, 0)
	body := &FileInfoResponse{Code:0, Message:"new file", Finished:false, Name:filename, Size:0}
	sendResponse(writer, body)
}

func (server *Server) handleUpload(writer http.ResponseWriter, req *http.Request) {
    log.Println("handleUpload", writer, req)
	startTime := time.Now()
	params, err := url.ParseQuery(req.URL.RawQuery)
	filename := params.Get("name")
	transferid := params.Get("transferid")
	if filename == "" || transferid != server.config.TransferID {
    	log.Println("handleUpload invalid params", filename, transferid, server.config.TransferID)
		writer.WriteHeader(500)
		return
	}
    log.Println("handleUpload file", filename, err)
	filepath := path.Join(server.config.Root, filename)
	fi, err := os.Stat(filepath)
	if fi != nil {
		if fi.IsDir() {
			log.Println("handleUpload filename is dir", filepath, err, fi.Size())
			body := &FileInfoResponse{Code:1, Message:"file is dir", Finished:false, Name:filename, Size:0}
			sendResponse(writer, body)
			return
		}
    	log.Println("handleUpload filesize", filepath, fi.Size(), err)
		body := &FileInfoResponse{Code:1, Message:"file is already existing", Finished:true, Name:filename, Size:fi.Size()}
		sendResponse(writer, body)
		return
	}
	pendingFilepath := filepath + ".kdt!"
	fi, err = os.Stat(pendingFilepath)
	var curfilesize int64 = 0
	if fi != nil {
		if fi.IsDir() {
			log.Println("handleUpload pending filename is dir", pendingFilepath, err, fi.Size())
			body := &FileInfoResponse{Code:1, Message:"file is dir", Finished:false, Name:filename, Size:0}
			sendResponse(writer, body)
			return
		}
    	log.Println("handleUpload filesize", filepath, fi.Size(), err)
		// body := &FileInfoResponse{Code:1, Message:"file is already existing", Finished:true, Name:filename, Size:fi.Size()}
		// sendResponse(writer, body)
		// return
		curfilesize = fi.Size()
	}
	log.Println("handleUpload file does not exist", filepath, err)
	fout, err := os.OpenFile(pendingFilepath, os.O_WRONLY | os.O_CREATE, 0666)
	if fout == nil {
    	log.Println("handleUpload failed to open pending file", pendingFilepath, err)
		body := &FileInfoResponse{Code:1, Message:"failed to open pending file", Finished:false, Name:filename, Size:0}
		sendResponse(writer, body)
		return
	}
	var body *FileInfoResponse = nil 
	finished := false
	{
		defer fout.Close()
		s := req.Header.Get("Content-Range")
		var startpos, endpos, totalsize int64 = 0, 0, 0
		_, err = fmt.Sscanf(s, "bytes %d-%d/%d", &startpos, &endpos, &totalsize)
		if err != nil {
			log.Println("handleUpload no valid content-range", s, err)
			body := &FileInfoResponse{Code:1, Message:"no valid content-range", Name:filename, Size:0}
			sendResponse(writer, body)
			return
		}
		curpos, err := fout.Seek(0, os.SEEK_END)
		if err != nil {
			log.Println("handleUpload failed to get current pos", err)
			body := &FileInfoResponse{Code:1, Message:"failed to get current pos", Name:filename, Size:0}
			sendResponse(writer, body)
			return
		}
		if curpos != curfilesize {
			log.Println("handleUpload invalid current pos", err, s, curpos, curfilesize, startpos)
			body := &FileInfoResponse{Code:1, Message:"invalid pos", Name:filename, Size:0}
			sendResponse(writer, body)
			return
		}
		if startpos != curpos {
			log.Println("handleUpload invalid start pos", err, s, curpos, curfilesize, startpos)
			body := &FileInfoResponse{Code:1, Message:"invalid start pos", Name:filename, Size:curpos}
			sendResponse(writer, body)
			return
		}
		copied, err := io.Copy(fout, req.Body)
		log.Println("handleUpload file create", pendingFilepath, copied, err)
		if copied < 0 {
			errmsg := "no error"
			if err != nil {
				errmsg = err.Error()
			}
			body = &FileInfoResponse{Code:1, Message:"failed to receive data:" + errmsg, Finished:false, Name:filename, Size:startpos}
			sendResponse(writer, body)
			return
		}
		newsize := copied + startpos
		finished = newsize == totalsize
		codeval := 1
		msg := "file is partially uploaded"
		if finished {
			codeval = 0
			msg = "file is all uploaded"
		}
		body = &FileInfoResponse{Code:codeval, Message:msg, Finished:finished, Name:filename, Size:newsize}
		sendResponse(writer, body)
	}
	if finished {
		err = os.Rename(pendingFilepath, filepath)
		log.Println("file is saved, rename", filepath, time.Since(startTime), err)
	}
}

func ReceiveFiles(config *ServerConfig) (*Server, error) {
    block := config.CreateBlockCrypt()
    log.Println("ReceiveFiles", config.Listen)
    slis, err := CreateKCPStreamListener(config, block)
	if err != nil {
		log.Println("failed to create kcp stream listener", err)
		return nil, err
	}
	log.Println("kcp stream listener is created")
    tcpListener, err := net.Listen("tcp", config.Listen)
	if err != nil {
		log.Println("failed to listen on kcp stream", err)
		return nil, err
	}
	log.Println("listen on kcp stream", config.Listen)
    server := &Server{config:config, kcpListener:slis, tcpListener:tcpListener}
	mux := http.NewServeMux()
	// mux.Handle("/", http.FileServer(http.Dir("./")))
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(config.Root))))
	mux.HandleFunc("/api/uploadinit", server.handleUploadInit)
	mux.HandleFunc("/api/upload", server.handleUpload)
	err = http.Serve(slis, mux)
    // err = http.Serve(tcpListener, mux)
	log.Println("ReceiveFiles: start server", config.Listen, err)
    return server, err
}
