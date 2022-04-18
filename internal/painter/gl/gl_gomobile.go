//go:build (android || ios || mobile) && (!js || !wasm || !test_web_driver)
// +build android ios mobile
// +build !js !wasm !test_web_driver

package gl

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"math"

	"fyne.io/fyne/v2/internal/driver/mobile/gl"
)

const (
	bitColorBuffer   = gl.ColorBufferBit
	bitDepthBuffer   = gl.DepthBufferBit
	clampToEdge      = gl.ClampToEdge
	colorFormatRGBA  = gl.RGBA
	scissorTest      = gl.ScissorTest
	texture0         = gl.Texture0
	texture2D        = gl.Texture2D
	textureMinFilter = gl.TextureMinFilter
	textureMagFilter = gl.TextureMagFilter
	textureWrapS     = gl.TextureWrapS
	textureWrapT     = gl.TextureWrapT
	unsignedByte     = gl.UnsignedByte
)

// Buffer represents a GL buffer
type Buffer gl.Buffer

// Program represents a compiled GL program
type Program gl.Program

var textureFilterToGL = []int32{gl.Linear, gl.Nearest}

func (p *painter) glctx() gl.Context {
	return p.contextProvider.Context().(gl.Context)
}

func (p *painter) compileShader(source string, shaderType gl.Enum) (gl.Shader, error) {
	shader := p.glctx().CreateShader(shaderType)

	p.glctx().ShaderSource(shader, source)
	p.logError()
	p.glctx().CompileShader(shader)
	p.logError()

	status := p.glctx().GetShaderi(shader, gl.CompileStatus)
	if status == gl.False {
		info := p.glctx().GetShaderInfoLog(shader)

		return shader, fmt.Errorf("failed to compile %v: %v", source, info)
	}

	return shader, nil
}

var vertexShaderSource = string(shaderSimpleesVert.StaticContent)
var fragmentShaderSource = string(shaderSimpleesFrag.StaticContent)
var vertexLineShaderSource = string(shaderLineesVert.StaticContent)
var fragmentLineShaderSource = string(shaderLineesFrag.StaticContent)

func (p *painter) Init() {
	p.ctx = &mobileContext{glContext: p.contextProvider.Context().(gl.Context)}
	p.glctx().Disable(gl.DepthTest)
	p.glctx().Enable(gl.Blend)

	vertexShader, err := p.compileShader(vertexShaderSource, gl.VertexShader)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := p.compileShader(fragmentShaderSource, gl.FragmentShader)
	if err != nil {
		panic(err)
	}

	prog := p.glctx().CreateProgram()
	p.glctx().AttachShader(prog, vertexShader)
	p.glctx().AttachShader(prog, fragmentShader)
	p.glctx().LinkProgram(prog)
	p.logError()

	p.program = Program(prog)
	p.logError()

	vertexLineShader, err := p.compileShader(vertexLineShaderSource, gl.VertexShader)
	if err != nil {
		panic(err)
	}
	fragmentLineShader, err := p.compileShader(fragmentLineShaderSource, gl.FragmentShader)
	if err != nil {
		panic(err)
	}

	lineProg := p.glctx().CreateProgram()
	p.glctx().AttachShader(lineProg, vertexLineShader)
	p.glctx().AttachShader(lineProg, fragmentLineShader)
	p.glctx().LinkProgram(lineProg)
	p.logError()

	p.lineProgram = Program(lineProg)
	p.logError()
}

func (p *painter) glScissorClose() {
	p.glctx().Disable(gl.ScissorTest)
	p.logError()
}

func (p *painter) glCreateBuffer(points []float32) Buffer {
	ctx := p.glctx()

	p.ctx.UseProgram(p.program)

	buf := ctx.CreateBuffer()
	p.logError()
	ctx.BindBuffer(gl.ArrayBuffer, buf)
	p.logError()
	ctx.BufferData(gl.ArrayBuffer, f32Bytes(binary.LittleEndian, points...), gl.DynamicDraw)
	p.logError()

	vertAttrib := ctx.GetAttribLocation(gl.Program(p.program), "vert")
	ctx.EnableVertexAttribArray(vertAttrib)
	ctx.VertexAttribPointer(vertAttrib, 3, gl.Float, false, 5*4, 0)
	p.logError()

	texCoordAttrib := ctx.GetAttribLocation(gl.Program(p.program), "vertTexCoord")
	ctx.EnableVertexAttribArray(texCoordAttrib)
	ctx.VertexAttribPointer(texCoordAttrib, 2, gl.Float, false, 5*4, 3*4)
	p.logError()

	return Buffer(buf)
}

func (p *painter) glCreateLineBuffer(points []float32) Buffer {
	ctx := p.glctx()

	p.ctx.UseProgram(p.lineProgram)

	buf := ctx.CreateBuffer()
	p.logError()
	ctx.BindBuffer(gl.ArrayBuffer, buf)
	p.logError()
	ctx.BufferData(gl.ArrayBuffer, f32Bytes(binary.LittleEndian, points...), gl.DynamicDraw)
	p.logError()

	vertAttrib := ctx.GetAttribLocation(gl.Program(p.lineProgram), "vert")
	ctx.EnableVertexAttribArray(vertAttrib)
	ctx.VertexAttribPointer(vertAttrib, 2, gl.Float, false, 4*4, 0)
	p.logError()

	normalAttrib := ctx.GetAttribLocation(gl.Program(p.lineProgram), "normal")
	ctx.EnableVertexAttribArray(normalAttrib)
	ctx.VertexAttribPointer(normalAttrib, 2, gl.Float, false, 4*4, 2*4)
	p.logError()

	return Buffer(buf)
}

func (p *painter) glFreeBuffer(b Buffer) {
	ctx := p.glctx()

	ctx.BindBuffer(gl.ArrayBuffer, gl.Buffer(b))
	p.logError()
	ctx.DeleteBuffer(gl.Buffer(b))
	p.logError()
}

func (p *painter) glDrawTexture(texture Texture, alpha float32) {
	ctx := p.glctx()

	p.ctx.UseProgram(p.program)

	// here we have to choose between blending the image alpha or fading it...
	// TODO find a way to support both
	if alpha != 1.0 {
		ctx.BlendColor(0, 0, 0, alpha)
		ctx.BlendFunc(gl.ConstantAlpha, gl.OneMinusConstantAlpha)
	} else {
		ctx.BlendFunc(1, gl.OneMinusSrcAlpha)
	}
	p.logError()

	p.ctx.ActiveTexture(texture0)
	p.ctx.BindTexture(texture2D, texture)
	p.logError()

	ctx.DrawArrays(gl.TriangleStrip, 0, 4)
	p.logError()
}

func (p *painter) glDrawLine(width float32, col color.Color, feather float32) {
	ctx := p.glctx()

	p.ctx.UseProgram(p.lineProgram)

	ctx.BlendFunc(gl.SrcAlpha, gl.OneMinusSrcAlpha)
	p.logError()

	colorUniform := ctx.GetUniformLocation(gl.Program(p.lineProgram), "color")
	r, g, b, a := col.RGBA()
	if a == 0 {
		ctx.Uniform4f(colorUniform, 0, 0, 0, 0)
	} else {
		alpha := float32(a)
		col := []float32{float32(r) / alpha, float32(g) / alpha, float32(b) / alpha, alpha / 0xffff}
		ctx.Uniform4fv(colorUniform, col)
	}
	lineWidthUniform := ctx.GetUniformLocation(gl.Program(p.lineProgram), "lineWidth")
	ctx.Uniform1f(lineWidthUniform, width)

	featherUniform := ctx.GetUniformLocation(gl.Program(p.lineProgram), "feather")
	ctx.Uniform1f(featherUniform, feather)

	ctx.DrawArrays(gl.Triangles, 0, 6)
	p.logError()
}

func (p *painter) glCapture(width, height int32, pixels *[]uint8) {
	p.glctx().ReadPixels(*pixels, 0, 0, int(width), int(height), gl.RGBA, gl.UnsignedByte)
	p.logError()
}

func (p *painter) glInit() {
}

// f32Bytes returns the byte representation of float32 values in the given byte
// order. byteOrder must be either binary.BigEndian or binary.LittleEndian.
func f32Bytes(byteOrder binary.ByteOrder, values ...float32) []byte {
	le := false
	switch byteOrder {
	case binary.BigEndian:
	case binary.LittleEndian:
		le = true
	default:
		panic(fmt.Sprintf("invalid byte order %v", byteOrder))
	}

	b := make([]byte, 4*len(values))
	for i, v := range values {
		u := math.Float32bits(v)
		if le {
			b[4*i+0] = byte(u >> 0)
			b[4*i+1] = byte(u >> 8)
			b[4*i+2] = byte(u >> 16)
			b[4*i+3] = byte(u >> 24)
		} else {
			b[4*i+0] = byte(u >> 24)
			b[4*i+1] = byte(u >> 16)
			b[4*i+2] = byte(u >> 8)
			b[4*i+3] = byte(u >> 0)
		}
	}
	return b
}

type mobileContext struct {
	glContext gl.Context
}

var _ context = (*mobileContext)(nil)

func (c *mobileContext) ActiveTexture(textureUnit uint32) {
	c.glContext.ActiveTexture(gl.Enum(textureUnit))
}

func (c *mobileContext) BindTexture(target uint32, texture Texture) {
	c.glContext.BindTexture(gl.Enum(target), gl.Texture(texture))
}

func (c *mobileContext) Clear(mask uint32) {
	c.glContext.Clear(gl.Enum(mask))
}

func (c *mobileContext) ClearColor(r, g, b, a float32) {
	c.glContext.ClearColor(r, g, b, a)
}

func (c *mobileContext) CreateTexture() (texture Texture) {
	return Texture(c.glContext.CreateTexture())
}

func (c *mobileContext) DeleteTexture(texture Texture) {
	c.glContext.DeleteTexture(gl.Texture(texture))
}

func (c *mobileContext) Enable(capability uint32) {
	c.glContext.Enable(gl.Enum(capability))
}

func (c *mobileContext) GetError() uint32 {
	return uint32(c.glContext.GetError())
}

func (c *mobileContext) Scissor(x, y, w, h int32) {
	c.glContext.Scissor(x, y, w, h)
}

func (c *mobileContext) TexImage2D(target uint32, level, width, height int, colorFormat, typ uint32, data []uint8) {
	c.glContext.TexImage2D(
		gl.Enum(target),
		level,
		int(colorFormat),
		width,
		height,
		gl.Enum(colorFormat),
		gl.Enum(typ),
		data,
	)
}

func (c *mobileContext) TexParameteri(target, param uint32, value int32) {
	c.glContext.TexParameteri(gl.Enum(target), gl.Enum(param), int(value))
}

func (c *mobileContext) UseProgram(program Program) {
	c.glContext.UseProgram(gl.Program(program))
}

func (c *mobileContext) Viewport(x, y, width, height int) {
	c.glContext.Viewport(x, y, width, height)
}
