package main

import (
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"

	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"bufio"
	"flag"
)

const (
	width              = 500
	height             = 500
	vertexShaderSource = `
	#version 410
	in vec3 vp;
	void main() {
		gl_Position = vec4(vp, 1.0);
	}
	` + "\x00"

	fragmentShaderSource = `
	#version 410
	out vec4 frag_colour;
	void main() {
		frag_colour = vec4(0, 0, 1, 1);
	}
	` + "\x00"
)

type cell struct {
	drawable uint32

	isAlive     bool
	isAliveNext bool

	x int
	y int
}

var (
	triangle = []float32{
		0, 0.5, 0, //top
		-0.5, -0.5, 0, //left
		0.5, -0.5, 0, // right
	}

	square = []float32{
		-0.5, 0.5, 0, //top
		-0.5, -0.5, 0, //left
		0.5, -0.5, 0, // right

		-0.5, 0.5, 0,
		0.5, 0.5, 0,
		0.5, -0.5, 0,
	}

	fps = 2
	rows    = 10
	columns = 10
)

func main() {
	runtime.LockOSThread()

	var filepath string

	flag.IntVar(&fps, "fps", 2, "FPS")
	flag.StringVar(&filepath, "path", "", "Filepath to map file")
	flag.Parse()
	fmt.Println(fps)	

	rand.Seed(time.Now().UTC().UnixNano())

	window := initGlfw()
	defer glfw.Terminate()

	program := initOpenGL()

	//vao := makeVao(square)
	var cells [][]*cell
	if filepath != ""{
		cells = LoadMap(filepath)
	} else {
		cells = makeCells()
	}

	for !window.ShouldClose() {
		t := time.Now()

		checkCellsAlive(cells)

		draw(cells, window, program)

		time.Sleep(time.Second/time.Duration(fps) - time.Since(t))
	}
}

//init GlFw initializes glfw and returns a window
func initGlfw() *glfw.Window {
	if err := glfw.Init(); err != nil {
		panic(err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(width, height, "Test", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	return window
}

// initOpenGL initializes OpenGL and returns an intiialized program.
func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)
	gl.LinkProgram(prog)
	return prog
}

func draw(cells [][]*cell, window *glfw.Window, program uint32) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(program)

	for x := range cells {
		for _, c := range cells[x] {
			if c.isAlive {
				c.draw()
			}
		}
	}
	glfw.PollEvents()
	window.SwapBuffers()
}

func (c *cell) draw() {

	gl.BindVertexArray(c.drawable)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(square))/3)
}

// makeVao initializes and returns a vertex array from the points provided.
func makeVao(points []float32) uint32 {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	return vao
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func makeCells() [][]*cell {
	cells := make([][]*cell, rows, rows)
	for x := 0; x < rows; x++ {
		for y := 0; y < columns; y++ {
			c := newCell(x, y, rand.Intn(5)%5 == 0)
			cells[x] = append(cells[x], c)
		}
	}
	return cells
}

func LoadMap(filepath string) [][]*cell {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println(err)
	}
	reader := bufio.NewReader(file)

	xb,_ := reader.ReadString('\n')
	yb,_ := reader.ReadString('\n')

	x, errx := strconv.Atoi(xb[:len(xb)-1])
	y, erry := strconv.Atoi(yb[:len(yb)-1])

	if errx != nil || erry != nil {
		fmt.Println(errx, erry)
	}

	rows = x
	columns = y

	fmt.Println(x, y)

	cells := make([][]*cell, x, y)

	for cx := 0; cx < x; cx++{
		fmt.Println("")
		aliveVals,_ := reader.ReadBytes('\n')
		fmt.Println(aliveVals)
		cy := 0
		for _,aliveB := range aliveVals {
			if aliveB == ' ' ||  aliveB == '\n'{
				fmt.Print(" ")
				continue
			}
			fmt.Print(string(aliveB))
			c := newCell(cx, cy, aliveB == '1')
			cells[cx] = append(cells[cx], c)
			cy++
		}
		fmt.Println(len(cells[cx]))
	}
	file.Close()
	return cells
}

func newCell(x, y int, alive bool) *cell {
	points := make([]float32, len(square), len(square))
	copy(points, square)

	for i := 0; i < len(points); i++ {
		var position float32
		var size float32
		switch i % 3 {
		case 0:
			size = 1.0 / float32(columns)
			position = float32(x) * size
		case 1:
			size = 1.0 / float32(rows)
			position = float32(y) * size
		default:
			continue
		}
		if points[i] < 0 {
			points[i] = (position * 2) - 1
		} else {
			points[i] = ((position + size) * 2) - 1
		}
	}

	return &cell{
		drawable: makeVao(points),

		isAlive:     alive,
		isAliveNext: alive,

		x: x,
		y: y,
	}
}

func checkCellsAlive(cells [][]*cell) {
	for _, row := range cells {
		for _, c := range row {
			c.checkState(cells)
		}
	}
	for _, row := range cells {
		for _, c := range row {
			c.isAlive = c.isAliveNext
		}
	}
}

func (c *cell) checkState(cells [][]*cell) {

	liveCount := c.liveNeighbors(cells)
	if c.isAlive {
		if liveCount < 2 || liveCount > 3 {
			c.isAliveNext = false
		} else {
			c.isAliveNext = true
		}
	} else if liveCount == 3 {
		c.isAliveNext = true
	}
}

func (c *cell) liveNeighbors(cells [][]*cell) int {
	var liveCount int
	add := func(x, y int) {
		if x == len(cells) {
			x = 0
		} else if x == -1 {
			x = len(cells) - 1
		}
		if y == len(cells[x]) {
			y = 0
		} else if y == -1 {
			y = len(cells[x]) - 1
		}
		if cells[x][y].isAlive {
			liveCount++
		}
	}
	add(c.x-1, c.y)
	add(c.x+1, c.y)
	add(c.x, c.y-1)
	add(c.x, c.y+1)
	add(c.x-1, c.y-1)
	add(c.x-1, c.y+1)
	add(c.x+1, c.y-1)
	add(c.x+1, c.y+1)
	return liveCount
}
