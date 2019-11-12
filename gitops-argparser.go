package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// our actual fixed/known arguments
var (
	argParserConfigFile string = "config.yaml" // can be overriden w/ ENV var: COMMIT_MSG_ARGPARSER_CONFIG_FILE
	argParserTmplFile   string = "output.tmpl" // can be overriden w/ ENV var: COMMIT_MSG_ARGPARSER_OUTPUT_TMPL_FILE
)

// ArgumentValue ... used in templates
type ArgumentValue struct {
	Name  string      // the argument name
	Value interface{} // the argument value
}

type argumentDef struct {
	Long         string      `yaml:"long"`
	DataType     string      `yaml:"dataType"`
	Help         string      `yaml:"help"`
	DefaultValue interface{} `yaml:"defaultValue"`
}

type argumentsConf struct {
	Arguments []argumentDef
}

func loadOutputTemplateFile() *template.Template {

	// load the output golang tmpl from ENV var location if it exists
	// otherwise the default is defined above
	envVal, exists := os.LookupEnv("COMMIT_MSG_ARGPARSER_OUTPUT_TMPL_FILE")
	if exists {
		argParserTmplFile = envVal
	}

	log.Debugf("loadOutputTemplateFile(): reading argparser output template from: %v", argParserTmplFile)

	// weird: https://stackoverflow.com/questions/49043292/error-template-is-an-incomplete-or-empty-template
	tmpl, err := template.New(path.Base(argParserTmplFile)).Funcs(sprig.TxtFuncMap()).ParseFiles(argParserTmplFile)
	if err != nil {
		log.Fatalf("loadOutputTemplateFile(): template.ParseFiles err #%v ", err)
	}

	return tmpl
}

func loadArgumentsConf() *argumentsConf {

	// load the arguments config from ENV var location if it exists
	// otherwise the default is defined above
	envVal, exists := os.LookupEnv("COMMIT_MSG_ARGPARSER_CONFIG_FILE")
	if exists {
		argParserConfigFile = envVal
	}

	log.Debugf("loadArgumentsConf(): reading argparser arguments conf from: %v", argParserConfigFile)

	yamlFile, err := ioutil.ReadFile(argParserConfigFile)
	if err != nil {
		log.Fatalf("loadArgumentsConf(): yamlFile.Get err #%v ", err)
	}

	argsConf := &argumentsConf{}
	err = yaml.Unmarshal(yamlFile, argsConf)
	if err != nil {
		log.Fatalf("loadArgumentsConf(): Unmarshal YAML error: %v", err)
	}

	return argsConf
}

func init() {

	// logging options
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	/*
		Tweak os.Args, ignore anything leading up to first token starting w/ '-'
		   given:  ./gitops-argparser whatever some comment -arg1 x -arg2 y
		   yields: ./gitops-argparser -arg1 x -arg2 y
	*/
	sliceFrom := 0
	for i, x := range os.Args {
		if strings.HasPrefix(x, "-") {
			sliceFrom = i
			break
		}
	}
	os.Args = append(os.Args[:1], os.Args[sliceFrom:]...)

	// load our arguments conf file
	argsConf := loadArgumentsConf()

	// Create our flags from the ArgumentDef's contained in ArgumentsConf
	for _, argDef := range argsConf.Arguments {

		if argDef.DataType == "string" {
			flag.String(argDef.Long, fmt.Sprintf("%v", argDef.DefaultValue), argDef.Help)
		} else if argDef.DataType == "int" {
			val, err := strconv.Atoi(fmt.Sprintf("%v", argDef.DefaultValue))
			if err != nil {
				flag.PrintDefaults()
				log.Fatalf("init(): failed to process argumentDef[%v].dataType.defaultValue %v", argDef.Long, err)
			}
			flag.Int(argDef.Long, val, argDef.Help)
		} else if argDef.DataType == "bool" {
			val, err := strconv.ParseBool(fmt.Sprintf("%v", argDef.DefaultValue))
			if err != nil {
				flag.PrintDefaults()
				log.Fatalf("init(): failed to process argumentDef[%v].dataType.defaultValue %v", argDef.Long, err)
			}
			flag.Bool(argDef.Long, val, argDef.Help)
		}
	}

}

func main() {

	// parse flags
	flag.Parse()

	// load the template
	tmpl := loadOutputTemplateFile()
	var allArgs []ArgumentValue
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		allArgs = append(allArgs, ArgumentValue{Name: f.Name, Value: fmt.Sprintf("%v", f.Value)})
	})

	err := tmpl.Execute(os.Stdout, struct{ Arguments []ArgumentValue }{Arguments: allArgs})
	if err != nil {
		log.Fatalf("main(): failed to execute template: %v", err)
	}

}
