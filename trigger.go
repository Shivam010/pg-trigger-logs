package pgtl

import (
	"bytes"
	"encoding/json"
	"log"
	"time"

	"github.com/lib/pq"
)

// Response collects whole response from the notification log triggered
// in two formats: JSON and Map of string-interface
type Response struct {
	JSON []byte
	Map  map[string]interface{}
	Err  error
}

// GetChangesLogs gets the changes log
func GetChangesLogs(ls *pq.Listener) <-chan *Response {
	res := make(chan *Response)
	go func() (err error) {
		defer func() {
			if err != nil {
				res <- &Response{
					Err: err,
				}
			}
			close(res)
		}()
		for {
			select {
			case noti := <-ls.Notify:
				var msg bytes.Buffer
				err = json.Indent(&msg, []byte(noti.Extra), "", "\t")
				if err != nil {
					return
				}
				mp := map[string]interface{}{}
				err = json.Unmarshal(msg.Bytes(), &mp)
				if err != nil {
					return
				}
				res <- &Response{
					JSON: msg.Bytes(),
					Map:  mp,
				}
			case <-time.After(time.Minute):
				log.Println("No events since last 60 seconds, checking connection")
				go func() {
					err = ls.Ping()
					if err != nil {
						return
					}
				}()
			}
		}
	}()
	return res
}
