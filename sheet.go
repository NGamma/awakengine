// Copyright 2016 Josh Deprez
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awakengine

import "github.com/DrJosh9000/vec"

type Sheet struct {
	Columns   int
	Key       string
	Frames    int
	FrameSize vec.I2
}

func (s *Sheet) ImageKey() string { return s.Key }

// Src returns the source rectangle for frame number f.
func (s *Sheet) FrameSrc(f int) (x0, y0, x1, y1 int) {
	f %= s.Frames
	if s.Columns == 0 {
		x0, y0 = vec.NewI2(f, 0).EMul(s.FrameSize).C()
		x1, y1 = x0+s.FrameSize.X, y0+s.FrameSize.Y
		return
	}
	x0, y0 = vec.Div(f, s.Columns).EMul(s.FrameSize).C()
	x1, y1 = x0+s.FrameSize.X, y0+s.FrameSize.Y
	return
}

// Dst returns the destination rectangle with the top-left corner at 0, 0.
func (s *Sheet) Dst() (x0, y0, x1, y1 int) {
	x1, y1 = s.FrameSize.C()
	return
}

// SheetFrame lets you specify a frame in addition to a sheet.
type SheetFrame struct {
	*Sheet
	Index int
}

func (s *SheetFrame) Src() (x0, y0, x1, y1 int) { return s.FrameSrc(s.Index) }
