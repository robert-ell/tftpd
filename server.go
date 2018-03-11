package tftp

import (
    "fmt"
    "net"
    "os"
//	"reflect"
)

/* A Simple function to verify error */
func CheckError(err error) {
    if err  != nil {
        fmt.Println("Error: " , err)
        os.Exit(0)
    }
}

func StartServer() {
	f, err := os.Create("/tmp/requests.log")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	StartLogger(f, os.Stdout, os.Stderr)
    ServerAddr,err := net.ResolveUDPAddr("udp",":10069")
    CheckError(err)

    ServerConn, err := net.ListenUDP("udp", ServerAddr)
    CheckError(err)
    defer ServerConn.Close()


    for {
		buf := make([]byte, MaxPacketSize)
        _,addr,_ := ServerConn.ReadFromUDP(buf)

		// Parse the packet, declaring the struct implementation, setting fields appropriately
		packet, err := ParsePacket(buf)
		// Get the response from the handling of the packet
		response := packet.Handle(addr)

		ServerConn.WriteToUDP(response, addr)

        if err != nil {
            fmt.Println("Error: ",err)
        }
    }
}
