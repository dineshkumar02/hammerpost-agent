package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"dbcontrol.tessell/model"
	"dbcontrol.tessell/mysql"
	"dbcontrol.tessell/pg"
	"github.com/alexflint/go-arg"
	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

var args CliArgs

type CliArgs struct {
	Listen    string `arg:"--listen" description:"Listen address" default:":8989"`
	StopCmd   string `arg:"--stop-cmd,required" help:"Command to stop the database pg_ctlcluster 14 main stop"`
	StartCmd  string `arg:"--start-cmd,required" help:"Command to start the database pg_ctlcluster 14 main start"`
	PgDsn     string `arg:"--pgdsn" help:"PostgreSQL DSN" default:"postgres://postgres:postgres@localhost:5432/postgres"`
	MyCnfPath string `arg:"--my-cnf-path" help:"Path to mysqld.cnf file" default:"/etc/mysql/mysql.conf.d/mysqld.cnf"`
	DbType    string `arg:"--db-type,required" help:"Database type (mysql or postgres)"`
	IoStatCmd string `arg:"--iostat-cmd" help:"iostat command, do not give any $var/num shell evaluate commands" default:"iostat -dmx 1 2 nvme3n1|tail -n 2|tr -s ' '|cut -d ' ' -f2,3,4,5,16"`
}

func init() {
	arg.MustParse(&args)

	if args.DbType != "mysql" && args.DbType != "postgres" {
		panic("invalid db type")
	}
}

func start() error {
	var outb, errb bytes.Buffer

	// Start PostgreSQL
	genCmd := strings.Split(args.StartCmd, " ")

	cmd := exec.Command(genCmd[0], genCmd[1:]...)

	cmd.Stdout = &outb
	cmd.Stderr = &errb

	fmt.Println("starting database using command: ", args.StartCmd)

	err := cmd.Run()

	if errb.String() != "" {
		fmt.Printf("message while starting service: %s", outb.String()+"\n"+errb.String())
	}
	return err
}

func stop() error {
	var outb, errb bytes.Buffer

	genCmd := strings.Split(args.StopCmd, " ")

	cmd := exec.Command(genCmd[0], genCmd[1:]...)

	cmd.Stdout = &outb
	cmd.Stderr = &errb

	fmt.Println("stopping database using command: ", args.StopCmd)
	err := cmd.Run()

	// If db stop failes, then ignore the error by logging the message into the log file
	if errb.String() != "" {
		fmt.Printf("message while starting service: %s", outb.String()+"\n"+errb.String())
	}
	return err
}

func nodeInfo() (string, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return "", fmt.Errorf("error while getting host info: %v", err)
	}

	cpu, err := cpu.InfoWithContext(context.Background())
	if err != nil {
		return "", fmt.Errorf("error while getting cpu info: %v", err)
	}

	load, err := load.AvgWithContext(context.Background())
	if err != nil {
		return "", fmt.Errorf("error while getting load info: %v", err)
	}

	mem, err := mem.VirtualMemory()
	if err != nil {
		return "", fmt.Errorf("error while getting memory info: %v", err)
	}

	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "OS\t", hostInfo.OS)
	fmt.Fprintln(w, "Platform\t", hostInfo.Platform+"-"+hostInfo.PlatformVersion)
	fmt.Fprintln(w, "Kernel\t", hostInfo.KernelVersion)
	fmt.Fprintln(w, "Uptime\t", hostInfo.Uptime)
	fmt.Fprintln(w, "Total Processes\t", hostInfo.Procs)
	fmt.Fprintln(w, "Load Avg\t", load.Load1)

	fmt.Fprintln(w, "CPU\t", cpu[0].ModelName)
	fmt.Fprintln(w, "CPU Count\t", len(cpu))
	fmt.Fprintln(w, "CPU Cores\t", cpu[0].Cores)
	fmt.Fprintln(w, "CPU Mhz\t", cpu[0].Mhz)
	fmt.Fprintln(w, "Total Memory(GB)\t", mem.Total/1024/1024/1024)
	fmt.Fprintln(w, "Free Memory(GB)\t", mem.Free/1024/1024/1024)
	fmt.Fprintln(w, "Used Memory(GB)\t", mem.Used/1024/1024/1024)

	w.Flush()

	return buf.String(), nil
}

func getIOStat() (tps float64, read float64, write float64, readmbps float64, writembps float64, util float64, err error) {
	var outb, errb bytes.Buffer
	//
	//XXX
	//This has been teste with iostat below version,
	//which is working fine.
	// [postgres@ip-10-0-10-27 ~]$ iostat -V
	// sysstat version 11.7.3
	// (C) Sebastien Godard (sysstat <at> orange.fr)
	cmd := exec.Command("bash", "-c", args.IoStatCmd)

	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err = cmd.Run()

	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("error while getting iostat: %v", err)
	}

	if errb.String() != "" {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("error while getting iostat: %v", errb.String())
	}

	// Parse the output
	// Device:                reads/s    writes/s    Mb_read    Mb_write ... Util

	output := strings.TrimSpace(outb.String())

	// Split string based on spaces

	iops := strings.Split(output, " ")

	read, err = strconv.ParseFloat(iops[0], 64)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("error while parsing read iostat: %v", err)
	}
	write, err = strconv.ParseFloat(iops[1], 64)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("error while parsing write iostat: %v", err)
	}

	readmbps, err = strconv.ParseFloat(iops[2], 64)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("error while parsing readmb iostat: %v", err)
	}

	writembps, err = strconv.ParseFloat(iops[3], 64)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("error while parsing writemb iostat: %v", err)
	}

	util, err = strconv.ParseFloat(iops[4], 64)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("error while parsing util iostat: %v", err)
	}

	return (read + write), read, write, readmbps, writembps, util, nil
}

func getMetrics() (*model.Metric, error) {
	cpu, err := cpu.PercentWithContext(context.Background(), time.Duration(1*time.Second), false)
	if err != nil {
		return nil, fmt.Errorf("error while getting cpu metrics: %v", err)
	}

	mem, err := mem.VirtualMemoryWithContext(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error while getting memory metrics: %v", err)
	}

	// disk, err := disk.IOCountersWithContext(context.Background())
	// if err != nil {
	// 	return nil, fmt.Errorf("error while getting disk metrics: %v", err)
	// }

	load, err := getLoadAvg()
	if err != nil {
		return nil, fmt.Errorf("error while getting load metrics: %v", err)
	}

	// Run iostat command and parse the output
	tps, read, write, readmb, writemb, util, err := getIOStat()
	if err != nil {
		return nil, fmt.Errorf("error while getting iostat metrics: %v", err)
	}

	// var weightedIO uint64
	// var ioTime uint64
	// var iopsInProgress uint64

	// for _, v := range disk {
	// 	weightedIO += v.WeightedIO
	// 	ioTime += v.IoTime
	// 	iopsInProgress += v.IopsInProgress
	// }

	// buf := new(bytes.Buffer)
	// w := tabwriter.NewWriter(buf, 0, 0, 1, ' ', tabwriter.AlignRight)
	// fmt.Fprintln(w, "IO Wait(sec)\t", int(weightedIO/1000))
	// fmt.Fprintln(w, "Spend IO Time(sec)\t", int(ioTime/1000))
	// fmt.Fprintln(w, "IOPS in Progress\t", iopsInProgress)

	// w.Flush()

	return &model.Metric{
		CpuUsage:    cpu[0],
		MemoryUsage: (float64(mem.Total-mem.Free) / float64(mem.Total)) * 100,
		LoadAvg:     load,
		DiskTps:     tps,
		Reads:       read,
		Writes:      write,
		ReadMbps:    readmb,
		WriteMbps:   writemb,
		Util:        util,
	}, nil

}

func getLoadAvg() (float64, error) {
	load, err := load.AvgWithContext(context.Background())
	if err != nil {
		return 0, fmt.Errorf("error while getting load info: %v", err)
	}

	return load.Load1, nil
}

func main() {

	r := gin.Default()
	r.GET("/start", func(c *gin.Context) {
		err := start()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "OK",
			})
		}
	})

	r.GET("/stop", func(c *gin.Context) {
		err := stop()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "OK",
			})
		}
	})

	r.GET("/info", func(c *gin.Context) {
		info, err := nodeInfo()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})

		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": info,
			})
		}
	})

	r.GET("/metrics", func(c *gin.Context) {
		metrics, err := getMetrics()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
		} else {
			b, err := json.Marshal(*metrics)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message": string(b),
			})
		}
	})

	r.POST("/set-param", func(c *gin.Context) {
		var params []model.Param
		if err := c.ShouldBindJSON(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if args.DbType == "postgres" {
			err := pg.UpdatePgParameter(args.PgDsn, params)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": err.Error(),
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"message": "OK",
				})
			}
		} else if args.DbType == "mysql" {
			err := mysql.UpdateMysqlParameter(args.MyCnfPath, params)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": err.Error(),
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"message": "OK",
				})
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "not supported db type",
			})
		}
	})

	r.GET("/load", func(c *gin.Context) {
		load, err := getLoadAvg()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": fmt.Sprintf("%.2f", load),
			})
		}
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.Run(args.Listen)
}
