package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cavaliergopher/cpio"
	"github.com/google/renameio"
	"github.com/klauspost/cpuid/v2"
)

var (
	output    = flag.String("output", "ucode.img", "Output image file")
	universal = flag.Bool("universal", false, "Include all available microcode files into the image")
)

var (
	// ucode files location at the host
	// these locations might be different at different distros, e.g. CentOS7 uses
	// /usr/share/microcode_ctl/ucode_with_caveats/intel-ucode path
	hostUcodeDirIntel = "/lib/firmware/intel-ucode"
	hostUcodeDirAmd   = "/lib/firmware/amd-ucode"

	imageUcodePath      = "kernel/x86/microcode"
	imageUcodePathIntel = imageUcodePath + "/GenuineIntel.bin"
	imageUcodePathAmd   = imageUcodePath + "/AuthenticAMD.bin"
)

type image struct {
	*cpio.Writer
	modificationTime time.Time
}

func (i *image) mkPath(dir string) error {
	curr := ""
	for _, elem := range strings.Split(dir, "/") {
		if curr != "" {
			curr += "/"
		}
		curr += elem
		h := &cpio.Header{
			Name:    curr,
			Mode:    0o755 | cpio.TypeDir,
			ModTime: i.modificationTime,
			Size:    0,
		}
		if err := i.WriteHeader(h); err != nil {
			return err
		}
	}

	return nil
}

func (i *image) addAllUcodeFiles() error {
	if _, err := os.Stat(hostUcodeDirIntel); !os.IsNotExist(err) {
		if err := i.appendFiles(imageUcodePathIntel, hostUcodeDirIntel+"/*"); err != nil {
			return err
		}
	}

	if _, err := os.Stat(hostUcodeDirAmd); !os.IsNotExist(err) {
		if err := i.appendFiles(imageUcodePathAmd, hostUcodeDirAmd+"/*"); err != nil {
			return err
		}
	}

	return nil
}

func (i *image) addHostSpecificUcodeFiles() error {
	vendor := cpuid.CPU.VendorID
	switch vendor {
	case cpuid.Intel:
		infile := fmt.Sprintf("%s/%02x-%02x-%02x", hostUcodeDirIntel, cpuid.CPU.Family, cpuid.CPU.Model, cpuid.CPU.Stepping)
		if err := i.appendFiles(imageUcodePathIntel, infile); err != nil {
			return err
		}
	case cpuid.AMD:
		family := cpuid.CPU.Family
		var infile string
		if family >= 21 {
			infile = fmt.Sprintf("%s/microcode_amd_fam%xh.bin", hostUcodeDirAmd, family)
		} else {
			infile = fmt.Sprintf("%s/microcode_amd.bin", hostUcodeDirAmd)
		}
		if err := i.appendFiles(imageUcodePathAmd, infile); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unable to find microcode for processor vendor %s", vendor.String())
	}
	return nil
}

func (i *image) appendFiles(outFile string, inputFiles string) error {
	ucode, err := readFilesContent(inputFiles)
	if err != nil {
		return err
	}

	h := &cpio.Header{
		Name:    outFile,
		Mode:    0o644 | cpio.TypeReg,
		ModTime: i.modificationTime,
		Size:    int64(len(ucode)),
	}
	if err := i.WriteHeader(h); err != nil {
		return err
	}
	if _, err := i.Write(ucode); err != nil {
		return err
	}

	return nil
}

func readFilesContent(glob string) ([]byte, error) {
	matches, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no files found for %s", glob)
	}

	var content bytes.Buffer
	for _, m := range matches {
		c, err := os.ReadFile(m)
		if err != nil {
			return nil, err
		}
		content.Write(c)
	}
	return content.Bytes(), nil
}

func fileModificationTime() time.Time {
	// https://reproducible-builds.org/docs/source-date-epoch/
	// $SOURCE_DATE_EPOCH contains time in seconds since 1970, e.g. "1520598896"
	epoch := os.Getenv("SOURCE_DATE_EPOCH")
	if epoch != "" {
		if timestamp, err := strconv.ParseInt(epoch, 10, 64); err == nil {
			return time.Unix(timestamp, 0)
		}
		fmt.Printf("unable to parse $SOURCE_DATE_EPOCH value '%s'\n", epoch)
	}
	return time.Now()
}

func generateImage() error {
	file, err := renameio.TempFile("", *output)
	if err != nil {
		return err
	}
	defer file.Cleanup()

	if err := file.Chmod(0o644); err != nil {
		return err
	}

	w := cpio.NewWriter(file)
	defer w.Close()

	i := image{w, fileModificationTime()}

	if err := i.mkPath(imageUcodePath); err != nil {
		return err
	}

	if *universal {
		if err := i.addAllUcodeFiles(); err != nil {
			return err
		}
	} else {
		if err := i.addHostSpecificUcodeFiles(); err != nil {
			return err
		}
	}

	if err := i.Close(); err != nil {
		return err
	}
	return file.CloseAtomicallyReplace()
}

func main() {
	flag.Parse()

	if err := generateImage(); err != nil {
		panic(err)
	}
}
