package sync

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type SyncBlobRecorder struct {
	Blobs  map[string](map[string]int64)
	onDisk *bufio.Writer

	syncC chan int
}

// Initialize a SyncBlobRecorder.
func NewSyncBlobRecorder(onDisk string) error {
	if SynchronizedBlobs != nil {
		return nil
	}
	file, err := os.OpenFile(onDisk, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	synchronizedBlobs := &SyncBlobRecorder{
		Blobs:  map[string](map[string]int64){},
		syncC:  make(chan int, 1),
		onDisk: nil,
	}

	// load record file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// each line of the recorder file on disk looks like: "<registry>,<digest_of_layer>,<layer_size>\n"
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

// Record infomation of a layer that has been synchronized
func (slr *SyncBlobRecorder) Record(registry, digest string, size int64) error {
	slr.LockRecorder()
	if slr.Blobs[registry] == nil {
		slr.Blobs[registry] = map[string]int64{}
	}

	slr.Blobs[registry][digest] = size
	if slr.onDisk != nil {
		_, err := slr.onDisk.WriteString(registry + "," + digest + "," + strconv.FormatInt(size, 10) + "\n")
		if err != nil {
			slr.UnlockRecorder()
			return err
		}
	}
	slr.UnlockRecorder()
	return nil
}

// Query the recorder if a layer has been synchronized
func (slr *SyncBlobRecorder) Query(registry, digest string) (int64, bool) {
	slr.LockRecorder()
	size, exist := slr.Blobs[registry][digest]
	slr.UnlockRecorder()
	return size, exist
}

func (slr *SyncBlobRecorder) GetRegistryRecords(registry string) map[string]int64 {
	slr.LockRecorder()
	recordList := slr.Blobs[registry]
	slr.UnlockRecorder()
	return recordList
}

func (slr *SyncBlobRecorder) UpdateRegistryRecords(registry string, recordList map[string]int64) error {
	slr.LockRecorder()
	for key, value := range recordList {
		if slr.Blobs[registry] == nil {
			slr.Blobs[registry] = map[string]int64{}
		}

		slr.Blobs[registry][key] = value
		if slr.onDisk != nil {
			_, err := slr.onDisk.WriteString(registry + "," + key + "," + strconv.FormatInt(value, 10) + "\n")
			if err != nil {
				slr.UnlockRecorder()
				return err
			}
		}
	}
	slr.UnlockRecorder()
	return nil
}

func (slr *SyncBlobRecorder) Flush() {
	slr.LockRecorder()
	slr.onDisk.Flush()
	slr.UnlockRecorder()
}

func (slr *SyncBlobRecorder) LockRecorder() {
	slr.syncC <- 1
}

func (slr *SyncBlobRecorder) UnlockRecorder() {
	<-slr.syncC
}
