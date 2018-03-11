package tftp

import "log"
import "io"

var (
    Request *log.Logger
    Info *log.Logger
    Error *log.Logger
)

func StartLogger(request io.Writer, info io.Writer, error io.Writer) {
    Request = log.New(request, "REQUEST: ", log.Ldate|log.Ltime|log.Lshortfile)
    Info = log.New(info, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
    Error = log.New(error, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

