package term

import "fmt"

// testcook confirms that all the flags needed for cooked mode is set.
func testcook(tr Termios, f string) error {
	if tr.Iflag&(BRKINT|IGNPAR|ISTRIP|ICRNL|IXON) != BRKINT+IGNPAR+ISTRIP+ICRNL+IXON {
		return fmt.Errorf("%q Cook failed setting Iflag , got: %d want: %d", f, tr.Iflag, BRKINT+IGNPAR+ISTRIP+ICRNL+IXON)
	}
	if (tr.Oflag & OPOST) != OPOST {
		return fmt.Errorf("%q Cook failed setting Oflag , got: %d want: %d", f, tr.Oflag, OPOST)
	}
	if (tr.Lflag & (ISIG | ICANON)) != ISIG+ICANON {
		return fmt.Errorf("%q Cook failed setting Lflag , got: %d want: %d", f, tr.Lflag, ECHO+ECHONL+ICANON+ISIG+IEXTEN)
	}
	return nil
}

// testraw Checks if we really got all the terminal raw flags set.
func testraw(tr Termios, f string) error {
	if (tr.Iflag & (IGNBRK | BRKINT | PARMRK | ISTRIP | INLCR | IGNCR | ICRNL | IXON)) != 0 {
		return fmt.Errorf("%q Raw failed setting c_iflag , got: %d want: 0", f, tr.Iflag)
	}
	if (tr.Oflag & OPOST) != 0 {
		return fmt.Errorf("%q Raw failed setting Oflag , got: %d want: 0", f, tr.Oflag)
	}
	if (tr.Lflag & (ECHO | ECHONL | ICANON | ISIG | IEXTEN)) != 0 {
		return fmt.Errorf("%q Raw failed setting Lflag, got: %d want: 0", f, tr.Lflag)
	}
	if (tr.Cflag & (PARENB)) != 0 {
		return fmt.Errorf("%q Raw failed setting Cflag , got: %d want: 0", f, tr.Cflag)
	}
	if (tr.Cflag & CSIZE) != CS8 {
		return fmt.Errorf("%q Raw failed setting Cflag CS8, got: %d ", f, tr.Cflag)
	}
	if !(tr.Cc[VMIN] == 1 && tr.Cc[VTIME] == 0) {
		return fmt.Errorf("%q Raw failed setting Cc, got: %d want: 0", f, tr.Cc)
	}
	return nil
}
