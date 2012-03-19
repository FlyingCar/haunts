package game

import (
  "github.com/runningwild/haunts/base"
  "github.com/runningwild/haunts/game/hui"
  "github.com/runningwild/glop/gui"
  "github.com/runningwild/glop/gin"
  "github.com/runningwild/glop/util/algorithm"
)

type hauntSetupLayout struct {
  Purposes []iconWithText
  Purpose struct {
    Dx, Dy int
  }
  Roster rosterLayout
}


// This is the UI that the haunts player uses to select his roster at the
// beginning of the game.  It will necessarily be centered on the screen
type hauntSetup struct {
  *gui.AnchorBox

  game *Game

  roster_chooser *hui.RosterChooser

  points int

  mode EntLevel
}

func makeEntityPlacer(game *Game, ents []*Entity) hui.Selector {
  return func(index int, selected map[int]bool, doit bool) (valid bool) {
    if index == -1 {
      valid = (len(selected) == 1)
      if valid {
        game.new_ent = nil
      }
    } else {
      valid = true
    }
    if doit {
      var other int
      for k,_ := range selected {
        other = k
      }
      delete(selected, other)
      selected[index] = true
      if game.new_ent != ents[index] {
        game.viewer.RemoveDrawable(game.new_ent)
        game.new_ent = MakeEntity(ents[index].Name, game)
        game.viewer.AddDrawable(game.new_ent)
      }
    }
    return
  }
}

func (hs *hauntSetup) makeServitorPlacer(ents []*Entity) hui.Selector {
  return func(index int, selected map[int]bool, doit bool) (valid bool) {
    if index == -1 {
      valid = true
      // hs.game.new_ent = nil
    } else {
      valid = ents[index].HauntEnt.Cost <= hs.points
      if _, ok := selected[index]; !valid && ok {
        delete(selected, index)
      }
    }
    if doit && valid {
      var other int
      for k,_ := range selected {
        other = k
      }
      delete(selected, other)
      selected[index] = true
      if hs.game.new_ent == nil || hs.game.new_ent.Name != ents[index].Name {
        hs.game.viewer.RemoveDrawable(hs.game.new_ent)
        hs.game.new_ent = MakeEntity(ents[index].Name, hs.game)
        hs.game.viewer.AddDrawable(hs.game.new_ent)
      }
    }
    return
  }
}

func getAllEntsWithSideAndLevel(game *Game, side Side, level EntLevel) []*Entity {
  names := base.GetAllNamesInRegistry("entities")
  ents := algorithm.Map(names, []*Entity{}, func(a interface{}) interface{} {
    return MakeEntity(a.(string), game)
  }).([]*Entity)
  ents = algorithm.Choose(ents, func(a interface{}) bool {
    return a.(*Entity).Side() == side && a.(*Entity).HauntEnt.Level == level
  }).([]*Entity)
  return ents
}

func MakeHauntSetupBar(game *Game) (*hauntSetup, error) {
  var hs hauntSetup
  hs.game = game
  hs.mode = LevelMaster

  ents := getAllEntsWithSideAndLevel(game, SideHaunt, LevelMaster)
  var roster []hui.Option
  for _, ent := range ents {
    roster = append(roster, makeEntLabel(ent))
  }

  hs.roster_chooser = hui.MakeRosterChooser(roster,
    makeEntityPlacer(game, ents),
    func(m map[int]bool) {},
    )

  hs.AnchorBox = gui.MakeAnchorBox(gui.Dims{1024, 768})
  hs.AnchorBox.AddChild(hs.roster_chooser, gui.Anchor{0, 0.5, 0, 0.5})

  return &hs, nil
}

func (hs *hauntSetup) Think(ui *gui.Gui, t int64) {
  hs.AnchorBox.Think(ui, t)
}

func (hs *hauntSetup) masterToServitor() {
  hs.mode = LevelServitor
  hs.AnchorBox.RemoveChild(hs.roster_chooser)

  ents := getAllEntsWithSideAndLevel(hs.game, SideHaunt, LevelServitor)
  var roster []hui.Option
  for _, ent := range ents {
    roster = append(roster, makeEntLabel(ent))
  }

  hs.roster_chooser = hui.MakeRosterChooser(roster,
    hs.makeServitorPlacer(ents),
    func(m map[int]bool) {},
    )

  hs.AnchorBox.AddChild(hs.roster_chooser, gui.Anchor{0, 0.5, 0, 0.5})
}

func (hs *hauntSetup) Respond(ui *gui.Gui, group gui.EventGroup) bool {
  if hs.AnchorBox.Respond(ui, group) {
    return true
  }
  if hs.game.new_ent != nil {
    x,y := gin.In().GetCursor("Mouse").Point()
    fbx, fby := hs.game.viewer.WindowToBoard(x, y)
    bx, by := DiscretizePoint32(fbx, fby)
    hs.game.new_ent.X, hs.game.new_ent.Y = float64(bx), float64(by)
    if found,event := group.FindEvent(gin.MouseLButton); found && event.Type == gin.Press {
      ix,iy := int(hs.game.new_ent.X), int(hs.game.new_ent.Y)
      r, f := hs.game.house.Floors[0].RoomAndFurnAtPos(ix, iy)
      if r == nil || f != nil { return true }
      for _,e := range hs.game.Ents {
        x,y := e.Pos()
        if x == ix && y == iy { return true }
      }
      hs.game.Ents = append(hs.game.Ents, hs.game.new_ent)
      if hs.mode == LevelMaster {
        hs.points = hs.game.new_ent.HauntEnt.Cost
        hs.masterToServitor()
        hs.game.new_ent = nil
      } else if hs.mode == LevelServitor {
        hs.points-=hs.game.new_ent.HauntEnt.Cost
        if hs.game.new_ent.HauntEnt.Cost <= hs.points {
          hs.game.new_ent = MakeEntity(hs.game.new_ent.Name, hs.game)
          hs.game.viewer.AddDrawable(hs.game.new_ent)
        } else {
          hs.game.new_ent = nil
        }
      }
    }
  }
  return false
}

func (hs *hauntSetup) Draw(r gui.Region) {
  hs.BasicZone.Render_region = r
  hs.AnchorBox.Draw(r)
}

func (hs *hauntSetup) String() string {
  return "haunt setup"
}

