package sync

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// SynchronizedBlobRecorder describes a recorder of synchronized blobs
type SynchronizedBlobRecorder struct {
	Blobs  map[string](map[string]int64)
	onDisk *bufio.Writer

	syncC chan int
}

// NewSynchronizedBlobRecorder initialize a SynchronizedBlobRecorder.
func NewSynchronizedBlobRecorder(onDisk string) error {
	if SynchronizedBlobs != nil {
		return nil
	}
	file, err := os.OpenFile(onDisk, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	synchronizedBlobs := &SynchronizedBlobRecorder{
		Blobs:  map[string](map[string]int64){},
		syncC:  make(chan int, 1),
		onDisk: nil,
	}

	// load record file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// each line of the recorder file on disk looks like: "<registry/namespace/repository>,<digest_of_layer>,<layer_size>\n"
		content := strings.Split(line, ",")
		if len(content) != 3 || err != nil {
			// ignore the illegal line of recorder file
			// it will take longer for pulling image because of such illegal recorder lines
			continue
		}

		var size int64
		size, err = strconv.ParseInt(content[2], 10, 64)
		synchronizedBlobs.Record(content[0], content[1], size)
	}
	synchronizedBlobs.onDisk = bufio.NewWriter(file)

	SynchronizedBlobs = synchronizedBlobs
	return nil
}

// Record information of a layer that has been synchronized
func (slr *SynchronizedBlobRecorder) Record(repositoryURL, digest string, size int64) error {
	slr.LockRecorder()
	if slr.Blobs[repositoryURL] == nil {
		slr.Blobs[repositoryURL] = map[string]int64{}
	}

	slr.Blobs[repositoryURL][digest] = size
	if slr.onDisk != nil {
		_, err := slr.onDisk.WriteString(repositoryURL + "," + digest + "," + strconv.FormatInt(size, 10) + "\n")
		if err != nil {
			slr.UnlockRecorder()
			return err
		}
	}
	slr.UnlockRecorder()
	return nil
}

// Query the recorder if a layer has been synchronized
func (slr *SynchronizedBlobRecorder) Query(repositoryURL, digest string) (int64, bool) {
	slr.LockRecorder()
	size, exist := slr.Blobs[repositoryURL][digest]
	slr.UnlockRecorder()
	return size, exist
}

// GetRecords gets records according related to the repositoryURL
func (slr *SynchronizedBlobRecorder) GetRecords(repositoryURL string) map[string]int64 {
	slr.LockRecorder()
	recordList := slr.Blobs[repositoryURL]
	slr.UnlockRecorder()
	return recordList
}

// UpdateRecords updates records related to the repositoryURL
func (slr *SynchronizedBlobRecorder) UpdateRecords(repositoryURL string, recordList map[string]int64) error {
	slr.LockRecorder()
	for key, value := range recordList {
		if slr.Blobs[repositoryURL] == nil {
			slr.Blobs[repositoryURL] = map[string]int64{}
		}

		slr.Blobs[repositoryURL][key] = value
		if slr.onDisk != nil {
			_, err := slr.onDisk.WriteString(repositoryURL + "," + key + "," + strconv.FormatInt(value, 10) + "\n")
			if err != nil {
				slr.UnlockRecorder()
				return err
			}
		}
	}
	slr.UnlockRecorder()
	return nil
}

// Flush records to disk
func (slr *SynchronizedBlobRecorder) Flush() {
	slr.LockRecorder()
	slr.onDisk.Flush()
	slr.UnlockRecorder()
}

// LockRecorder locks the syncC mutex
func (slr *SynchronizedBlobRecorder) LockRecorder() {
	slr.syncC <- 1
}

// UnlockRecorder unlocks the syncC mutex
func (slr *SynchronizedBlobRecorder) UnlockRecorder() {
	<-slr.syncC
}
