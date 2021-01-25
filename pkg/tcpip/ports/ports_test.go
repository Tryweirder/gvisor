// Copyright 2018 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ports

import (
	"math/rand"
	"testing"

	"gvisor.dev/gvisor/pkg/tcpip"
)

const (
	fakeTransNumber   tcpip.TransportProtocolNumber = 1
	fakeNetworkNumber tcpip.NetworkProtocolNumber   = 2

	fakeIPAddress  = tcpip.Address("\x08\x08\x08\x08")
	fakeIPAddress1 = tcpip.Address("\x08\x08\x08\x09")
)

type portReserveTestAction struct {
	port    uint16
	ip      tcpip.Address
	want    func(tcpip.Error) tcpip.Error
	flags   Flags
	release bool
	device  tcpip.NICID
	dest    tcpip.FullAddress
}

func TestPortReservation(t *testing.T) {
	for _, test := range []struct {
		tname   string
		actions []portReserveTestAction
	}{
		{
			tname: "bind to ip",
			actions: []portReserveTestAction{
				{port: 80, ip: fakeIPAddress, want: nil},
				{port: 80, ip: fakeIPAddress1, want: nil},
				/* N.B. Order of tests matters! */
				{port: 80, ip: anyIPAddress, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 80, ip: fakeIPAddress, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}, flags: Flags{LoadBalanced: true}},
			},
		},
		{
			tname: "bind to inaddr any",
			actions: []portReserveTestAction{
				{port: 22, ip: anyIPAddress, want: nil},
				{port: 22, ip: fakeIPAddress, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				/* release fakeIPAddress, but anyIPAddress is still inuse */
				{port: 22, ip: fakeIPAddress, release: true},
				{port: 22, ip: fakeIPAddress, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 22, ip: fakeIPAddress, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}, flags: Flags{LoadBalanced: true}},
				/* Release port 22 from any IP address, then try to reserve fake IP address on 22 */
				{port: 22, ip: anyIPAddress, want: nil, release: true},
				{port: 22, ip: fakeIPAddress, want: nil},
			},
		}, {
			tname: "bind to zero port",
			actions: []portReserveTestAction{
				{port: 00, ip: fakeIPAddress, want: nil},
				{port: 00, ip: fakeIPAddress, want: nil},
				{port: 00, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: nil},
			},
		}, {
			tname: "bind to ip with reuseport",
			actions: []portReserveTestAction{
				{port: 25, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 25, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: nil},

				{port: 25, ip: fakeIPAddress, flags: Flags{}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 25, ip: anyIPAddress, flags: Flags{}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},

				{port: 25, ip: anyIPAddress, flags: Flags{LoadBalanced: true}, want: nil},
			},
		}, {
			tname: "bind to inaddr any with reuseport",
			actions: []portReserveTestAction{
				{port: 24, ip: anyIPAddress, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: anyIPAddress, flags: Flags{LoadBalanced: true}, want: nil},

				{port: 24, ip: anyIPAddress, flags: Flags{}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, flags: Flags{}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},

				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, release: true, want: nil},

				{port: 24, ip: anyIPAddress, flags: Flags{LoadBalanced: true}, release: true},
				{port: 24, ip: anyIPAddress, flags: Flags{}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},

				{port: 24, ip: anyIPAddress, flags: Flags{LoadBalanced: true}, release: true},
				{port: 24, ip: anyIPAddress, flags: Flags{}, want: nil},
			},
		}, {
			tname: "bind twice with device fails",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 3, want: nil},
				{port: 24, ip: fakeIPAddress, device: 3, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind to device",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 1, want: nil},
				{port: 24, ip: fakeIPAddress, device: 2, want: nil},
			},
		}, {
			tname: "bind to device and then without device",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 123, want: nil},
				{port: 24, ip: fakeIPAddress, device: 0, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind without device",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, want: nil},
				{port: 24, ip: fakeIPAddress, device: 123, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{LoadBalanced: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind with device",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 123, want: nil},
				{port: 24, ip: fakeIPAddress, device: 123, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{LoadBalanced: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 0, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 0, flags: Flags{LoadBalanced: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 456, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 789, want: nil},
				{port: 24, ip: fakeIPAddress, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind with reuseport",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 123, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 0, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 0, flags: Flags{LoadBalanced: true}, want: nil},
			},
		}, {
			tname: "binding with reuseport and device",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 123, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 0, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 456, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 0, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 789, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 999, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "mixing reuseport and not reuseport by binding to device",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 456, want: nil},
				{port: 24, ip: fakeIPAddress, device: 789, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 999, want: nil},
			},
		}, {
			tname: "can't bind to 0 after mixing reuseport and not reuseport",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 456, want: nil},
				{port: 24, ip: fakeIPAddress, device: 0, flags: Flags{LoadBalanced: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind and release",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 0, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 345, flags: Flags{}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 789, flags: Flags{LoadBalanced: true}, want: nil},

				// Release the bind to device 0 and try again.
				{port: 24, ip: fakeIPAddress, device: 0, flags: Flags{LoadBalanced: true}, want: nil, release: true},
				{port: 24, ip: fakeIPAddress, device: 345, flags: Flags{}, want: nil},
			},
		}, {
			tname: "bind twice with reuseport once",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 0, flags: Flags{LoadBalanced: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "release an unreserved device",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 456, flags: Flags{}, want: nil},
				// The below don't exist.
				{port: 24, ip: fakeIPAddress, device: 345, flags: Flags{}, want: nil, release: true},
				{port: 9999, ip: fakeIPAddress, device: 123, flags: Flags{}, want: nil, release: true},
				// Release all.
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{}, want: nil, release: true},
				{port: 24, ip: fakeIPAddress, device: 456, flags: Flags{}, want: nil, release: true},
			},
		}, {
			tname: "bind with reuseaddr",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 123, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{MostRecent: true}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 0, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
				{port: 24, ip: fakeIPAddress, device: 0, flags: Flags{MostRecent: true}, want: nil},
			},
		}, {
			tname: "bind twice with reuseaddr once",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, device: 123, flags: Flags{}, want: nil},
				{port: 24, ip: fakeIPAddress, device: 0, flags: Flags{MostRecent: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind with reuseaddr and reuseport",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
			},
		}, {
			tname: "bind with reuseaddr and reuseport, and then reuseaddr",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind with reuseaddr and reuseport, and then reuseport",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind with reuseaddr and reuseport twice, and then reuseaddr",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true}, want: nil},
			},
		}, {
			tname: "bind with reuseaddr and reuseport twice, and then reuseport",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: nil},
			},
		}, {
			tname: "bind with reuseaddr, and then reuseaddr and reuseport",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind with reuseport, and then reuseaddr and reuseport",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true, LoadBalanced: true}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{MostRecent: true}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind tuple with reuseaddr, and then wildcard with reuseaddr",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{TupleOnly: true}, dest: tcpip.FullAddress{Addr: fakeIPAddress, Port: 24}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{TupleOnly: true}, dest: tcpip.FullAddress{}, want: nil},
			},
		}, {
			tname: "bind tuple with reuseaddr, and then wildcard",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{TupleOnly: true}, dest: tcpip.FullAddress{Addr: fakeIPAddress, Port: 24}, want: nil},
				{port: 24, ip: fakeIPAddress, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind wildcard with reuseaddr, and then tuple with reuseaddr",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{TupleOnly: true}, dest: tcpip.FullAddress{}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{TupleOnly: true}, dest: tcpip.FullAddress{Addr: fakeIPAddress, Port: 24}, want: nil},
			},
		}, {
			tname: "bind tuple with reuseaddr, and then wildcard",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{TupleOnly: true}, dest: tcpip.FullAddress{Addr: fakeIPAddress, Port: 24}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind two tuples with reuseaddr",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, flags: Flags{TupleOnly: true}, dest: tcpip.FullAddress{Addr: fakeIPAddress, Port: 24}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{TupleOnly: true}, dest: tcpip.FullAddress{Addr: fakeIPAddress, Port: 25}, want: nil},
			},
		}, {
			tname: "bind two tuples",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, dest: tcpip.FullAddress{Addr: fakeIPAddress, Port: 24}, want: nil},
				{port: 24, ip: fakeIPAddress, dest: tcpip.FullAddress{Addr: fakeIPAddress, Port: 25}, want: nil},
			},
		}, {
			tname: "bind wildcard, and then tuple with reuseaddr",
			actions: []portReserveTestAction{
				{port: 24, ip: fakeIPAddress, dest: tcpip.FullAddress{}, want: nil},
				{port: 24, ip: fakeIPAddress, flags: Flags{TupleOnly: true}, dest: tcpip.FullAddress{Addr: fakeIPAddress, Port: 24}, want: func(err tcpip.Error) tcpip.Error {
					if _, ok := err.(*tcpip.ErrPortInUse); ok {
						return nil
					}
					return &tcpip.ErrPortInUse{}
				}},
			},
		}, {
			tname: "bind wildcard twice with reuseaddr",
			actions: []portReserveTestAction{
				{port: 24, ip: anyIPAddress, flags: Flags{TupleOnly: true}, want: nil},
				{port: 24, ip: anyIPAddress, flags: Flags{TupleOnly: true}, want: nil},
			},
		},
	} {
		t.Run(test.tname, func(t *testing.T) {
			pm := NewPortManager()
			net := []tcpip.NetworkProtocolNumber{fakeNetworkNumber}

			for _, test := range test.actions {
				if test.release {
					pm.ReleasePort(net, fakeTransNumber, test.ip, test.port, test.flags, test.device, test.dest)
					continue
				}
				gotPort, err := pm.ReservePort(net, fakeTransNumber, test.ip, test.port, test.flags, test.device, test.dest, nil /* testPort */)
				if fn := test.want; fn != nil {
					if want := fn(err); want != nil {
						t.Fatalf("ReservePort(.., .., %s, %d, %+v, %d, %v) = %v, want %s", test.ip, test.port, test.flags, test.device, test.dest, err, want)
					}
				} else if err != nil {
					t.Fatalf("ReservePort(.., .., %s, %d, %+v, %d, %v) = %v, want nil", test.ip, test.port, test.flags, test.device, test.dest, err)
				}
				if test.port == 0 && (gotPort == 0 || gotPort < FirstEphemeral) {
					t.Fatalf("ReservePort(.., .., .., 0, ..) = %d, want port number >= %d to be picked", gotPort, FirstEphemeral)
				}
			}
		})
	}
}

func TestPickEphemeralPort(t *testing.T) {
	for _, test := range []struct {
		name     string
		f        func(port uint16) (bool, tcpip.Error)
		wantErr  func(tcpip.Error) tcpip.Error
		wantPort uint16
	}{
		{
			name: "no-port-available",
			f: func(port uint16) (bool, tcpip.Error) {
				return false, nil
			},
			wantErr: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrNoPortAvailable); ok {
					return nil
				}
				return &tcpip.ErrNoPortAvailable{}
			},
		},
		{
			name: "port-tester-error",
			f: func(port uint16) (bool, tcpip.Error) {
				return false, &tcpip.ErrBadBuffer{}
			},
			wantErr: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrBadBuffer); ok {
					return nil
				}
				return &tcpip.ErrBadBuffer{}
			},
		},
		{
			name: "only-port-16042-available",
			f: func(port uint16) (bool, tcpip.Error) {
				if port == FirstEphemeral+42 {
					return true, nil
				}
				return false, nil
			},
			wantPort: FirstEphemeral + 42,
		},
		{
			name: "only-port-under-16000-available",
			f: func(port uint16) (bool, tcpip.Error) {
				if port < FirstEphemeral {
					return true, nil
				}
				return false, nil
			},
			wantErr: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrNoPortAvailable); ok {
					return nil
				}
				return &tcpip.ErrNoPortAvailable{}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			pm := NewPortManager()
			port, err := pm.PickEphemeralPort(test.f)
			if fn := test.wantErr; fn != nil {
				if want := fn(err); want != nil {
					t.Fatalf("got PickEphemeralPort(..) = (_, %v); want (_, %s)", err, want)
				}
			} else if err != nil {
				t.Fatalf("got PickEphemeralPort(..) = (_, %v); want (_, nil)", err)
			}
			if port != test.wantPort {
				t.Errorf("got PickEphemeralPort(..) = (%d, nil); want (%d, nil)", port, test.wantPort)
			}
		})
	}
}

func TestPickEphemeralPortStable(t *testing.T) {
	for _, test := range []struct {
		name     string
		f        func(port uint16) (bool, tcpip.Error)
		wantErr  func(tcpip.Error) tcpip.Error
		wantPort uint16
	}{
		{
			name: "no-port-available",
			f: func(port uint16) (bool, tcpip.Error) {
				return false, nil
			},
			wantErr: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrNoPortAvailable); ok {
					return nil
				}
				return &tcpip.ErrNoPortAvailable{}
			},
		},
		{
			name: "port-tester-error",
			f: func(port uint16) (bool, tcpip.Error) {
				return false, &tcpip.ErrBadBuffer{}
			},
			wantErr: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrBadBuffer); ok {
					return nil
				}
				return &tcpip.ErrBadBuffer{}
			},
		},
		{
			name: "only-port-16042-available",
			f: func(port uint16) (bool, tcpip.Error) {
				if port == FirstEphemeral+42 {
					return true, nil
				}
				return false, nil
			},
			wantPort: FirstEphemeral + 42,
		},
		{
			name: "only-port-under-16000-available",
			f: func(port uint16) (bool, tcpip.Error) {
				if port < FirstEphemeral {
					return true, nil
				}
				return false, nil
			},
			wantErr: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrNoPortAvailable); ok {
					return nil
				}
				return &tcpip.ErrNoPortAvailable{}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			pm := NewPortManager()
			portOffset := uint32(rand.Int31n(int32(numEphemeralPorts)))
			port, err := pm.PickEphemeralPortStable(portOffset, test.f)
			if fn := test.wantErr; fn != nil {
				if want := fn(err); want != nil {
					t.Fatalf("got PickEphemeralPort(..) = (_, %v); want (_, %s)", err, want)
				}
			} else if err != nil {
				t.Fatalf("got PickEphemeralPort(..) = (_, %v); want (_, nil)", err)
			}
			if port != test.wantPort {
				t.Errorf("got PickEphemeralPort(..) = (%d, nil); want (%d, nil)", port, test.wantPort)
			}
		})
	}
}
