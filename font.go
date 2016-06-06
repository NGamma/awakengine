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

type CharMetrics map[byte]CharInfo

type Font interface {
	ImageKey(invert bool) string
	Metrics() CharMetrics
	LineHeight() int
	YOffset() int
}

type CharInfo struct {
	X, Y, Width, Height, XOffset, YOffset, XAdvance int
}

type oneChar struct {
	text *Text
	ChildOf
	pos     vec.I2
	c       byte
	visible bool
}

func (s *oneChar) ImageKey() string { return s.text.Font.ImageKey(s.text.Invert) }

func (s *oneChar) Src() (x0, y0, x1, y1 int) {
	m := s.text.Metrics()
	ci := m[s.c]
	return ci.X, ci.Y, ci.X + ci.Width, ci.Y + ci.Height
}

func (s *oneChar) Dst() (x0, y0, x1, y1 int) {
	m := s.text.Metrics()
	ci := m[s.c]
	x0, y0 = s.pos.X+ci.XOffset, s.pos.Y+ci.YOffset+s.text.YOffset()
	return x0, y0, x0 + ci.Width, y0 + ci.Height
}

func (s *oneChar) Visible() bool { return s.visible && s.text.Visible() }

type Text struct {
	Pos, Size vec.I2
	Font
	ChildOf
	Text   string
	Invert bool
	chars  []oneChar
	next   int
}

func (t *Text) Dst() (x0, y0, x1, y1 int) {
	x0, y0 = t.Pos.C()
	x1, y1 = t.Pos.Add(t.Size).C()
	return
}

func (t *Text) AddToScene(s *Scene) {
	for i := range t.chars {
		s.AddObject(&t.chars[i])
	}
}

// Advance makes the next character visible.
func (t *Text) Advance() error {
	if t.next < len(t.chars) {
		t.chars[t.next].visible = true
	}
	t.next++
	return nil
}

// Layout causes the text to lay out all the characters, and update
// the size to exactly contain the text. Text will be wrapped to the
// existing Size.X as a width.
func (t *Text) Layout(visible bool) {
	width := t.Size.X
	maxW := 0
	chars := make([]oneChar, 0, len(t.Text))
	cm := t.Metrics()
	x, y := 0, 0
	wordStartC, wordStartI := 0, 0 // chars index, Text index
	wrapIt := func(end int) {
		if x < width {
			return
		}
		if x > maxW {
			maxW = x
		}
		x = 0
		y += t.LineHeight()
		// Fix previous word.
		for i, j := wordStartC, wordStartI; j < end; i, j = i+1, j+1 {
			c := t.Text[j]
			ci := cm[c]
			chars[i].pos = vec.I2{x, y}
			x += ci.XAdvance
		}
	}
	for i := range t.Text {
		if t.Text[i] == '\n' {
			x = 0
			y += t.LineHeight()
			wordStartC = len(chars)
			wordStartI = i + 1
			continue
		}
		c := t.Text[i]
		ci := cm[c]
		if t.Text[i] == ' ' {
			wrapIt(i)
			wordStartC = len(chars)
			wordStartI = i + 1
			x += ci.XAdvance
			continue
		}
		chars = append(chars, oneChar{
			text:    t,
			ChildOf: ChildOf{t},
			pos:     vec.I2{x, y},
			c:       c,
			visible: visible,
		})
		x += ci.XAdvance
	}
	wrapIt(len(t.Text))
	if x > maxW {
		maxW = x
	}
	t.chars = chars
	t.Size = vec.I2{maxW, y + t.LineHeight()}
}
