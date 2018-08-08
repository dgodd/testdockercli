package main

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	tmpDir, err := ioutil.TempDir("", "testdockercli.")
	checkErr(err)
	afs := afero.NewBasePathFs(afero.NewOsFs(), tmpDir)
	fmt.Println("TEMP DIR:", tmpDir)

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}
	res1, err := httpc.Get("http://unix/images/myorg/myapp/json")
	checkErr(err)
	io.Copy(os.Stdout, res1.Body)

	res2, err := httpc.Get("http://unix/images/myorg/myapp/get")
	checkErr(err)
	fmt.Printf("RES: %#v\n", res2)

	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)
		tr := tar.NewReader(res2.Body)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break // End of archive
			}
			if err != nil {
				log.Fatal(err)
			}
			if hdr.Typeflag == tar.TypeDir {
				continue
			}

			if false {
				fmt.Printf("File: %s\n", hdr.Name)
				checkErr(afs.MkdirAll(filepath.Dir(hdr.Name), 0755))
				fh, err := afs.Create(hdr.Name)
				checkErr(err)
				if _, err := io.Copy(fh, tr); err != nil {
					log.Fatal(err)
				}
				fh.Close()
			} else if true {
				// Print data to screen
				if strings.HasSuffix(hdr.Name, ".tar") {
					fmt.Printf("File: %s\n", hdr.Name)
				} else {
					fmt.Printf("Contents of %s:\n", hdr.Name)
					if _, err := io.Copy(os.Stdout, tr); err != nil {
						log.Fatal(err)
					}
					fmt.Println()
				}
			} else {
				// Copy to new location
				fmt.Printf("File: %s\n", hdr.Name)
				if hdr.Name == "repositories" || hdr.Name == "manifest.json" {
					var txt string
					if hdr.Name == "manifest.json" {
						txt = `[{"Config":"c629769d361c37bc177bc5e5ef11455045e49c37e018b14abd525f00e9a2ef51.json","RepoTags":["myorg/myapp-dgodd:latest"],"Layers":["8fc0be68bf52ca75cf0a7265b143cacd56004bf99a058090fe01a44d43661c60/layer.tar","967d31b3861fe134f9579c29731ad466e91bd9ad80bf5f59aae19b87bf6b188b/layer.tar","02c610a6b119d43a44ee7e79e405c0d1281f987ce742c8b364ade1c56b82e98b/layer.tar","1f5380b75df72ce96b29baf8cc699ad45ab863c088e339953bc351852678ba97/layer.tar","fa102bc0ad8c3ff68e71653f06d817856c1f6895082d6b4a143ea9788740e711/layer.tar","9fdd69644fb74d3ceb56de117613a1aa7e785465330eb5e4cfcb0aa4dc9a2508/layer.tar","326210a9f1eec18f8d2dd8e851d9d2570aefa5ab1796381e7e756efb47a5442b/layer.tar","11be762a0aaaf354bda69937936d4db04a2654a5ba3d59120990e659ec20c73f/layer.tar","cfff9f16f307d8d66c72735438a9a6589801b2d243966ed049413b59c28f28ec/layer.tar","ceda91807de502dc26e4965d1385c38272c16a3bac9b5e7382af4803124158fa/layer.tar","119fcddb7abdaf87c7de66759b43d97dd3cecbe891baf51d9beeac896200601d/layer.tar","577460197cde1f92e2017a4c1ce0f19c320c741502ea8169ba701dcab0257154/layer.tar","3e2d3956e5922a53c7664cff47d7505c085a4d65c6efdbaeee397739a6b9a037/layer.tar","58aa57119a10db3a096ccdc3e7d3ecfe676b67efaa21f8709be7266dd0e2534d/layer.tar"]}]`
					} else {
						txt = `{"myorg/myapp-dgodd":{"latest":"58aa57119a10db3a096ccdc3e7d3ecfe676b67efaa21f8709be7266dd0e2534d"}}`
					}
					hdr.Size = int64(len(txt))
					if err := tw.WriteHeader(hdr); err != nil {
						log.Fatal(err)
					}
					if _, err := tw.Write([]byte(txt)); err != nil {
						log.Fatal(err)
					}
				} else {
					if err := tw.WriteHeader(hdr); err != nil {
						log.Fatal(err)
					}
					if _, err := io.Copy(tw, tr); err != nil {
						log.Fatal(err)
					}
				}
			}
		}
		tw.Close()
		pw.Close()
	}()

	res3, err := httpc.Post("http://unix/images/load", "application/x-tar", pr)
	checkErr(err)
	io.Copy(os.Stdout, res3.Body)
}

func main2() {
	// tag := "ubuntu:18.04"
	tag := "myorg/myapp"

	cli, err := client.NewEnvClient()
	checkErr(err)
	ctx := context.Background()
	ii, _, err := cli.ImageInspectWithRaw(ctx, tag)
	checkErr(err)
	dump(ii)
	// fmt.Println(string(b))

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
