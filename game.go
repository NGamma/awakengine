package awakengine

import (
	"image/color"
	"log"
	"os"
	"sort"

	"github.com/DrJosh9000/vec"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

var (
	// Debug controls display of debug graphics.
	Debug bool

	game      Game
	gameFrame int

	mouseDown bool

	pixelSize = 3
	camSize   = vec.I2{267, 150}
	camPos    = vec.I2{0, 0}
	title     = "AwakEngine"

	terrain          *Terrain
	obstacles, paths *vec.Graph

	triggers        map[string]*Trigger
	dialogueStack   []DialogueLine
	currentDialogue *DialogueDisplay

	player Unit
	units  []Unit
)

// Unit can be told to update and provide information for drawing.
// Examples of units include the player character, NPCs, etc.
type Unit interface {
	GoIdle()                       // stop whatever you're doing.
	Footprint() (ul, dr vec.I2)    // from the sprite position, the ground area of the unit
	Path() []vec.I2                // the current position is implied
	Sprite                         // for drawing
	Update(frame int, event Event) // time moves on, so compute new state
}

// UnitsByYPos orders Sprites by Y position (least to greatest).
type UnitsByYPos []Unit

// Len implements sort.Interface.
func (b UnitsByYPos) Len() int { return len(b) }

// Less implements sort.Interface.
func (b UnitsByYPos) Less(i, j int) bool { return b[i].Pos().Y < b[j].Pos().Y }

// Swap implements sort.Interface.
func (b UnitsByYPos) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Level abstracts things needed for a base terrain/level.
type Level interface {
	// Doodads provides objects above the base, that can be flattened onto the terrain
	// most of the time.
	Doodads() []*Doodad

	// Source is the paletted PNG to use as the base terrain layer - pixel at (x,y) becomes
	// the tile at (x,y).
	Source() string

	// TileInfos maps indexes to information about the terrain.
	TileInfos() []TileInfo

	// Tiles is an image containing square tiles.
	Tiles() (key string, tileSize int)
}

// Game abstracts the non-engine parts of the game: the story, art, level design, etc.
type Game interface {
	// Terrain provides the base level.
	Level() Level

	// Triggers provide some dynamic behaviour.
	Triggers() map[string]*Trigger

	// Units provides all units in the level.
	Units() []Unit

	// Viewport is the size of the window and the pixels in the window.
	Viewport() (camSize vec.I2, pixelSize int, title string)
}

// Load prepares assets for use by the game.
func Load(g Game, debug bool) error {
	game = g
	Debug = debug
	camSize, pixelSize, title = game.Viewport()

	if err := loadAllImages(); err != nil {
		return err
	}

	triggers = game.Triggers()
	units = game.Units()

	b, err := NewBubble(vec.I2{10, camSize.Y - 80}, vec.I2{camSize.X - 20, 70})
	if err != nil {
		return err
	}
	dialogueBubble = b

	t, err := loadTerrain(game.Level())
	if err != nil {
		return err
	}
	terrain = t

	// TODO: distinguish unit 0 as the player.
	// TODO: compute unfattened static obstacles and fully dynamic paths.
	// Invert the footprint to fatten the obstacles with.
	player = units[0]
	ul, dr := player.Footprint()
	ul = ul.Mul(-1)
	dr = dr.Mul(-1)
	obstacles, paths = t.ObstaclesAndPaths(ul, dr)

	return nil
}

// Run runs the game (ebiten.Run) i n addition to setting up any necessary GIF recording.
func Run(rf string, frameCount int) error {
	up := update
	if rf != "" {
		f, err := os.Create(rf)
		if err != nil {
			return err
		}
		defer f.Close()
		up = ebitenutil.RecordScreenAsGIF(up, f, frameCount)
	}
	return ebiten.Run(up, camSize.X, camSize.Y, pixelSize, title)
}

// drawDebug draws debugging graphics onto the screen if Debug is true.
func drawDebug(screen *ebiten.Image) error {
	if !Debug {
		return nil
	}
	obsView := GraphView{
		edges:        obstacles.Edges(),
		edgeColour:   color.RGBA{0xff, 0, 0, 0xff},
		normalColour: color.RGBA{0, 0xff, 0, 0xff},
	}
	if err := screen.DrawLines(obsView); err != nil {
		return err
	}
	pathsView := GraphView{
		edges:        paths.Edges(),
		edgeColour:   color.RGBA{0, 0, 0xff, 0xff},
		normalColour: color.Transparent,
	}
	if err := screen.DrawLines(pathsView); err != nil {
		return err
	}
	if len(player.Path()) > 0 {
		u := player.Pos().Sub(camPos)
		for _, v := range player.Path() {
			v = v.Sub(camPos)
			if err := screen.DrawLine(u.X, u.Y, v.X, v.Y, color.RGBA{0, 0xff, 0xff, 0xff}); err != nil {
				return err
			}
			u = v
		}
	}
	return nil
}

// update is the main update function.
func update(screen *ebiten.Image) error {
	// Read inputs
	md := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	e := Event{Pos: vec.NewI2(ebiten.CursorPosition()).Add(camPos)}
	switch {
	case md && !mouseDown:
		mouseDown = true
		e.Type = EventMouseDown
	case !md && mouseDown:
		mouseDown = false
		e.Type = EventMouseUp
	}

	// Do we proceed with the game, or with the dialogue display?
	if currentDialogue == nil {
		// Got any triggers?
		for k, trig := range triggers {
			if !trig.Fired && trig.Active() {
				// All dependencies fired?
				for _, dep := range trig.Depends {
					if !triggers[dep].Fired {
						continue
					}
				}
				if Debug {
					log.Printf("firing %s with %d dialogues", k, len(trig.Dialogues))
				}
				if trig.Fire != nil {
					trig.Fire()
				}
				dialogueStack = trig.Dialogues
				currentDialogue = nil
				player.GoIdle()
				if len(dialogueStack) > 0 {
					d, err := DialogueFromLine(dialogueStack[0])
					if err != nil {
						return err
					}
					currentDialogue = d
				}
				trig.Fired = true
				break
			}
		}
		if currentDialogue == nil {
			gameFrame++
			player.Update(gameFrame, e)
		}
	} else if currentDialogue.Update(e) {
		// Play
		dialogueStack = dialogueStack[1:]
		currentDialogue = nil
		if len(dialogueStack) > 0 {
			d, err := DialogueFromLine(dialogueStack[0])
			if err != nil {
				return err
			}
			currentDialogue = d
		}
	}

	// Update camera to focus on player.
	camPos = player.Pos().Sub(camSize.Div(2)).ClampLo(vec.I2{}).ClampHi(terrain.Size().Sub(camSize))

	// Draw all the things.
	if err := terrain.Draw(screen); err != nil {
		return err
	}

	// Tiny sort.
	sort.Sort(UnitsByYPos(units))
	for _, s := range units {
		if err := (SpriteParts{s, true}.Draw(screen)); err != nil {
			return err
		}
	}

	// Any doodads overlapping the player?
	pp := player.Pos()
	pu := pp.Sub(player.Anim().Offset)
	pd := pu.Add(player.Anim().FrameSize)
	for _, dd := range terrain.doodads {
		if pp.Y >= dd.P.Y {
			continue
		}
		tu := dd.P.Sub(dd.Anim().Offset)
		td := tu.Add(dd.Anim().FrameSize)
		if tu.Y > pd.Y || td.Y < pu.Y {
			// td.Y < pu.Y is essentially given, but consistency.
			continue
		}
		if tu.X > pd.X || td.X < pu.X {
			continue
		}
		if err := (SpriteParts{dd, true}.Draw(screen)); err != nil {
			return err
		}
	}

	// The W is special. All hail the W!
	wu := theW.pos.Sub(theW.Anim().Offset)
	wd := wu.Add(theW.Anim().FrameSize)
	cd := camPos.Add(camSize)
	if (wu.X < cd.X || wd.X >= camPos.X) && (wu.Y < cd.Y || wd.Y >= camPos.X) {
		if err := (SpriteParts{theW, true}.Draw(screen)); err != nil {
			return err
		}
	}

	if currentDialogue != nil {
		if err := currentDialogue.Draw(screen); err != nil {
			return err
		}
	}
	return drawDebug(screen)
	//return nil
}
