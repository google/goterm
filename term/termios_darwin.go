package term

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// IOCTL terminal stuff.
const (
	TCGETS     = 0x5401     // TCGETS get terminal attributes
	TCSETS     = 0x5402     // TCSETS set terminal attributes
	TIOCGWINSZ = 0x5413     // TIOCGWINSZ used to get the terminal window size
	TIOCSWINSZ = 0x5414     // TIOCSWINSZ used to set the terminal window size
	TIOCGPTN   = 0x80045430 // TIOCGPTN IOCTL used to get the PTY number
	TIOCSPTLCK = 0x40045431 // TIOCSPTLCK IOCT used to lock/unlock PTY
	CBAUD      = 0010017    // CBAUD Serial speed settings
	CBAUDEX    = 0010000    // CBAUDX Serial speed settings
)

// Set Sets terminal t attributes on file.
func (t *Termios) Set(file *os.File) error {
	fd := file.Fd()
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(TCSETS), uintptr(unsafe.Pointer(t)))
	if errno != 0 {
		return errno
	}
	return nil
}

// Attr Gets (terminal related) attributes from file.
func Attr(file *os.File) (Termios, error) {
	var t Termios
	fd := file.Fd()
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(TCGETS), uintptr(unsafe.Pointer(&t)))
	if errno != 0 {
		return t, errno
	}
	t.Ispeed &= CBAUD | CBAUDEX
	t.Ospeed &= CBAUD | CBAUDEX
	return t, nil
}

// Isatty returns true if file is a tty.
func Isatty(file *os.File) bool {
	_, err := Attr(file)
	return err == nil
}

// GetPass reads password from a TTY with no echo.
func GetPass(prompt string, f *os.File, pbuf []byte) ([]byte, error) {
	t, err := Attr(f)
	if err != nil {
		return nil, err
	}
	defer t.Set(f)
	noecho := t
	noecho.Lflag = noecho.Lflag &^ ECHO
	if err := noecho.Set(f); err != nil {
		return nil, err
	}
	b := make([]byte, 1, 1)
	i := 0
	if _, err := f.Write([]byte(prompt)); err != nil {
		return nil, err
	}
	for ; i < len(pbuf); i++ {
		if _, err := f.Read(b); err != nil {
			b[0] = 0
			clearbuf(pbuf[:i+1])
		}
		if b[0] == '\n' || b[0] == '\r' {
			return pbuf[:i], nil
		}
		pbuf[i] = b[0]
		b[0] = 0
	}
	clearbuf(pbuf[:i+1])
	return nil, errors.New("ran out of bufferspace")
}

// clearbuf clears out the buffer incase we couldn't read the full password.
func clearbuf(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Winsz Fetches the current terminal windowsize.
// example handling changing window sizes with PTYs:
//
// import "os"
// import "os/signal"
//
// var sig = make(chan os.Signal,2) 		// Channel to listen for UNIX SIGNALS on
// signal.Notify(sig, syscall.SIGWINCH) // That'd be the window changing
//
// for {
//	<-sig
// 	term.Winsz(os.Stdin)			// We got signaled our terminal changed size so we read in the new value
//  term.Setwinsz(pty.Slave) // Copy it to our virtual Terminal
// }
func (t *Termios) Winsz(file *os.File) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(TIOCGWINSZ), uintptr(unsafe.Pointer(&t.Wz)))
	if errno != 0 {
		return errno
	}
	return nil
}

// Setwinsz Sets the terminal window size.
func (t *Termios) Setwinsz(file *os.File) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(TIOCSWINSZ), uintptr(unsafe.Pointer(&t.Wz)))
	if errno != 0 {
		return errno
	}
	return nil
}

// OpenPTY Creates a new Master/Slave PTY pair.
func OpenPTY() (*PTY, error) {
	// Opening ptmx gives you the FD of a brand new PTY
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	// unlock pty slave
	var unlock int // 0 => Unlock
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(master.Fd()), uintptr(TIOCSPTLCK), uintptr(unsafe.Pointer(&unlock))); errno != 0 {
		master.Close()
		return nil, errno
	}

	// get path of pts slave
	pty := &PTY{Master: master}
	slaveStr, err := pty.PTSName()
	if err != nil {
		master.Close()
		return nil, err
	}

	// open pty slave
	pty.Slave, err = os.OpenFile(slaveStr, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, err
	}

	return pty, nil
}

// Close closes the PTYs that OpenPTY created.
func (p *PTY) Close() error {
	slaveErr := errors.New("Slave FD nil")
	if p.Slave != nil {
		slaveErr = p.Slave.Close()
	}
	masterErr := errors.New("Master FD nil")
	if p.Master != nil {
		masterErr = p.Master.Close()
	}
	if slaveErr != nil || masterErr != nil {
		var errs []string
		if slaveErr != nil {
			errs = append(errs, "Slave: "+slaveErr.Error())
		}
		if masterErr != nil {
			errs = append(errs, "Master: "+masterErr.Error())
		}
		return errors.New(strings.Join(errs, " "))
	}
	return nil
}

// PTSName return the name of the pty.
func (p *PTY) PTSName() (string, error) {
	n, err := p.PTSNumber()
	if err != nil {
		return "", err
	}
	return "/dev/pts/" + strconv.Itoa(int(n)), nil
}

// PTSNumber return the pty number.
func (p *PTY) PTSNumber() (uint, error) {
	var ptyno uint
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(p.Master.Fd()), uintptr(TIOCGPTN), uintptr(unsafe.Pointer(&ptyno)))
	if errno != 0 {
		return 0, errno
	}
	return ptyno, nil
}
