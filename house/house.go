package house

import (
  "glop/gui"
  "glop/gin"
  "haunts/texture"
  "haunts/base"
)

type Room struct {
  Defname string
  *roomDef
  RoomInst
}

type WallFacing int
const (
  NearLeft WallFacing = iota
  NearRight
  FarLeft
  FarRight
)

func MakeDoor(name string) *Door {
  d := Door{ Defname: name }
  base.LoadObject("doors", &d)
  return &d
}

func GetAllDoorNames() []string {
  return base.GetAllNamesInRegistry("doors")
}

func LoadAllDoorsInDir(dir string) {
  base.RemoveRegistry("doors")
  base.RegisterRegistry("doors", make(map[string]*doorDef))
  base.RegisterAllObjectsInDir("doors", dir, ".json", "json")
}

func (d *Door) Load() {
  base.LoadObject("doors", d)
}

type doorDef struct {
  // Name of this texture as it appears in the editor, should be unique among
  // all Doors
  Name string

  // Number of cells wide the door is
  Width int

  Opened_texture texture.Object  `registry:"autoload"`  
  Closed_texture texture.Object  `registry:"autoload"`  
}

type Door struct {
  Defname string
  *doorDef
  DoorInst
}

func (d *Door) TextureData() *texture.Data {
  if d.Opened {
    return d.Opened_texture.Data()
  }
  return d.Closed_texture.Data()
}

type DoorInst struct {
  // Which wall the door is on
  Facing WallFacing

  // How far along this wall the door is located
  Pos int

  // Whether or not the door is opened - determines what texture to use
  Opened bool
}

type RoomInst struct {
  // The placement of doors in this room
  Doors []*Door  `registry:"loadfrom-doors"`

  // The offset of this room on this floor
  X,Y int
}

func (ri *RoomInst) Pos() (x,y int) {
  return ri.X, ri.Y
}

type Floor struct {
  Rooms []*Room
}

type houseDef struct {
  Floors []*Floor

  // The floor that the explorers start on
  Starting_floor int
}

func MakeHouseDef() *houseDef {
  var h houseDef
  return &h
}

type HouseEditor struct {
  *gui.HorizontalTable
  tab *gui.TabFrame
  widgets []tabWidget

  house  *houseDef
  viewer *HouseViewer
}

func (he *HouseEditor) GetViewer() Viewer {
  return he.viewer
}

func (w *HouseEditor) SelectTab(n int) {
  if n < 0 || n >= len(w.widgets) { return }
  if n != w.tab.SelectedTab() {
    w.widgets[w.tab.SelectedTab()].Collapse()
    w.tab.SelectTab(n)
    // w.viewer.SetEditMode(editNothing)
    w.widgets[n].Expand()
  }
}

type houseDataTab struct {
  *gui.VerticalTable

  num_floors *gui.ComboBox
  theme      *gui.ComboBox

  house  *houseDef
  viewer *HouseViewer

  // Distance from the mouse to the center of the object, in board coordinates
  drag_anchor struct{ x,y float32 }

  // Which floor we are viewing and editing
  current_floor int
}
func makeHouseDataTab(house *houseDef, viewer *HouseViewer) *houseDataTab {
  var hdt houseDataTab
  hdt.VerticalTable = gui.MakeVerticalTable()
  hdt.house = house
  hdt.viewer = viewer

  num_floors_options := []string{ "1 Floor", "2 Floors", "3 Floors", "4 Floors" }
  hdt.num_floors = gui.MakeComboTextBox(num_floors_options, 300)
  hdt.theme = gui.MakeComboTextBox(tags.Themes, 300)

  hdt.VerticalTable.AddChild(hdt.num_floors)
  hdt.VerticalTable.AddChild(hdt.theme)

  names := GetAllRoomNames()
  for _,name := range names {
    hdt.VerticalTable.AddChild(gui.MakeButton("standard", name, 300, 1, 1, 1, 1, func(int64) {
      hdt.viewer.Temp.Room = MakeRoom(name)
    }))
  }

  return &hdt
}
func (hdt *houseDataTab) Think(ui *gui.Gui, t int64) {
  if hdt.viewer.Temp.Room != nil {
    mx,my := gin.In().GetCursor("Mouse").Point()
    bx,by := hdt.viewer.WindowToBoard(mx, my)
    hdt.viewer.Temp.Room.X = int(bx - hdt.drag_anchor.x)
    hdt.viewer.Temp.Room.Y = int(by - hdt.drag_anchor.y)
  }
  hdt.VerticalTable.Think(ui, t)
  num_floors := hdt.num_floors.GetComboedIndex() + 1
  if len(hdt.house.Floors) != num_floors {
    for len(hdt.house.Floors) < num_floors {
      hdt.house.Floors = append(hdt.house.Floors, &Floor{})
    }
    if len(hdt.house.Floors) > num_floors {
      hdt.house.Floors = hdt.house.Floors[0 : num_floors]
    }
  }
}
func (hdt *houseDataTab) Respond(ui *gui.Gui, group gui.EventGroup) bool {
  if hdt.VerticalTable.Respond(ui, group) {
    return true
  }

  if found,event := group.FindEvent(gin.Escape); found && event.Type == gin.Press {
    hdt.viewer.Temp.Room = nil
    return true
  }

  floor := hdt.house.Floors[hdt.current_floor]
  if found,event := group.FindEvent(gin.MouseLButton); found && event.Type == gin.Press {
    if hdt.viewer.Temp.Room != nil {
      if floor.canAddRoom(hdt.viewer.Temp.Room) {
        floor.Rooms = append(floor.Rooms, hdt.viewer.Temp.Room)
        hdt.viewer.Temp.Room = nil
      }
    } else {
      bx,by := hdt.viewer.WindowToBoard(event.Key.Cursor().Point())
      for i := range floor.Rooms {
        x,y := floor.Rooms[i].Pos()
        dx,dy := floor.Rooms[i].Dims()
        if int(bx) >= x && int(bx) < x + dx && int(by) >= y && int(by) < y + dy {
          hdt.viewer.Temp.Room = floor.Rooms[i]
          floor.Rooms[i] = floor.Rooms[len(floor.Rooms) - 1]
          floor.Rooms = floor.Rooms[0 : len(floor.Rooms) - 1]
          break
        }
      }
      if hdt.viewer.Temp.Room != nil {
        px,py := hdt.viewer.Temp.Room.Pos()
        hdt.drag_anchor.x = bx - float32(px)
        hdt.drag_anchor.y = by - float32(py)
      }
    }
    return true
  }

  return false
}
func (hdt *houseDataTab) Collapse() {}
func (hdt *houseDataTab) Expand() {}

func MakeHouseEditorPanel(house *houseDef, datadir string) Editor {
  var he HouseEditor
  he.HorizontalTable = gui.MakeHorizontalTable()
  he.viewer = MakeHouseViewer(house, 62)
  he.HorizontalTable.AddChild(he.viewer)

  r1 := MakeRoom("name")
  r2 := MakeRoom("name")
  r3 := MakeRoom("name")
  door := MakeDoor("Test Door 1")
  door.Facing = FarLeft
  door.Pos = 1
  r1.Doors = append(r1.Doors, door)
  door = MakeDoor("Test Door 2")
  door.Facing = FarLeft
  door.Pos = 3
  door.Opened = true
  r1.Doors = append(r1.Doors, door)
  door = MakeDoor("Test Door 1")
  door.Facing = FarLeft
  door.Pos = 6
  door.Opened = true
  r1.Doors = append(r1.Doors, door)
  door = MakeDoor("Test Door 2")
  door.Facing = FarLeft
  door.Pos = 9
  r1.Doors = append(r1.Doors, door)
  r1.X,r1.Y = 0,0
  r2.X,r2.Y = 20,5
  r3.X,r3.Y = 0,15
  house.Floors = append(house.Floors, &Floor{ Rooms: []*Room{ r1, r2, r3 }})
  he.widgets = append(he.widgets, makeHouseDataTab(house, he.viewer))
  var tabs []gui.Widget
  for _,w := range he.widgets {
    tabs = append(tabs, w.(gui.Widget))
  }
  he.tab = gui.MakeTabFrame(tabs)
  he.HorizontalTable.AddChild(he.tab)

  return &he
}

// Manually pass all events to the tabs, regardless of location, since the tabs
// need to know where the user clicks.
func (he *HouseEditor) Respond(ui *gui.Gui, group gui.EventGroup) bool {
  return he.widgets[he.tab.SelectedTab()].Respond(ui, group)
}