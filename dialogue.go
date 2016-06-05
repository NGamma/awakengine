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

const dialogueZ = 100000

type ButtonSpec struct {
	Label  string
	Action func()
}

// DialogueLine is information for displaying a singe line of dialogue in a display.
type DialogueLine struct {
	Avatar   *SheetFrame
	Text     string
	Buttons  []ButtonSpec
	AutoNext bool
	Slowness int
}

// Dialogue is all the things needed for displaying blocking dialogue text.
type DialogueDisplay struct {
	bubble   *Bubble
	buttons  []*Button
	frame    int // frame number for this dialogue.
	text     *Text
	complete bool
	retire   bool
	visible  bool
	avatar   *Billboard

	line *DialogueLine
}

// DialogueFromLine creates a new DialogueDisplay.
func DialogueFromLine(line *DialogueLine, scene *Scene) *DialogueDisplay {
	basePos := vec.I2{10, scene.CameraSize.Y - 80}
	baseSize := vec.I2{scene.CameraSize.X - 20, 70}
	textPos := basePos.Add(vec.I2{15, 15})
	var avatar *Billboard
	if line.Avatar != nil {
		// Provide space for the avatar.
		textPos.X += line.Avatar.Sheet.FrameSize.X + 5
		avatar = &Billboard{
			SheetFrame: line.Avatar,
			P:          vec.I2{15, scene.CameraSize.Y - 80 + 2},
		}
	}
	bk, _ := game.BubbleKey()
	d := &DialogueDisplay{
		line:    line,
		avatar:  avatar,
		frame:   0,
		visible: true,
		text: &Text{
			Text: line.Text,
			Pos:  textPos,
			Size: vec.I2{scene.CameraSize.X - textPos.X - 35, 0},
			Font: game.Font(),
		},
		bubble: &Bubble{
			UL:  basePos,
			DR:  basePos.Add(baseSize),
			Key: bk,
		},
	}
	d.bubble.ChildOf = ChildOf{d}
	d.text.ChildOf = ChildOf{d.bubble}
	d.text.Layout(false) // Rolls out the text for each Advance.
	p := vec.I2{textPos.X + 15, basePos.Y + baseSize.Y - 30}
	for _, s := range line.Buttons {
		d.buttons = append(d.buttons, NewButton(s.Label, s.Action, p, p.Add(vec.I2{50, 18}), ChildOf{d.bubble}))
		p.X += 65
	}
	return d
}

func (d *DialogueDisplay) Parent() Semiobject { return nil }
func (d *DialogueDisplay) Fixed() bool        { return true }
func (d *DialogueDisplay) Retire() bool       { return d.retire }
func (d *DialogueDisplay) Visible() bool      { return d.visible }
func (d *DialogueDisplay) Z() int             { return dialogueZ }

func (d *DialogueDisplay) AddToScene(s *Scene) {
	d.bubble.AddToScene(s)
	d.text.AddToScene(s)
	for _, b := range d.buttons {
		b.AddToScene(s)
	}
	if d.avatar == nil {
		return
	}
	s.AddObject(&struct {
		*Billboard
		ChildOf
	}{d.avatar, ChildOf{d.bubble}})
}

func (d *DialogueDisplay) finish() {
	d.complete = true
	for d.text.next < len(d.text.chars) {
		d.text.Advance()
	}
}

// Update updates things in the dialogue, based on user input or passage of time.
// Returns true if the event is handled.
func (d *DialogueDisplay) Handle(event Event) bool {
	for _, b := range d.buttons {
		if b.Handle(event) {
			d.retire = true
			return true
		}
	}
	if d.complete && d.line.AutoNext {
		d.retire = true
		return true
	}
	if event.Type == EventMouseUp {
		if d.complete && len(d.buttons) == 0 {
			d.retire = true
			return true
		}
		if !d.line.AutoNext {
			d.finish()
		}
	}
	if !d.complete {
		if d.line.Slowness < 0 {
			d.finish()
		}
		if d.line.Slowness == 0 || d.frame%d.line.Slowness == 0 {
			d.text.Advance()
			if d.text.next >= len(d.text.chars) {
				d.complete = true
			}
		}
	}
	d.frame++
	return false
}
