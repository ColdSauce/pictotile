//A simple tool for converting png, gif & jpg images into tiles useful for gameboy programming, with possible further applications.
//In the simplest case, tiles are to be read left to right then top to bottom, and each tile is to be individually converted into GBC 2-bit format.
//Future versions may include ability to write more than one tile together, e.g., so that each tile of a 16x16 image are writted one after the other.
//Color palettes are determined on a tile by tile basis. Colors are ordered by sum of RGB highest → lowest, with B > G > R being used for ties.
//Additional colors are set to black, and an error printed. With -t enabled the first color on each tile will be treated as transparency (color 0),
//overriding color sorting in this instance.
package main

import ("flag"
	"os"
 	"image"
	"log"
	"fmt"
	"image/color"
 	_ "image/png"
 	_ "image/jpeg"
	_ "image/gif")

	//Organise settings flags
	var dim uint
	var dimX uint
	var dimY uint
	var offset uint
	var offsetX uint
	var offsetY uint
	var spacing uint
	var spacingX uint
	var spacingY uint
	var spriteMode bool
	var delimiter string

type sImage interface {
	image.Image
	SubImage(r image.Rectangle) image.Image
}

func main() {
	var file *os.File
	var err error

	flag.UintVar(&dim, "d", 8, "Square dimension of each tile. Use only for square. Non multiple-of-8 values may cause undefined behaviour")
	flag.UintVar(&dimX, "w", 8, "Width of each tile")
	flag.UintVar(&dimY, "h", 8, "Height of each tile")
	flag.UintVar(&offset, "o", 0, "Offset of the first tile from both the top and left edge")
	flag.UintVar(&offsetX, "x", 0, "Horizontal offset of first tile from left")
	flag.UintVar(&offsetY, "y", 0, "Vertical offset of first tile from top")
	flag.UintVar(&spacing, "s", 0, "Distance between tiles")
	flag.UintVar(&spacingX, "sx", 0, "Horizontal distance between tiles")
	flag.UintVar(&spacingY, "sy", 0, "Vertical distance between tiles")
	flag.BoolVar(&spriteMode, "t", false, "Sets first color in image as transparency (color 0) for entire image")
	flag.StringVar(&delimiter, "c", "", "Sets any characters used to seperate values in output")
	flag.Parse();

	//if dimX, dimY are unset
	if dimX == dimY && dimY == 8 {
		//use the value from dim instead
		dimY = dim
		dimX = dim
	}
	if offsetX == offsetY && offsetY == 0 {
		offsetX = offset
		offsetY = offset
	}
	if spacingX == spacingY && spacingY == 0 {
		spacingX = spacing
		spacingY = spacing
	}

	//Program uses arg0 as read directory for input
	var fname = flag.Arg(0)

	//Default behaviour is read from stdin
	if fname == "-" || fname == "" {
		//read from standard input
		file = os.Stdin
	} else {
		//read from file
		file, err = os.Open(fname)
		if err != nil {
			log.Fatal(err)
		}
	}


	//decode file into image
	var outputData []byte
	tileset, format, err := image.Decode(file)
	if err == nil {
		log.Println(fname, "decoded from format", format)
	} else {
		log.Fatal(err)
	}
	tilesetSize := tileset.Bounds()

	//iterate through every tile fully contained within image
	for y := offsetY; y + dimY - 1 < uint(tilesetSize.Max.Y); y += dimY + spacingY {
		for x := offsetX; x + dimX - 1 < uint(tilesetSize.Max.X); x += dimX + spacingX {
			tile := tileset.(sImage).SubImage(image.Rect(int(x),int(y),int(x+dimX),int(y+dimY)))
			//Elipsis explodes the slice
			outputData = append(outputData, Encode(tile)...)
			//append slice to data
		}
	}
	//output data to file or stdOut
	var outFile *os.File
	if flag.Arg(1) == "-" || flag.Arg(1) == "" {
		outFile = os.Stdout
		log.Println("Outputting to stdout")
	} else {
		outFile, err = os.Create(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Outputting to file")
	}
	for i := 0; i < len(outputData); i++ {
		if i%16 == 0 && i != 0 {
			_, err = outFile.WriteString("\n")
		}
		_, err = fmt.Fprintf(outFile, "0x%X%s", []byte{outputData[i]}, delimiter)
		if err != nil {
			log.Fatal(err)
		}
		//_, err = outFile.WriteString(delimiter)
		//if err != nil {
		//	log.Fatal(err)
		//}
	}
	return
}



func Encode(tile image.Image) []byte {
	var palette [4]color.Color
	for i := range palette {
		palette[i] = color.Gray{0}
	}
	//var paletteMap map[color.Color]byte
	var colCount byte = 0;
	//Lets just see if Go supports this
	size := tile.Bounds()
	//Not a huge fan of using the globals here but size.Max.Y-size.Min.Y is hella messy
	var rawData = make([]byte, dimX*dimY)
	var data = make([]byte, dimX*dimY/4)
	//list all colors. Drop any colors more than 4
	for y := size.Min.Y; y < size.Max.Y; y++ {
		for x:= size.Min.X; x < size.Max.X; x++ {
			color := tile.At(x,y)
			colorFound := false
			for i := 0; i<int(colCount); i++ {
				if color == palette[i] {
					colorFound = true
					break
				}
			}
			if !colorFound {
				palette[colCount] = color
				colCount++
			}
			if colCount >= 4 {
				break
			}
		}
		if colCount >= 4 {
			break
		}
	}

	//sort colors (checking for -t)
	var min int = 0 //unnecessarily large because casting is annoying
	if spriteMode {
		min = 1
	}
	//Since it's such a small list, we're not checking if swapping is still
	//occurring
	for i := 0; i<4; i++ {
		for j := min; j<3-i; j++ {
			r0, g0, b0, _ := palette[j].RGBA()
			r1, g1, b1, _ := palette[j+1].RGBA()
			if r1 + g1 + b1 > r0 + g0 + b0 {
				palette[j], palette[j+1] = palette[j+1], palette[j]
			} else if r1 + g1 + b1 == r0 + g0 + b0 {
				if g1 + b1 > g0 + b0 {
					palette[j], palette[j+1] = palette[j+1], palette[j]
				} else if g1 + b1 == g0 + b0 {
					if b1 > b0 {
						palette[j], palette[j+1] = palette[j+1], palette[j]
					}
				}
			}
			//compare
		}
	}

	//create slice of color indices
	var pixelCount uint
	for y := size.Min.Y; y < size.Max.Y; y++ {
		for x:= size.Min.X; x < size.Max.X; x++ {
			var i byte
			for i = 0; i < 4; i++ {
				if tile.At(x,y) == palette[i] {
					break
				}
			}
			rawData[pixelCount] = i
			pixelCount++
		}
	}
	//loop until you find your color
	//set the index in the slice
	//"Encode" into gameboy format
	//for each row
	for i := 0; i < int(dimX*dimY/8); i += 1 {
		//for each pixel in the row
		for n:= 0; n<8; n++ {
			//I hope this works
			//First byte is less significant bits of first row
			data[2*i] = ((rawData[8*i+n] & 1) << (7-byte(n))) | data[2*i]
			//Second byte is more significant bits of second row
			data[2*i+1] = ((rawData[8*i+n] & 2) >> 1 << (7-byte(n))) | data[2*i+1]
		}
	}
	return data
}
