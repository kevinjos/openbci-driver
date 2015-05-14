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
	"github.com/tarm/serial"
	"io"
	"log"
	"time"
)

var Command map[string][]byte = map[string][]byte{
	"stop":  []byte{'\x73'},
	"start": []byte{'\x62'},
	"reset": []byte{'\x76'},
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
		if val == Command["reset"][0] {
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
	n0, err = d.Write(Command["stop"])
	if err != nil {
		return 0, err
	}
	n += n0
	time.Sleep(10 * time.Millisecond)
	log.Printf("Writing %v to device", Command["reset"])
	n1, err = d.w.Write(Command["reset"])
	if err != nil {
		return n, err
	}
	n += n1
	time.Sleep(10 * time.Millisecond)

	init_array = [3]byte{'\x24', '\x24', '\x24'}

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
			n2, err = d.Write(Command["start"])
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
