In-memory TFTP Server
=====================

This is a simple in-memory TFTP server, implemented in Go.  
It's not complete.  I spent 11 hours working on it, and I want to 
present what I have completed within the estimated time allocation.

It works to send and receive files.  There's a bug
somewhere, if the mode is not octet, sometimes it will save a 0 byte
file to disk using the linux tftp binary.  I'm not exactly sure why this 
is happening, as I certainly don't send an initial data packet if the 
file doesn't exist on the server.  My brain is a little fried after an
11-hour coding session and I'm bugging it here as I don't have the mental
fortitude or time to dig into it right now =p

I believe the implementation to be mostly thread-safe.  I read / save the file to 
a buffer keyed off the UDP address of the transaction, so multiple sessions
can't stomp on each other.  That said, it does not actually run asynchronously, 
meaning there are likely semaphores to be implemented at I/O time.  I simply did not
have time to go down this rabbit hole.

Given more time, I would fix that bug, write unit tests, implement
real concurrency, and <write a Makefile/learn go build tools>.

I used the following website as a reference, to get the initial sockets going, 
thus the server code will look shockingly similar:
https://varshneyabhi.wordpress.com/2014/12/23/simple-udp-clientserver-in-golang/

As well as the usual googling and punching of monitor that goes along with
implementing something in a new language.  I would say about 1/3 of my time
was spent not understanding things about go, 5/9 implementing the actual
server, and 1/9 saying bad words to vim.

I did not look at any docs regarding TFTP implementations, in any language,
except for the single reference doc RFC1350.

I wish I had a more complete thing to share, but I really enjoyed this exercise anyway.
It's been a very long time since I've had to think about these types of things, and
dipping my toes in go was a (mostly) great experience.

Usage
-----
Install to GOPATH.  Start server as such:

import "igneous.io/tftp"

func main() {
	tftp.StartServer()
}


Testing
-------
None, ship to prod!
