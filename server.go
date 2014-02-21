package main

import(
	"fmt"
	"net/http"
	"io/ioutil"
	"math/rand"
	"math"
	"time"
	"strings"
	"flag"
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"reflect"
)

const (
	NUMBER_OF_PIECES = 9
	SHOE_SIZE = 17
)

type Board []interface{}

func NewBoard() Board {
	b := make(Board, 25)
	for i := range b {
		b[i] = make([]interface{}, 12)
	}
	return b
}

type Player struct {
	ID uint32
	WS	*websocket.Conn
	Board
	Deck int64
	Message map[string]interface{}
	Enc *json.Encoder
	Dec *json.Decoder
	Live bool
}

func NewPlayer(ws *websocket.Conn) *Player {
	return &Player{ID: 0, WS: ws, Board: NewBoard(), Deck: 0, Message: make(map[string]interface{}, 2), Enc: json.NewEncoder(ws), Dec: json.NewDecoder(ws), Live: true}	
}

func (p *Player) NewMessage(label string, data interface{}) {
	p.Message["label"] = label
	p.Message["data"] = data
}

func (b Board) ClearFullLines() (linesCleared int) {
	linesCleared = 0
	for i := len(b) - 2; i >= 0; i-- {
		x := int64(1)
		for _,w := range(b[i].([]interface{})) {
			x *= int64(w.(float64))
		}
		if x > 0 { 
			linesCleared++
			for j := i; j < len(b)-1; j++ {
				b[j] = b[j+1]
			}
			b[len(b)-1] = make([]interface{}, 12)
		}
	}
	return linesCleared
}

func registerPlayer(ws *websocket.Conn) {
	println("RegisterPlayer")
	p := NewPlayer(ws)
	defer func() {
		fmt.Println("Closing connection ...")
		p.Live = false
		playerChan<-p
		ws.Close()
	}()
	var v interface{}
	playerChan<-p
	
	for {
		err := p.Dec.Decode(&v)
		if err != nil {
			fmt.Println("Error on line 94: ", err)
			break
		}
		if v != nil {
			if err != nil {
				fmt.Println("Error line 68: ", err)
			}
			p.NewMessage(((reflect.ValueOf(v).MapIndex(reflect.ValueOf("label"))).Interface()).(string),(reflect.ValueOf(v).MapIndex(reflect.ValueOf("data"))).Interface())
			playerChan<-p
		}
	}
}

type Game struct {
	CurrentPlayers map[*websocket.Conn]*Player
	MessageOut map[string]interface{}
	R *rand.Rand
	Decks []Deck
	HasStarted bool
}

type Deck int64

func NewGame() *Game {
	cP := make(map[*websocket.Conn]*Player, 5)
	m := make(map[string]interface{}, 2)
	t := time.Now()
	r := rand.New(rand.NewSource(int64(t.Nanosecond())))
	d := make([]Deck,0,1)
	d = append(d, Deck(r.Int63n(int64(math.Gamma(float64(SHOE_SIZE+1))))))
	return &Game{CurrentPlayers: cP, MessageOut: m, R: r, Decks: d, HasStarted: false}
}

func (g *Game) Broadcast(ps []*Player, label string, data interface{}) {
	g.MessageOut["label"] = interface{}(label)
	g.MessageOut["data"] = data
	for _, p := range ps {
		err := p.Enc.Encode(g.MessageOut)
		if err != nil {
			fmt.Println("error to player ", p.ID, ": ",err)
		}
	}
}

// Broadcast wrapper functions for readability. 

func (g *Game) SendPieces(p *Player) {
	for i := int64(len(g.Decks)) - 1; i <= p.Deck; i++ {
		g.Decks = append(g.Decks, Deck(g.R.Int63n(int64(math.Gamma(float64(SHOE_SIZE+1))))))
	}
	g.Broadcast(g.OnlyPlayer(p),"pieces",interface{}((g.Decks[p.Deck]).Deal(SHOE_SIZE)))
	p.Deck++
}

func (g *Game) BroadcastBoard(p *Player) {
	data := make(map[string]interface{}, 2)
	data["player"] = interface{}(p.ID)
	data["board"] = interface{}(p.Board)

	g.Broadcast(g.AllPlayersExcept(p), "board", data)
}

func (g *Game) SendLines(p *Player, lines int) {
	if lines--; lines > 0 {
		g.Broadcast(g.AllPlayersExcept(p), "lines", lines)
	}
}

func (g *Game) AnnouncePlayers() {
	for _, c := range g.CurrentPlayers {
		g.Broadcast(g.AllPlayersExcept(c), "newPlayer", c.ID)
	}
}

// Quick player subset selectors for readability. 

func (g *Game) AllPlayersExcept(p *Player) (ps []*Player) {
	ps = make([]*Player, 0)
	for _,v := range g.CurrentPlayers {
		if v.WS != p.WS {
			ps = append(ps, v)
		}
	}
	return
}

func (g *Game) OnlyPlayer(p *Player) (ps []*Player) {
	ps = make([]*Player,0)
	ps = append(ps, p)
	return
}

func (g *Game) AllPlayers() (ps []*Player) {
	ps = make([]*Player, 0)
	for _,v := range g.CurrentPlayers {
		ps = append(ps, v)
	}
	return
}

// Game server. 

func GameServer() {
	game := NewGame()

	for {
		p := <-playerChan 
		if game.HasStarted {
			if p.Live {
				switch p.Message["label"] {
					case interface{}("request"):
						switch p.Message["data"] {
							case interface{}("sendPieces"):
								game.SendPieces(p)
								break
						}
						break
					case interface{}("board"):
						p.Board = p.Message["data"].([]interface{})
						game.SendLines(p, p.Board.ClearFullLines())
						game.BroadcastBoard(p)
						break
					case interface{}("debug"):
						fmt.Println(p.Message["data"])
						break
				}
			} else {
				delete(game.CurrentPlayers, p.WS)
				game.Broadcast(game.AllPlayersExcept(p), "removePlayer", interface{}(p.ID))
				if len(game.CurrentPlayers) == 0 {
					game = NewGame()
				}
			}
		} else {
			// accept new players, give first deck, send notice of new players
			if p.Live {
				if _, ok := game.CurrentPlayers[p.WS]; !ok {
					delete(game.CurrentPlayers, p.WS)
					p.ID = game.R.Uint32()
					game.SendPieces(p)
					game.AnnouncePlayers()
				}
				if p.Message["label"] == interface{}("request") && p.Message["data"] == interface{}("startGame") {
					game.Broadcast(game.AllPlayers(), "request", interface{}("startGame"))
					game.HasStarted = true
				}
			} else {
				delete(game.CurrentPlayers, p.WS)
				game.Broadcast(game.AllPlayersExcept(p), "removePlayer", interface{}(p.ID))
			}
		}
	}
}

func (deck Deck) Deal(d int) (v []int) {
	n := deck
	fc := make([]int, d) // factoradic coefficients
	fd := make([]int, d) // fresh deck
	v = make([]int, 0)

	for i, _ := range fd {
		fd[i] = i+1
	}
	for i := 1; i <= d; i++ {
		fc[i-1], n = int(n % Deck(i)), n / Deck(i)
	}
	for i := 0; i < d; i++ {
		v = append(v, fd[fc[d-1-i]])
		fd = append(fd[:fc[d-1-i]], fd[fc[d-1-i]+1:]...)
	}
	return v
}

var pPathLength = len("/blickles/p/")
var playerChan = make(chan *Player)
var addr = flag.String("addr",":8081","http service address")

func pngs(w http.ResponseWriter, r *http.Request) {
	pic := r.URL.Path[pPathLength:]
	http.ServeFile(w, r, pic)
}

func css(w http.ResponseWriter, r *http.Request) {
	pic := r.URL.Path[pPathLength:]
	http.ServeFile(w, r, pic)
}


func client(w http.ResponseWriter, r *http.Request) {
	clientmaker()
	page, _ := ioutil.ReadFile("index.html")
	fmt.Fprint(w, string(page))
}

func devclient(w http.ResponseWriter, r *http.Request) {
	page, _ := ioutil.ReadFile("blickles-dev.html")
	fmt.Fprint(w, string(page))
}

func clientmaker() {
	wholefile, _ := ioutil.ReadFile("blickles-dev.html")
	sFile := string(wholefile)
	sFileByLine := strings.Split(sFile, "\n")
	var parsedBFile []byte
	for _, l := range sFileByLine {
		if strings.Index(l,"DBGGR") == -1 {
			parsedBFile = append(parsedBFile, []byte(fmt.Sprintln(l))...)
		}
	}
	ioutil.WriteFile("index.html", parsedBFile, 0600)
}

func main() {
	flag.Parse()
	go GameServer()
	clientmaker()
	http.Handle("/register", websocket.Handler(registerPlayer))
	http.HandleFunc("/", client)
	http.HandleFunc("/blickles/p/", pngs)
	http.HandleFunc("/s.css", css)
	http.HandleFunc("/dev", devclient)
	err := http.ListenAndServe(*addr,nil)
	if err != nil {
		fmt.Println("Error with ListenAndServe: ",err)
	}
}
