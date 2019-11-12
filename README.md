# gitops-argparser

This project provides a simple utility that can be used in gitops driven CI/CD processes which support setting variables via echoing and/or writing specific syntaxes to STDOUT. The utility was created to support the idea of defining and enforcing custom CI/CD *arguments* that developers can decorate within git commit messages which would then be used to alter default CI/CD behavior. 

`gitops-argparser` permits you to define your available arguments, expected types, default values etc in a YAML file, and then be invoked with any set of those arguments, and it will emit custom output using a golang template with [Sprig functions support](https://github.com/Masterminds/sprig).

## Azure DevOps example

Azure DevOps pipelines supports [Logging Commands](https://github.com/microsoft/azure-pipelines-tasks/blob/master/docs/authoring/commands.md) which detect various commands in STDOUT and then take further action based on those commands in a pipeline; one of which is `##vso[task.setvariable variable=varname]varvalue`

### Lets define our supported commit message arguments

Create a file called `config.yaml`

```
arguments:
  - long: arg1
    dataType: string
    help: This is argument number one
    defaultValue: arg1default
  - long: arg2
    dataType: string
    help: This is argument number two
    defaultValue: "hi"
  - long: arg3
    dataType: int
    help: This is argument number three
    defaultValue: 2
  - long: arg4
    dataType: bool
    help: This is argument number four
    defaultValue: true
```

### Lets define how we will handle the args in a template

Create a file called `output.tmpl`. This is a [golang text/template](https://golang.org/pkg/text/template/) with [Sprig functions support](https://github.com/Masterminds/sprig)
```
{{ range $arg := .Arguments }}
##vso[task.setvariable variable={{$arg.Name}}]{{$arg.Value}}
{{ end }}
```

### Lets run it manually to simulate a commit message being processed

```
> ./gitops-argparser some raw commit message value -arg1 arg1value -arg2 arg2val -arg3 9999 -arg4=true

{"level":"debug","msg":"loadArgumentsConf(): reading argparser arguments conf from: config.yaml","time":"2019-11-12T13:59:09-05:00"}
{"level":"debug","msg":"loadOutputTemplateFile(): reading argparser output template from: output.tmpl","time":"2019-11-12T13:59:09-05:00"}

##vso[task.setvariable variable=arg1]arg1value

##vso[task.setvariable variable=arg2]arg2val

##vso[task.setvariable variable=arg3]9999

##vso[task.setvariable variable=arg4]true
```

We can see it validates the arguments and converts them to Azure *log commands* which will set variables in a pipeline.

If you call it with nothing you get the defaults:
```
> ./gitops-argparser 

{"level":"debug","msg":"loadArgumentsConf(): reading argparser arguments conf from: config.yaml","time":"2019-11-12T13:59:20-05:00"}
{"level":"debug","msg":"loadOutputTemplateFile(): reading argparser output template from: output.tmpl","time":"2019-11-12T13:59:20-05:00"}

##vso[task.setvariable variable=arg1]arg1default

##vso[task.setvariable variable=arg2]hi

##vso[task.setvariable variable=arg3]2

##vso[task.setvariable variable=arg4]true
```

### Hook it up in an Azure pipeline task

Process a normal a commit message:
```
...

- task: Bash@3
    displayName: Parse commit message args
    targetType: 'Inline'
    script: ./gitops-argparser $(Build.SourceVersionMessage)

- task: Bash@3
    displayName: Print pipeline vars from commit message args
    targetType: 'Inline'
    script: | 
        echo $(arg1)
        echo $(arg2)
        echo $(arg3)
        echo $(arg4)
```

Process an annotated tag message:
```
...

- task: Bash@3
    displayName: Parse commit message args
    targetType: 'Inline'
    script: |
      export GIT_TAG_MSG=`git tag -l --format='%(contents:lines=1)' $(Build.SourceBranchName)`
      echo "GIT_TAG_MSG=$GIT_TAG_MSG"
      ./gitops-argparser $GIT_TAG_MSG

- task: Bash@3
    displayName: Print pipeline vars from commit message args
    targetType: 'Inline'
    script: | 
        echo $(arg1)
        echo $(arg2)
        echo $(arg3)
        echo $(arg4)
```

## Important behavior notes

Parsing starts at the first token after the command that begins with a dash/hypen `-` character
```
# OK: gets all args
./gitops-argparser whatever text -arg1 x -arg2 y

# OK: gets all args
./gitops-argparser whatever text -arg1 x -arg2 y some trailing text

# NOT OK: stops parsing on first NON argument (only captures -arg1)
./gitops-argparser whatever text -arg1 x other-non-quoted-text -arg2 y some trailing text
```
