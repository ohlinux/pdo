package cmd

import (
	"fmt"
	"os"
	"io/ioutil"
	"bytes"
	"path/filepath"
	"time"
	"strings"

	"github.com/ohlinux/pdo/pkg/pdo"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)



var cfgFile, projectBase, userLicense,home string
var newpdo pdo.Pdo


var RootCmd = &cobra.Command{
	Use:     "pdo ",
	Short:   "parallel ssh tool",
	Long:    `   PDO is a simple tool that parallel do somthing using ssh , no agent.`,
	Args: cobra.MinimumNArgs(1),
	Example: `cat host.list | pdo -r 10 "pwd"`,
	Run: func(cmd *cobra.Command, args []string) {

		// command
			cmdArgs:=cmd.Flags().Args()
			cmdArgstr:=strings.Join(cmdArgs," ")
			newpdo.Command=pdo.Command{
				Inputcmd: cmdArgstr,
				Display: cmdArgstr,
				Execmd: cmdArgs[0],
				Args: cmdArgs[1:],
			}
			if strings.Contains(cmdArgstr, "{{.}}") ||
				strings.Contains(cmdArgstr, "{{.Host}}") ||
					strings.Contains(cmdArgstr, "{{.User}}") ||
						strings.Contains(cmdArgstr, "{{.Port}}") ||
							strings.Contains(cmdArgstr, "{{.Passwd}}") ||
								strings.Contains(cmdArgstr, "{{.Path}}") {
								newpdo.Command.Local = true
			}

			newpdo.PrepareInput(cmd ,args)

		    newpdo.PrepareOutput(cmd ,args)

			//todo: validate host list
			//todo: filter host list

			if err:=newpdo.CreateJobList();err!=nil {
				fmt.Errorf("input host list fail %v",err)
			}
			newpdo.Run()
	},
}

func init() {

	cobra.OnInitialize(initConfig)

	////全局变量
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config","c" ,home+"/.pdo/pdo.yml", "config file")

	// input
	RootCmd.PersistentFlags().StringP("in-file","f","","input from file")
	RootCmd.PersistentFlags().BoolP("in-last-fail","R",false,"input from last failure list")
	RootCmd.PersistentFlags().BoolP("in-last-success","S",false,"input from last success list")
	RootCmd.PersistentFlags().StringP("in-format","F","row","input format eg:json yaml ")
	RootCmd.PersistentFlags().StringP("in-regex","E","","input host regex to filter")

	// output
	//RootCmd.PersistentFlags().String("show","text","output format eg: json yaml row ")
	RootCmd.PersistentFlags().String("out","text","output format eg: json yaml row ")
	RootCmd.PersistentFlags().StringP("out-directory","o","","output save in directory")
	RootCmd.PersistentFlags().String("out-file","","output save to new file")
	RootCmd.PersistentFlags().String("out-file-append","","output append to a file")
	RootCmd.PersistentFlags().String("out-regex","","output highlight by regular")
	RootCmd.PersistentFlags().Bool("summary",true,"output summary information")
	RootCmd.PersistentFlags().BoolP("quiet","q",false,"output quiet mode , no summary and header information")
	RootCmd.PersistentFlags().Bool("header",true,"output header information ")
	RootCmd.PersistentFlags().Bool("out-nocolor",false,"output no color")

	// parallel
	RootCmd.PersistentFlags().IntP("concurrent","r",1,"thread control , concurrent processing")
	RootCmd.PersistentFlags().DurationP("time-over","t",time.Duration(time.Minute * 5),"thread control,over the time ,kill process.")
	RootCmd.PersistentFlags().DurationP("time-inter","T",time.Duration(time.Second * 0),"thread control,interval time between concurrent jobs")
	RootCmd.PersistentFlags().BoolP("yes","y",false,"thread control,input yes when ask")
	RootCmd.PersistentFlags().Int("ask",0,"thread control,ask every numbers,default is 0 means no ask expect the first one ")

	// Auth
	RootCmd.PersistentFlags().String("auth-priv",home+"/.ssh/id_rsb","authentication ,specified private key file")
	RootCmd.PersistentFlags().StringP("auth-user","u","","authentication ,specified user name")
	RootCmd.PersistentFlags().StringP("auth-passwd","p","","authentication ,specified password")
	RootCmd.PersistentFlags().String("auth-knownHosts",home+"/.ssh/known_hosts","authentication ,specified known_hosts file")

	// Log
	RootCmd.PersistentFlags().String("log-path","./logs","log ,log output directory")
	RootCmd.PersistentFlags().Int("log-backup",7 ,"log ,log backup number files")
	RootCmd.PersistentFlags().Int("log-maxfile",10000,"log , max value for each log file")
	RootCmd.PersistentFlags().String("log-level","info","log , output log level")

	// 使用viper可以绑定flag
	// log
	viper.BindPFlag("log.path", RootCmd.PersistentFlags().Lookup("log-path"))
	viper.BindPFlag("log.backup", RootCmd.PersistentFlags().Lookup("log-backup"))
	viper.BindPFlag("log.maxfile", RootCmd.PersistentFlags().Lookup("log-maxfile"))
	viper.BindPFlag("log.level", RootCmd.PersistentFlags().Lookup("log-level"))

	// Auth
	viper.BindPFlag("auth.user", RootCmd.PersistentFlags().Lookup("auth-user"))
	viper.BindPFlag("auth.privateKey", RootCmd.PersistentFlags().Lookup("auth-priv"))
	viper.BindPFlag("auth.passwd", RootCmd.PersistentFlags().Lookup("auth-passwd"))
	viper.BindPFlag("auth.knowHosts", RootCmd.PersistentFlags().Lookup("auth-knownHosts"))

	// parallel
	viper.BindPFlag("parallel.numbers", RootCmd.PersistentFlags().Lookup("concurrent"))
	viper.BindPFlag("parallel.overTime", RootCmd.PersistentFlags().Lookup("time-over"))
	viper.BindPFlag("parallel.intervalTime", RootCmd.PersistentFlags().Lookup("time-inter"))
	viper.BindPFlag("parallel.yes", RootCmd.PersistentFlags().Lookup("yes"))
	viper.BindPFlag("parallel.ask", RootCmd.PersistentFlags().Lookup("ask"))


	// output
	viper.BindPFlag("output.format", RootCmd.PersistentFlags().Lookup("show"))
	viper.BindPFlag("output.format", RootCmd.PersistentFlags().Lookup("out"))
	viper.BindPFlag("output.nocolor", RootCmd.PersistentFlags().Lookup("out-nocolor"))
	viper.BindPFlag("output.summary", RootCmd.PersistentFlags().Lookup("summary"))
	viper.BindPFlag("output.header", RootCmd.PersistentFlags().Lookup("header"))

	// input
	viper.BindPFlag("input.format", RootCmd.PersistentFlags().Lookup("in-format"))

	//viper.BindPFlag("useViper", RootCmd.PersistentFlags().Lookup("viper"))
	//viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
	//viper.SetDefault("license", "apache")


	// 局部变量
	//RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")



	// 自定义Usage
	RootCmd.SetUsageTemplate(temp)

}

func Execute()  {
	RootCmd.Execute()
}

func initConfig() {

	var body []byte
	var err error

	if cfgFile != "" {
		ext:=filepath.Ext(cfgFile)
		switch ext {
		case "json":
			fallthrough
		case "js":
			viper.SetConfigType("json")
		case "toml":
			fallthrough
		case "tml":
			viper.SetConfigType("toml")
		default:
			viper.SetConfigType("yaml")
		}

		body, err = ioutil.ReadFile(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	} else {
		// 找到home文件
		home, err = homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// 在home文件夹中搜索以“.cobra”为名称的config
		viper.AddConfigPath(home)
		viper.SetConfigName("pdo")
	}
	// 读取符合的环境变量
	viper.AutomaticEnv()


	if err := viper.ReadConfig(bytes.NewBuffer(body)); err != nil {
		fmt.Println("Can not read config:", viper.ConfigFileUsed(),err)
		os.Exit(1)
	}

	//config to struct
	var c pdo.Pdo
	viper.Unmarshal(&c)
	newpdo=pdo.NewPdo(c)
}


var temp=`Usage:{{if .Runnable}}

  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}

  {{.CommandPath}} <input control> [thread control] [output control] [subcommand] <function> [Argument]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
