package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/client"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	// tag := "ubuntu:18.04"
	tag := "myorg/myapp"

	cli, err := client.NewEnvClient()
	checkErr(err)
	ctx := context.Background()
	ii, b, err := cli.ImageInspectWithRaw(ctx, tag)
	checkErr(err)
	dump(ii)
	fmt.Println(string(b))

	var data interface{}
	err = json.Unmarshal([]byte(ii.Config.Labels["sh.packs.build"]), &data)
	checkErr(err)
	dump(data)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func dump(i interface{}) {
	s, err := yaml.Marshal(i)
	checkErr(err)
	fmt.Println(string(s))
}
