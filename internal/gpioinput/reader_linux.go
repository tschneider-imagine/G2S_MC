//go:build linux

package gpioinput

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

const (
	gpioMaxNameSize          = 32
	gpioV2LinesMax           = 64
	gpioV2LineNumAttrsMax    = 10
	gpioV2LineFlagInput      = 1 << 2
	gpioV2LineFlagBiasPullUp = 1 << 8
)

type gpioChipInfo struct {
	Name  [gpioMaxNameSize]byte
	Label [gpioMaxNameSize]byte
	Lines uint32
}

type gpioV2LineAttribute struct {
	ID      uint32
	Padding uint32
	Value   uint64
}

type gpioV2LineConfigAttribute struct {
	Attr gpioV2LineAttribute
	Mask uint64
}

type gpioV2LineConfig struct {
	Flags    uint64
	NumAttrs uint32
	Padding  [5]uint32
	Attrs    [gpioV2LineNumAttrsMax]gpioV2LineConfigAttribute
}

type gpioV2LineRequest struct {
	Offsets         [gpioV2LinesMax]uint32
	Consumer        [gpioMaxNameSize]byte
	Config          gpioV2LineConfig
	NumLines        uint32
	EventBufferSize uint32
	Padding         [5]uint32
	FD              int32
}

type gpioV2LineValues struct {
	Bits uint64
	Mask uint64
}

const (
	ioctlNRBits   = 8
	ioctlTypeBits = 8
	ioctlSizeBits = 14
	ioctlDirBits  = 2

	ioctlNRShift   = 0
	ioctlTypeShift = ioctlNRShift + ioctlNRBits
	ioctlSizeShift = ioctlTypeShift + ioctlTypeBits
	ioctlDirShift  = ioctlSizeShift + ioctlSizeBits

	ioctlWrite = 1
	ioctlRead  = 2

	gpioIoctlMagic = 0xB4
)

var (
	gpioGetChipInfoIOCTL = ior(gpioIoctlMagic, 0x01, unsafe.Sizeof(gpioChipInfo{}))
	gpioV2GetLineIOCTL   = iowr(gpioIoctlMagic, 0x07, unsafe.Sizeof(gpioV2LineRequest{}))
	gpioV2GetValuesIOCTL = iowr(gpioIoctlMagic, 0x0E, unsafe.Sizeof(gpioV2LineValues{}))
)

func ioc(dir, iocType, nr, size uintptr) uintptr {
	return (dir << ioctlDirShift) | (iocType << ioctlTypeShift) | (nr << ioctlNRShift) | (size << ioctlSizeShift)
}

func ior(iocType, nr uintptr, size uintptr) uintptr {
	return ioc(ioctlRead, iocType, nr, size)
}

func iowr(iocType, nr uintptr, size uintptr) uintptr {
	return ioc(ioctlRead|ioctlWrite, iocType, nr, size)
}

func ioctlPtr(fd int, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

func (r *Reader) Read(ctx context.Context, gpioChannel string) (inputs.DigitalState, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if r == nil {
		return "", fmt.Errorf("gpio reader is nil")
	}

	offset, canonical, err := ParseBCMChannel(gpioChannel)
	if err != nil {
		return "", err
	}

	chipPath := strings.TrimSpace(r.ChipPath)
	if chipPath == "" {
		chipPath = DefaultChipPath
	}
	consumer := strings.TrimSpace(r.Consumer)
	if consumer == "" {
		consumer = "g2s_gpio_probe"
	}

	state, err := readLineWithPullUp(chipPath, consumer, offset)
	if err != nil {
		return "", fmt.Errorf("read %s on %s: %w", canonical, chipPath, err)
	}
	return state, nil
}

func readLineWithPullUp(chipPath string, consumer string, offset int) (inputs.DigitalState, error) {
	chip, err := os.OpenFile(chipPath, os.O_RDONLY, 0)
	if err != nil {
		return "", fmt.Errorf("open gpio chip: %w", err)
	}
	defer chip.Close()

	var request gpioV2LineRequest
	request.Offsets[0] = uint32(offset)
	request.NumLines = 1
	request.Config.Flags = gpioV2LineFlagInput | gpioV2LineFlagBiasPullUp
	copy(request.Consumer[:], []byte(consumer))

	if err := ioctlPtr(int(chip.Fd()), gpioV2GetLineIOCTL, unsafe.Pointer(&request)); err != nil {
		if errors.Is(err, syscall.EINVAL) {
			return "", fmt.Errorf("pull-up bias request unsupported by kernel/driver on %s: %w", chipPath, err)
		}
		return "", fmt.Errorf("request GPIO line with pull-up failed: %w", err)
	}

	lineFD := int(request.FD)
	if lineFD <= 0 {
		return "", fmt.Errorf("request GPIO line returned invalid fd %d", lineFD)
	}
	defer syscall.Close(lineFD)

	values := gpioV2LineValues{Mask: 1}
	if err := ioctlPtr(lineFD, gpioV2GetValuesIOCTL, unsafe.Pointer(&values)); err != nil {
		return "", fmt.Errorf("read GPIO line value failed: %w", err)
	}

	return mapRawValueToDigitalState(values.Bits & 0x1)
}

func readChipInfo(chipPath string) (*gpioChipInfo, error) {
	chip, err := os.OpenFile(chipPath, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open gpio chip: %w", err)
	}
	defer chip.Close()

	var info gpioChipInfo
	if err := ioctlPtr(int(chip.Fd()), gpioGetChipInfoIOCTL, unsafe.Pointer(&info)); err != nil {
		return nil, fmt.Errorf("read chip info ioctl failed: %w", err)
	}
	return &info, nil
}

func probePullUpSupport(chipPath string, channel string) error {
	offset, _, err := ParseBCMChannel(channel)
	if err != nil {
		return err
	}
	_, err = readLineWithPullUp(chipPath, "g2s_gpio_probe_check", offset)
	return err
}

func parseCString(value []byte) string {
	n := 0
	for n < len(value) && value[n] != 0 {
		n++
	}
	return string(value[:n])
}
