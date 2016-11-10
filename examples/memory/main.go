package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/goburrow/serial"
	"github.com/xiegeo/modbusone"
)

var _ = time.Second

var address = flag.String("l", "", "required device location, such as: /dev/ttyS0 in linux or com1 in windows")
var baudRate = flag.Int("r", 19200, "baud rate")
var parity = flag.String("p", "E", "parity: N - None, E - Even, O - Odd")
var stopBits = flag.Int("s", 1, "stop bits: 1 or 2")

var isClient = flag.Bool("c", false, "true for client, false (default) for server. The client is interactive.")
var slaveID = flag.Uint64("id", 1, "the slaveId of the server for serial communication, 0 for multicast only")
var fillData = flag.String("d", "am3", "data to start with, am3 starts memory "+
	"with bools as address (mod 3) == 0, and registers as address * 3 (mod uint16)")

var writeSizeLimit = flag.Int("wsl", modbusone.MaxRTUSize, "client only, the max size in bytes of a write to server to send")
var readSizeLimit = flag.Int("rsl", modbusone.MaxRTUSize, "client only, the max size in bytes of a read from server to request")

var verbose = flag.Bool("v", false, "prints debugging information")

func main() {
	flag.Parse()
	if *verbose {
		modbusone.DebugOut = os.Stdout
	}
	var com = &modbusone.SerialPort{}
	err := com.Open(serial.Config{
		Address:  *address,
		BaudRate: *baudRate,
		Parity:   *parity,
		StopBits: *stopBits,

		Timeout: time.Hour, //a hack for https://github.com/goburrow/serial/issues/5
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open serial error: %v\n", err)
		os.Exit(1)
	}
	defer com.Close()

	id, err := modbusone.Uint64ToSlaveID(*slaveID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "set slaveID error: %v\n", err)
		os.Exit(1)
	}
	if *fillData == "am3" {
		fillAm3()
	}
	var device modbusone.Server
	if *isClient {
		if *writeSizeLimit > modbusone.MaxRTUSize || *readSizeLimit > modbusone.MaxRTUSize {
			fmt.Fprintf(os.Stderr, "write/read size limit is too big")
			os.Exit(1)
		}
		client := modbusone.NewRTUCLient(com, id)
		go runClient(client)
		device = client
	} else {
		device = modbusone.NewRTUServer(com, id)
	}
	h := modbusone.SimpleHandler{
		ReadDiscreteInputs: func(address, quantity uint16) ([]bool, error) {
			fmt.Printf("ReadDiscreteInputs from %v, quantity %v\n", address, quantity)
			return discretes[address : address+quantity], nil
		},
		WriteDiscreteInputs: func(address uint16, values []bool) error {
			fmt.Printf("WriteDiscreteInputs from %v, quantity %v\n", address, len(values))
			for i, v := range values {
				discretes[address+uint16(i)] = v
			}
			return nil
		},

		ReadCoils: func(address, quantity uint16) ([]bool, error) {
			fmt.Printf("ReadCoils from %v, quantity %v\n", address, quantity)
			return coils[address : address+quantity], nil
		},
		WriteCoils: func(address uint16, values []bool) error {
			fmt.Printf("WriteCoils from %v, quantity %v\n", address, len(values))
			for i, v := range values {
				coils[address+uint16(i)] = v
			}
			return nil
		},

		ReadInputRegisters: func(address, quantity uint16) ([]uint16, error) {
			fmt.Printf("ReadInputRegisters from %v, quantity %v\n", address, quantity)
			return inputRegisters[address : address+quantity], nil
		},
		WriteInputRegisters: func(address uint16, values []uint16) error {
			fmt.Printf("WriteInputRegisters from %v, quantity %v\n", address, len(values))
			for i, v := range values {
				inputRegisters[address+uint16(i)] = v
			}
			return nil
		},

		ReadHoldingRegisters: func(address, quantity uint16) ([]uint16, error) {
			fmt.Printf("ReadHoldingRegisters from %v, quantity %v\n", address, quantity)
			return holdingRegisters[address : address+quantity], nil
		},
		WriteHoldingRegisters: func(address uint16, values []uint16) error {
			fmt.Printf("WriteHoldingRegisters from %v, quantity %v\n", address, len(values))
			for i, v := range values {
				holdingRegisters[address+uint16(i)] = v
			}
			return nil
		},

		OnErrorImp: func(req modbusone.PDU, errRep modbusone.PDU) {
			fmt.Printf("error received: %v from req:\n", errRep, req)
		},
	}
	err = device.Serve(&h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
		os.Exit(1)
	}
}

const size = 0x10000

var discretes [size]bool
var coils [size]bool
var inputRegisters [size]uint16
var holdingRegisters [size]uint16

func fillAm3() {
	for i := range discretes {
		discretes[i] = i%3 == 0
	}
	for i := range coils {
		coils[i] = i%3 != 0
	}
	for i := range inputRegisters {
		inputRegisters[i] = uint16(i * 3)
	}
	for i := range holdingRegisters {
		holdingRegisters[i] = uint16(0xFFFF - i)
	}
}
