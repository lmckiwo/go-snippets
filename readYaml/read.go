package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// "github.com/bndr/gojenkins"
// "honnef.co/go/tools/config"

type Configuration struct {
	Server struct {
		BaseURL string `yaml:"baseUrl"`
	} `yaml:"server"`
	Token    string `yaml:"token"`
	Projects struct {
		Name string `yaml:"name"`
	} `yaml:"projects"`
}

func main() {
	// config:=config.
	// jenkins := gojenkins.CreateJenkins()

	var configuration Configuration
	// configuration := make(map[interface{}]interface{})

	c, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		fmt.Println(err)
	}

	err = yaml.Unmarshal(c, &configuration)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(configuration)
	fmt.Println(configuration.Server)
	fmt.Println("token: " + configuration.Token)
	fmt.Println("project: ", configuration.Projects)

	// for k,v := range configuration {
	//     fmt.Printf("%s -> %d\n", k, v)
	// }

}
