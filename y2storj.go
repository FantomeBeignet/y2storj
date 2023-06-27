package y2storj

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
	"storj.io/uplink"
)

type Location struct {
	bucket string
	key    string
}

type progress struct {
	contentLength     float64
	totalWrittenBytes float64
	downloadLevel     float64
}

func (dl *progress) Write(p []byte) (n int, err error) {
	n = len(p)
	dl.totalWrittenBytes = dl.totalWrittenBytes + float64(n)
	currentPercent := (dl.totalWrittenBytes / dl.contentLength) * 100
	if (dl.downloadLevel <= currentPercent) && (dl.downloadLevel < 100) {
		dl.downloadLevel++
	}
	return
}

func validateRemoteLocation(loc string) (*Location, error) {
	if strings.HasPrefix(loc, "sj://") {
		trimmed := loc[5:]                     // remove the scheme
		idx := strings.IndexByte(trimmed, '/') // find the bucket index

		// handles sj:// or sj:///foo
		if len(trimmed) == 0 || idx == 0 {
			return nil, errors.New("invalid path: empty bucket in path")
		}

		var bucket, key string
		if idx == -1 { // handles sj://foo
			bucket, key = trimmed, ""
		} else { // handles sj://foo/bar
			bucket, key = trimmed[:idx], trimmed[idx+1:]
		}

		return &Location{bucket: bucket, key: key}, nil
	}
	return nil, errors.New("invalid remote location")
}

func DownloadAndStore(url, location, grant, quality string) error {
	client := youtube.Client{}
	// get the youtube video from the url
	video, err := client.GetVideo(url)
	if err != nil {
		return err
	}
	// get the requested quality
	formats := video.Formats.FindByQuality(quality)
	// get the byte stream
	stream, size, err := client.GetStream(video, formats)
	if err != nil {
		return err
	}
	// check storj location is valid
	loc, err := validateRemoteLocation(location)
	if err != nil {
		return err
	}
	// parse the provided access grant
	access, err := uplink.ParseAccess(grant)
	// open the project from the grant
	project, err := uplink.OpenProject(context.Background(), access)
	if err != nil {
		return err
	}
	defer project.Close()
	// ensure bucket exists
	_, err = project.EnsureBucket(context.Background(), loc.bucket)
	if err != nil {
		return nil
	}
	// get byte stream to upload to
	upload, err := project.UploadObject(context.Background(), loc.bucket, loc.key, nil)
	if err != nil {
		return nil
	}

	// set metadata from youtube
	upload.SetCustomMetadata(context.Background(), uplink.CustomMetadata{
		"OriginalTitle": video.Title,
		"Author":        video.Author,
		"UploadDate":    video.PublishDate.Format(time.DateOnly),
	})

	// progress bar shenanigans
	prog := &progress{
		contentLength: float64(size),
	}
	progress := mpb.New(mpb.WithWidth(64))
	bar := progress.AddBar(
		int64(prog.contentLength),

		mpb.PrependDecorators(
			decor.CountersKibiByte("% .2f / % .2f"),
			decor.Percentage(decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.Name(" | "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
	)

	// more progress bar shenanigans
	reader := bar.ProxyReader(stream)
	mw := io.MultiWriter(upload, prog)
	_, err = io.Copy(mw, reader)
	if err != nil {
		return nil
	}
	// commit the upload transaction
	if err := upload.Commit(); err != nil {
		return err
	}
	return nil
}