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

func (c *Configuration) readYaml(yamlFile string) *Configuration {

	f, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		fmt.Println(err)
	}

	err = yaml.Unmarshal(f, c)
	if err != nil {
		fmt.Println(err)
	}

	return c
}
func main() {
	// config:=config.
	// jenkins := gojenkins.CreateJenkins()

	var configuration Configuration
	// configuration := make(map[interface{}]interface{})

	c := configuration.readYaml("config.yaml")

	fmt.Println(c)
	fmt.Println(c.Server)
	fmt.Println("token: " + c.Token)
	fmt.Println("project: ", c.Projects)

	// for k,v := range configuration {
	//     fmt.Printf("%s -> %d\n", k, v)
	// }

}
