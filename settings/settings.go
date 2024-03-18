package settings

import (
	"encoding/json"
	"fmt"
	"os"
)

type Setting struct {
	Address string
	Port    string
	DbHost  string
	DbPort  string
	DbUser  string
	DbPass  string
	DbName  string
}

var settings Setting

func LoadFile(filename string) ([]byte, error) {
	var file *os.File
	var e error

	file, e = os.Open(filename)
	if e != nil {
		return nil, e
	}

	defer file.Close()

	stat, e := file.Stat()
	if e != nil {
		return nil, e
	}

	bs := make([]byte, stat.Size())
	_, e = file.Read(bs)
	if e != nil {
		return nil, e
	}

	return bs, nil
}

func Load(filename string) *Setting {
	bytes, e := LoadFile(filename)
	if e != nil {
		fmt.Println(e)
		return nil
	}
	e = json.Unmarshal(bytes, &settings)
	if e != nil {
		fmt.Println(e)
		return nil
	}
	return &settings
}
