package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	parser "github.com/buger/jsonparser"
)

func main() {

	fileName := flag.String("n", "empty.log", "Log File")
	filter := flag.String("f", "filter.txt", "Filter File")
	valueType := flag.Bool("t", true, "Value Type")
	arrayType := flag.Bool("a", true, "Array Type")
	output := flag.String("o", "output", "Output File Name")
	flag.Parse()

	fi, err := os.Open(*fileName)
	if err != nil {
		panic(err)
	} else {
		defer fi.Close()
	}

	condition, err := os.Open(*filter)
	if err != nil {
		panic(err)
	} else {
		defer condition.Close()
	}

	out, _ := os.Create(*output)

	key, keyVals := filterParser(condition)

	scanner := bufio.NewScanner(fi)
	scanner.Split(bufio.ScanLines)

	// Read one line
	for scanner.Scan() {
		check := true
		data := scanner.Bytes()

		for key, val := range keyVals {
			result, _ := parser.GetString(data, key)

			if result == val {
				continue
			} else {
				check = false
				break
			}
		}

		if check {
			var log string
			time, _ := parser.GetFloat(data, "time")

			if *arrayType {

				newKey := make([]string, len(key)-1)

				copy(newKey, key[:len(key)])

				parser.ArrayEach(data, func(value []byte, dataType parser.ValueType, offset int, err error) {
					if *valueType {
						if num, err := parser.GetInt(value, key[len(key)-1]); err == nil {
							log = fmt.Sprintf("%f, %d\n", time, num)
						}
					} else {
						if num, err := parser.GetFloat(value, key[len(key)-1]); err == nil {
							log = fmt.Sprintf("%f, %f\n", time, num)
						}
					}
				}, newKey...)

			} else {
				if *valueType {
					num, _ := parser.GetInt(data, key...)
					log = fmt.Sprintf("%f, %d\n", time, num)
				} else {
					num, _ := parser.GetFloat(data, key...)
					log = fmt.Sprintf("%f, %f\n", time, num)
				}
			}

			out.Write([]byte(log))
		}
	}
}

func filterParser(fiter *os.File) ([]string, map[string]string) {

	key := make([]string, 0)
	keyVals := make(map[string]string)

	scanner := bufio.NewScanner(fiter)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()

		data := strings.Split(line, " ")

		if data[1] == "=" {
			keyVals[data[0]] = data[2]
		} else {
			for i := 0; i < len(data); i += 2 {
				key = append(key, data[i])
				fmt.Printf("key:%s/%d\n", data[i], len(key))
			}
		}
	}

	return key, keyVals
}
