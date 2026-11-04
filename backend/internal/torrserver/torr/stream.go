package torr

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/anacrolix/missinggo/v2/httptoo"
	"github.com/anacrolix/torrent"

	mt "plenka/internal/torrserver/mimetype"
	sets "plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr/state"
)

// Add atomic counter for concurrent streams
var activeStreams int32

func (t *Torrent) Stream(fileID int, req *http.Request, resp http.ResponseWriter) error {
	// Increment active streams counter
	streamID := atomic.AddInt32(&activeStreams, 1)
	defer atomic.AddInt32(&activeStreams, -1)
	// Stream disconnect timeout (same as torrent)
	streamTimeout := sets.BTsets.TorrentDisconnectTimeout

	if !t.GotInfo() {
		http.NotFound(resp, req)
		return errors.New("torrent doesn't have info yet")
	}
	// Get file information
	st := t.Status()
	var stFile *state.TorrentFileStat
	for _, fileStat := range st.FileStats {
		if fileStat.Id == fileID {
			stFile = fileStat
			break
		}
	}
	// The handler has already checked the request, so every failure from here on must
	// still answer it: returning without a write made gin send an empty 200 OK, which
	// players read as a zero-length file.
	if stFile == nil {
		err := fmt.Errorf("file with id %v not found", fileID)
		http.Error(resp, err.Error(), http.StatusNotFound)
		return err
	}
	// Find the actual torrent file
	files := t.Files()
	var file *torrent.File
	for _, tfile := range files {
		if tfile.Path() == stFile.Path {
			file = tfile
			break
		}
	}
	if file == nil {
		err := fmt.Errorf("file with id %v not found", fileID)
		http.Error(resp, err.Error(), http.StatusNotFound)
		return err
	}
	// Create reader with context for timeout
	reader := t.NewReader(file)
	if reader == nil {
		// NewReader returns nil only for a torrent closed after the GotInfo check above
		err := errors.New("cannot create torrent reader")
		http.Error(resp, err.Error(), http.StatusServiceUnavailable)
		return err
	}
	// Ensure reader is always closed
	defer t.CloseReader(reader)

	if sets.BTsets.ResponsiveMode {
		reader.SetResponsive()
	}
	// Log connection
	host, port, clerr := net.SplitHostPort(req.RemoteAddr)

	if sets.BTsets.EnableDebug {
		if clerr != nil {
			log.Printf("[Stream:%d] Connect client (Active streams: %d)", streamID, atomic.LoadInt32(&activeStreams))
		} else {
			log.Printf("[Stream:%d] Connect client %s:%s (Active streams: %d)",
				streamID, host, port, atomic.LoadInt32(&activeStreams))
		}
	}

	// Set response headers
	resp.Header().Set("Connection", "close")
	// Add timeout header if configured
	if streamTimeout > 0 {
		resp.Header().Set("X-Stream-Timeout", fmt.Sprintf("%d", streamTimeout))
	}
	// Add ETag
	etag := hex.EncodeToString([]byte(fmt.Sprintf("%s/%s", t.Hash().HexString(), file.Path())))
	resp.Header().Set("ETag", httptoo.EncodeQuotedString(etag))
	// add MimeType
	mime, err := mt.MimeTypeByPath(file.Path())
	if err == nil && mime.IsMedia() {
		resp.Header().Set("content-type", mime.String())
	}
	// Add support for range requests
	if req.Header.Get("Range") != "" {
		resp.Header().Set("Accept-Ranges", "bytes")
	}
	http.ServeContent(resp, req, file.Path(), time.Unix(t.Timestamp, 0), newBufferedStreamReader(reader, 1<<20))

	if sets.BTsets.EnableDebug {
		if clerr != nil {
			log.Printf("[Stream:%d] Disconnect client", streamID)
		} else {
			log.Printf("[Stream:%d] Disconnect client %s:%s", streamID, host, port)
		}
	}
	return nil
}
