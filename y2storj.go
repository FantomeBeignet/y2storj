package y2storj

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"github.com/wader/goutubedl"
	"storj.io/uplink"
)

type Location struct {
	bucket string
	key    string
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
	goutubedl.Path = "yt-dlp"
	result, err := goutubedl.New(
		context.Background(),
		url,
		goutubedl.Options{},
	)
	if err != nil {
		return err
	}
	downloadResult, err := result.Download(context.Background(), quality)
	defer downloadResult.Close()
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
		"URL":           result.Info.URL,
		"OriginalTitle": result.Info.Title,
		"Author":        result.Info.Creator,
		"UploadDate":    result.Info.ReleaseDate,
	})

	// progress bar shenanigans
	progress := mpb.New(
		mpb.WithWidth(10),
	)
	s := mpb.SpinnerStyle(
		"( ●    )",
		"(  ●   )",
		"(   ●  )",
		"(    ● )",
		"(     ●)",
		"(    ● )",
		"(   ●  )",
		"(  ●   )",
		"( ●    )",
		"(●     )",
	)
	bar := progress.New(
		int64(0),
		s,
		mpb.BarRemoveOnComplete(),
		mpb.AppendDecorators(
			decor.CurrentKibiByte("% .2f"),
			decor.Name(" | "),
			decor.Elapsed(decor.ET_STYLE_GO),
		),
	)
	bar.EnableTriggerComplete()

	// more progress bar shenanigans
	writer := bar.ProxyWriter(upload)
	_, err = io.Copy(writer, downloadResult)
	if err != nil {
		return nil
	}
	// commit the upload transaction
	if err := upload.Commit(); err != nil {
		return err
	}
	bar.SetTotal(-1, true)
	return nil
}
