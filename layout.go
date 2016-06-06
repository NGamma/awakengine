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

type GridDelegate interface {
	Columns() int
	NumItems() int
	ItemSize() vec.I2
	Item(i int, par ChildOf) Object
}

type Grid struct {
	GridDelegate
	ChildOf
	UL    vec.I2
	items []*GridItem
}

func (g *Grid) Dst() (x0, y0, x1, y1 int) {
	x0, y0 = g.UL.C()
	return
}

// AddToScene (re)loads all the items.
func (g *Grid) AddToScene(s *Scene) {
	for _, i := range g.items {
		i.retire = true
	}
	g.items = make([]*GridItem, 0, g.NumItems())
	for i := 0; i < g.NumItems(); i++ {
		item := &GridItem{
			Grid:    g,
			ChildOf: ChildOf{g},
			Index:   i,
		}
		o := g.Item(i, ChildOf{item})
		g.items = append(g.items, item)
		s.AddObject(o)
	}
}

type GridItem struct {
	*Grid
	ChildOf
	Index  int
	retire bool
}

func (i *GridItem) Dst() (x0, y0, x1, y1 int) {
	is := i.ItemSize()
	x0, y0 = vec.Div(i.Index, i.Columns()).EMul(is).C()
	x1, y1 = x0+is.X, y0+is.Y
	return
}

func (i *GridItem) Retire() bool { return i.retire }
