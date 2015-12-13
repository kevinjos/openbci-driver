/*  OpenBCI golang server allows users to control, visualize and store data
    collected from the OpenBCI microcontroller.
    Copyright (C) 2015  Kevin Schiesser

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as
    published by the Free Software Foundation, either version 3 of the
    License, or (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
package openbci

import (
	"crypto/rand"
	"io"
	"log"
	"time"

	"github.com/tarm/serial"
)

var Command map[string]byte = map[string]byte{
	"stop":   '\x73',
	"start":  '\x62',
	"reset":  '\x76',
	"footer": '\xc0',
	"header": '\xa0',
	"init":   '\x24',
}

func NewDevice(location string, baud int, readTimeout time.Duration) (io.ReadWriteCloser, error) {
	conf := &serial.Config{
		Name:        location,
		Baud:        baud,
		ReadTimeout: readTimeout,
	}
	conn, err := serial.OpenPort(conf)
	if err != nil {
		return nil, err
	}
	return &Device{
		r: conn,
		w: conn,
		c: conn,
	}, nil
}

type Device struct {
	r io.Reader
	w io.Writer
	c io.Closer
}

func (d *Device) Read(buf []byte) (int, error) {
	n, err := d.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func isReset(buf []byte) bool {
	for _, val := range buf {
		if val == Command["reset"] {
			return true
		}
	}
	return false
}

func (d *Device) Write(buf []byte) (int, error) {
	if isReset(buf) {
		n, err := d.reset(buf)
		if err != nil {
			return 0, err
		}
		return n, nil
	}
	log.Printf("Writing %v to device", buf)
	n, err := d.w.Write(buf)
	time.Sleep(50 * time.Millisecond)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (d *Device) reset(buf []byte) (n int, err error) {
	var (
		n0, n1, n2, idx int
		init_array      [3]byte
		scrolling       [3]byte
	)
	buf = make([]byte, 1)
	n0, err = d.Write([]byte{Command["stop"]})
	if err != nil {
		return 0, err
	}
	n += n0
	time.Sleep(10 * time.Millisecond)
	log.Printf("Writing %v to device", Command["reset"])
	n1, err = d.Write([]byte{Command["reset"]})
	if err != nil {
		return n, err
	}
	n += n1
	time.Sleep(10 * time.Millisecond)

	init_array = [3]byte{Command["init"], Command["init"], Command["init"]}

	for {
		_, err := d.Read(buf)
		if err == io.EOF {
			continue
		} else if err != nil {
			return n, err
		}
		scrolling[idx%3] = buf[0]
		idx++
		if scrolling == init_array {
			n2, err = d.Write([]byte{Command["start"]})
			if err != nil {
				return n, err
			}
			n += n2
			return n, nil
		}
	}
}

func (d *Device) Close() error {
	err := d.c.Close()
	if err != nil {
		return err
	}
	return nil
}

func NewMockDevice() *MockDevice { return &MockDevice{on: false} }

type MockDevice struct {
	on          bool
	seqcounter  uint8
	datacounter uint8
	readstate   uint8
}

func (md *MockDevice) Read(p []byte) (n int, err error) {
	var b int
	if md.on {
		for idx := range p {
			switch md.readstate {
			case 0:
				p[idx] = Command["footer"]
				md.readstate++
				b++
			case 1:
				p[idx] = Command["header"]
				md.readstate++
				b++
			case 2:
				p[idx] = md.seqcounter
				md.readstate++
				b++
			case 3:
				buf := make([]byte, 1)
				rand.Read(buf)
				p[idx] = buf[0]
				b++
				md.datacounter++
				if md.datacounter == 30 {
					md.readstate++
					md.datacounter = 0
				}
			case 4:
				p[idx] = Command["footer"]
				md.readstate = 1
				md.seqcounter++
				b++
				time.Sleep(time.Millisecond * 25)
			}

		}
	}
	return b, nil
}

func (md *MockDevice) Write(p []byte) (n int, err error) {
	switch len(p) {
	case 0:
		return 0, nil
	default:
		switch p[0] {
		case Command["start"]:
			md.on = true
		case Command["stop"]:
			md.on = false
		}
	}
	return len(p), nil
}

func (md *MockDevice) Close() (err error) {
	return nil
}
