package main

import (
	"fmt"
	"github.com/urfave/cli"
	"github.com/zrepl/zrepl/jobrun"
	"github.com/zrepl/zrepl/rpc"
	"github.com/zrepl/zrepl/sshbytestream"
	"github.com/zrepl/zrepl/zfs"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

var conf Config
var runner *jobrun.JobRunner
var logFlags int = log.LUTC | log.Ldate | log.Ltime
var defaultLog Logger

func main() {

	defer func() {
		e := recover()
		if e != nil {
			defaultLog.Printf("panic:\n%s\n\n", debug.Stack())
			defaultLog.Printf("error: %t %s", e, e)
			os.Exit(1)
		}
	}()

	app := cli.NewApp()

	app.Name = "zrepl"
	app.Usage = "replicate zfs datasets"
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "config"},
	}
	app.Before = func(c *cli.Context) (err error) {

		defaultLog = log.New(os.Stderr, "", logFlags)

		if !c.GlobalIsSet("config") {
			return cli.NewExitError("config flag not set", 2)
		}
		if conf, err = ParseConfig(c.GlobalString("config")); err != nil {
			return cli.NewExitError(err, 2)
		}

		jobrunLogger := log.New(os.Stderr, "jobrun ", logFlags)
		runner = jobrun.NewJobRunner(jobrunLogger)
		return
	}
	app.Commands = []cli.Command{
		{
			Name:    "stdinserver",
			Aliases: []string{"s"},
			Usage:   "start in stdin server mode (from authorized keys)",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "identity"},
				cli.StringFlag{Name: "logfile"},
			},
			Action: cmdStdinServer,
		},
		{
			Name:    "run",
			Aliases: []string{"r"},
			Usage:   "do replication",
			Action:  cmdRun,
		},
	}

	app.Run(os.Args)

}

func cmdStdinServer(c *cli.Context) (err error) {

	if !c.IsSet("identity") {
		return cli.NewExitError("identity flag not set", 2)
	}
	identity := c.String("identity")

	var logOut io.Writer
	if c.IsSet("logfile") {
		var logFile *os.File
		logFile, err = os.OpenFile(c.String("logfile"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			return
		}

		if err = unix.Dup2(int(logFile.Fd()), int(os.Stderr.Fd())); err != nil {
			logFile.WriteString(fmt.Sprintf("error duping logfile to stderr: %s\n", err))
			return
		}
		logOut = logFile
	} else {
		logOut = os.Stderr
	}

	var sshByteStream io.ReadWriteCloser
	if sshByteStream, err = sshbytestream.Incoming(); err != nil {
		return
	}

	findMapping := func(cm []ClientMapping) zfs.DatasetMapping {
		for i := range cm {
			if cm[i].From == identity {
				return cm[i].Mapping
			}
		}
		return nil
	}

	sinkLogger := log.New(logOut, fmt.Sprintf("sink[%s] ", identity), logFlags)
	handler := Handler{
		Logger:      sinkLogger,
		PushMapping: findMapping(conf.Sinks),
		PullMapping: findMapping(conf.PullACLs),
	}

	if err = rpc.ListenByteStreamRPC(sshByteStream, handler, sinkLogger); err != nil {
		//os.Exit(1)
		err = cli.NewExitError(err, 1)
		defaultLog.Printf("listenbytestreamerror: %#v\n", err)
	}

	return

}

func cmdRun(c *cli.Context) error {

	// Do every pull, do every push
	// Scheduling

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runner.Start()
	}()

	for i := range conf.Pulls {
		pull := conf.Pulls[i]

		j := jobrun.Job{
			Name:     fmt.Sprintf("pull%d", i),
			Interval: time.Duration(5 * time.Second),
			Repeats:  true,
			RunFunc: func(log jobrun.Logger) error {
				log.Printf("doing pull: %v", pull)
				return jobPull(pull, c, log)
			},
		}

		runner.AddJob(j)
	}

	for i := range conf.Pushs {
		push := conf.Pushs[i]

		j := jobrun.Job{
			Name:     fmt.Sprintf("push%d", i),
			Interval: time.Duration(5 * time.Second),
			Repeats:  true,
			RunFunc: func(log jobrun.Logger) error {
				log.Printf("%v: %#v\n", time.Now(), push)
				return nil
			},
		}

		runner.AddJob(j)
	}

	for {
		select {
		case job := <-runner.NotificationChan():
			log.Printf("notificaiton on job %s: error=%v\n", job.Name, job.LastError)
		}
	}

	wg.Wait()

	return nil
}

func jobPull(pull Pull, c *cli.Context, log jobrun.Logger) (err error) {

	if lt, ok := pull.From.Transport.(LocalTransport); ok {
		lt.SetHandler(Handler{
			Logger:      log,
			PullMapping: pull.Mapping,
		})
		pull.From.Transport = lt
		log.Printf("fixing up local transport: %#v", pull.From.Transport)
	}

	var remote rpc.RPCRequester

	if remote, err = pull.From.Transport.Connect(); err != nil {
		return
	}

	defer closeRPCWithTimeout(log, remote, time.Second*10, "")

	return doPull(PullContext{remote, log, pull.Mapping, pull.InitialReplPolicy})
}
