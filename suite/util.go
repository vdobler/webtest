// Some uit functions

package suite

import (
	"fmt"
	"os"
	"strings"
)

// Return filesize of file path, -1 on error and 0 for nonexisting files.
func filesize(path string) (s int64, err error) {
	file, err := os.Open(path)
	if err != nil {
		s = -1
		return
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		s = -1
		return
	}
	return fi.Size(), nil
}

func determinLogfileSize(logs []LogCondition, test *Test) map[string]int64 {
	if len(logs) == 0 {
		return nil
	}
	logfilesize := make(map[string]int64, len(logs))
	for _, log := range logs {
		if _, ok := logfilesize[log.Path]; ok {
			continue
		}
		s, err := filesize(log.Path)
		tracef("Filesize of %s = %d", log.Path, logfilesize[log.Path])
		if err != nil {
			test.Error(log.Id, "Cannot read "+log.Path, err.Error())
		}
		logfilesize[log.Path] = s
	}
	return logfilesize
}

// Sanitize t (by replacing anything uncomfortable in a filename) by _.
// The default output path is prepended automatically.
func titleToFilename(t string) (f string) {
	f = OutputPath
	if !strings.HasSuffix(f, "/") {
		f += "/"
	}

	for _, cp := range t {
		switch true {
		case cp >= 'a' && cp <= 'z', cp >= 'A' && cp <= 'Z', cp >= '0' && cp <= '9',
			cp == '-', cp == '+', cp == '.', cp == ',', cp == '_':
			f += string(cp)
		case cp <= 32, cp >= 127:
			// eat
		default:
			f += "_"
		}
	}
	for strings.Contains(f, "__") {
		f = strings.Replace(f, "__", "_", -1)
	}
	f = strings.Replace(f, "--", "-", -1)
	return
}

// Write body to a new file (name pattern is <TestTitle>.<N>.<FileExtension>).
// N is increased up to 999 to find a "new" file.
func dumpBody(body []byte, title, url_, ct string) {
	name := titleToFilename(title)
	ext := determineExt(url_, ct)
	var fname string
	for i := 0; i < 1000; i++ {
		fname = fmt.Sprintf("%s.%03d.%s", name, i, ext)
		_, err := os.Stat(fname)
		if e, ok := err.(*os.PathError); ok && e == os.ErrNotExist {
			break
		}
	}

	file, err := os.Create(fname)
	if err != nil {
		errorf("Cannot dump body to file '%s': %s.", fname, err.Error())
	} else {
		defer file.Close()
		file.Write(body)
	}
}
