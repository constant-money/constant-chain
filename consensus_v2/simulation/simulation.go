package main

import (
	"fmt"
	"os"
)

type Simulation struct {
	nodeList      []*Node
	fd            os.File
	maxTimeSlot   uint64
	startTimeSlot uint64
	scenario      Scenario
	expected      Expected
}

type Scenario struct {
	proposeComm map[uint64][]int // timeSlot -> sender -> receiver
	voteComm    map[uint64]map[string][]int
	sync        map[string][]string
}

type Expected struct {
}

var simulation *Simulation

func InitSimulation(scenario []map[string]Scenario, expected []map[string]Expected) *Simulation {
	return simulation
}

func GetSimulation() *Simulation {
	if simulation == nil {
		simulation = new(Simulation)
		simulation.scenario = Scenario{
			proposeComm: make(map[uint64][]int),
			voteComm:    make(map[uint64]map[string][]int),
			sync:        make(map[string][]string),
		}
	}
	return simulation
}

func (s *Simulation) setMaxTimeSlot(max uint64) {
	fmt.Println("Set Max Time Slot", max)
	s.maxTimeSlot = max
}
func (s *Simulation) setStartTimeSlot(start uint64) {
	fmt.Println("Set Start Time Slot", start)
	s.startTimeSlot = start
}

func (s *Simulation) setLogfile(fd os.File) {
	s.fd = fd

}
