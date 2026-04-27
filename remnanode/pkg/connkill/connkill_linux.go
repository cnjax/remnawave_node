//go:build linux

package connkill

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
)

// Netlink constants not in syscall package
const (
	netlinkInetDiag  = 4  // NETLINK_INET_DIAG
	sockDiagByFamily = 20 // SOCK_DIAG_BY_FAMILY
	sockDestroy      = 21 // SOCK_DESTROY
	tcpAllStates     = 0xffffffff
	inetDiagNoCookie = 0xffffffff
	nlHdrSize        = 16 // sizeof(nlmsghdr)
	reqSize          = 56 // sizeof(inet_diag_req_v2)
	// offsets within inet_diag_req_v2: [family(1) proto(1) ext(1) pad(1) states(4) id(48)]
	reqIDOff = 8
	// offsets within inet_diag_sockid: sport(2) dport(2) src(16) dst(16) if(4) cookie(8)
	idDstOff     = 20
	idCookieOff  = 40
	idSize       = 48
	// inet_diag_msg layout: family(1) state(1) timer(1) retrans(1) id(48) ...
	msgIDOff = 4
	msgMinLen = msgIDOff + idSize
)

// DropByIPs RSTs all established TCP connections whose remote peer matches any IP in the list.
func DropByIPs(ips []string) error {
	if len(ips) == 0 {
		return nil
	}

	targets := make(map[string]bool, len(ips))
	for _, s := range ips {
		if p := net.ParseIP(s); p != nil {
			// Normalise: IPv4-mapped IPv6 → plain IPv4
			if v4 := p.To4(); v4 != nil {
				targets[v4.String()] = true
			} else {
				targets[p.String()] = true
			}
		}
	}
	if len(targets) == 0 {
		return nil
	}

	sock, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC, netlinkInetDiag)
	if err != nil {
		return fmt.Errorf("netlink socket: %w", err)
	}
	defer syscall.Close(sock)

	if err := syscall.Bind(sock, &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK}); err != nil {
		return fmt.Errorf("netlink bind: %w", err)
	}

	// Set 5-second receive timeout so we don't block forever
	tv := syscall.Timeval{Sec: 5}
	_ = syscall.SetsockoptTimeval(sock, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

	for _, family := range []uint8{syscall.AF_INET, syscall.AF_INET6} {
		msgs, err := querySockets(sock, family)
		if err != nil {
			continue
		}
		for _, msg := range msgs {
			remoteIP := parseRemoteIP(msg, family)
			if remoteIP != nil && targets[remoteIP.String()] {
				_ = sendDestroy(sock, family, msg)
			}
		}
	}
	return nil
}

// querySockets sends SOCK_DIAG_BY_FAMILY and collects inet_diag_msg payloads.
func querySockets(sock int, family uint8) ([][]byte, error) {
	req := buildQueryMsg(family, 1)
	if err := syscall.Sendto(sock, req, 0, &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK}); err != nil {
		return nil, fmt.Errorf("sendto: %w", err)
	}

	var results [][]byte
	buf := make([]byte, 65536)

	for {
		n, _, err := syscall.Recvfrom(sock, buf, 0)
		if err != nil {
			break
		}

		data := buf[:n]
		done := false

		for len(data) >= nlHdrSize {
			msgLen := binary.LittleEndian.Uint32(data[0:4])
			msgType := binary.LittleEndian.Uint16(data[4:6])

			if msgLen < nlHdrSize || int(msgLen) > len(data) {
				break
			}

			switch msgType {
			case syscall.NLMSG_DONE:
				done = true
			case syscall.NLMSG_ERROR:
				done = true
			case sockDiagByFamily:
				payload := data[nlHdrSize:msgLen]
				if len(payload) >= msgMinLen {
					cp := make([]byte, len(payload))
					copy(cp, payload)
					results = append(results, cp)
				}
			}

			// Advance past this message (aligned to 4 bytes)
			aligned := (msgLen + 3) &^ 3
			if int(aligned) >= len(data) {
				data = data[len(data):]
			} else {
				data = data[aligned:]
			}
		}

		if done {
			break
		}
	}
	return results, nil
}

// buildQueryMsg constructs a SOCK_DIAG_BY_FAMILY netlink message that matches all TCP sockets.
func buildQueryMsg(family uint8, seq uint32) []byte {
	req := make([]byte, reqSize)
	req[0] = family
	req[1] = syscall.IPPROTO_TCP
	// ext=0, pad=0
	binary.LittleEndian.PutUint32(req[4:], tcpAllStates)
	// socket id wildcard: sport=0 dport=0 src=0 dst=0 if=0; cookie=NOCOOKIE
	binary.LittleEndian.PutUint32(req[reqIDOff+idCookieOff:], inetDiagNoCookie)
	binary.LittleEndian.PutUint32(req[reqIDOff+idCookieOff+4:], inetDiagNoCookie)

	hdr := make([]byte, nlHdrSize)
	binary.LittleEndian.PutUint32(hdr[0:], uint32(nlHdrSize+reqSize))
	binary.LittleEndian.PutUint16(hdr[4:], sockDiagByFamily)
	binary.LittleEndian.PutUint16(hdr[6:], syscall.NLM_F_REQUEST|syscall.NLM_F_DUMP)
	binary.LittleEndian.PutUint32(hdr[8:], seq)

	return append(hdr, req...)
}

// parseRemoteIP extracts the remote (dst) IP from an inet_diag_msg payload.
func parseRemoteIP(msg []byte, family uint8) net.IP {
	dstStart := msgIDOff + idDstOff
	if family == syscall.AF_INET {
		if len(msg) < dstStart+4 {
			return nil
		}
		b := make([]byte, 4)
		copy(b, msg[dstStart:dstStart+4])
		return net.IP(b)
	}
	if len(msg) < dstStart+16 {
		return nil
	}
	b := make([]byte, 16)
	copy(b, msg[dstStart:dstStart+16])
	return net.IP(b)
}

// sendDestroy sends a SOCK_DESTROY netlink message for the socket described by diagMsg.
func sendDestroy(sock int, family uint8, diagMsg []byte) error {
	req := make([]byte, reqSize)
	req[0] = family
	req[1] = syscall.IPPROTO_TCP
	binary.LittleEndian.PutUint32(req[4:], tcpAllStates)
	// Copy the exact 48-byte socket ID from the diagnostic message
	copy(req[reqIDOff:reqIDOff+idSize], diagMsg[msgIDOff:msgIDOff+idSize])

	hdr := make([]byte, nlHdrSize)
	binary.LittleEndian.PutUint32(hdr[0:], uint32(nlHdrSize+reqSize))
	binary.LittleEndian.PutUint16(hdr[4:], sockDestroy)
	binary.LittleEndian.PutUint16(hdr[6:], syscall.NLM_F_REQUEST)
	binary.LittleEndian.PutUint32(hdr[8:], 2)

	return syscall.Sendto(sock, append(hdr, req...), 0,
		&syscall.SockaddrNetlink{Family: syscall.AF_NETLINK})
}
