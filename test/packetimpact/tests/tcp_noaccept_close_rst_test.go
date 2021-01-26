// Copyright 2020 The gVisor Authors.
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

package tcp_noaccept_close_rst_test

import (
	"flag"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/test/packetimpact/testbench"
)

func init() {
	testbench.Initialize(flag.CommandLine)
}

func waitUntilEstablished(t *testing.T, dut *testbench.DUT, listenFd int32, timeout time.Duration) {
	t.Helper()
	pfds := dut.Poll(t, []unix.PollFd{{Fd: listenFd, Events: unix.POLLIN}}, timeout)
	if len(pfds) != 1 || pfds[0].Fd != listenFd || (pfds[0].Revents&unix.POLLIN == 0) {
		t.Fatalf("the connection is still not established after %s", timeout)
	}
}

func TestTcpNoAcceptCloseReset(t *testing.T) {
	dut := testbench.NewDUT(t)
	listenFd, remotePort := dut.CreateListener(t, unix.SOCK_STREAM, unix.IPPROTO_TCP, 1)
	conn := dut.Net.NewTCPIPv4(t, testbench.TCP{DstPort: &remotePort}, testbench.TCP{SrcPort: &remotePort})
	conn.Connect(t)
	defer conn.Close(t)
	waitUntilEstablished(t, &dut, listenFd, time.Second)
	dut.Close(t, listenFd)
	if _, err := conn.Expect(t, testbench.TCP{Flags: testbench.Uint8(header.TCPFlagRst | header.TCPFlagAck)}, 1*time.Second); err != nil {
		t.Fatalf("expected a RST-ACK packet but got none: %s", err)
	}
}
