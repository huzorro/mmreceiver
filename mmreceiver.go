package main

import (
	"bytes"
	"database/sql"
	"encoding/xml"
	"fmt"
	"github.com/codegangsta/martini"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

//<?xml version="1.0" encoding="utf-8"?>
//<request>
//        <id>14092209191300001</id>
//        <command>sync_mo_req</command>
//        <operator>CM</operator>
//        <type>0</type>
//        <gateway>801057</gateway>
//        <msgid>14092209191300001</msgid>
//        <from>13910422228</from>
//        <to>10669501</to>
//        <serviceid>115103</serviceid>
//        <msgfmt>0</msgfmt>
//        <msg>ZEJmQ1lY</msg>
//        <linkid>55877283390400855976</linkid>
//        <spid>mms01</spid>
//        <t>20140922 09:19:12</t>
//</request>

//返回格式：
//<response>
//	<id></id>
//	<command>sync_mo_resp</command>
//	<result>0</result>
//</response>
type mmrequest struct {
	XMLName   xml.Name `xml:"request"`
	id        string   `xml:"id"`
	command   string   `xml:"command`
	operator  string   `xml:"operator`
	mtype     string   `xml:"type"`
	gateway   string   `xml:"gateway"`
	msgid     string   `xml:"msgid"`
	from      string   `xml:"from"`
	to        string   `xml:"to"`
	serviceid string   `xml:"serviceid"`
	msgfmt    string   `xml:"msgfmt"`
	msg       string   `xml:"msg"`
	linkid    string   `xml:"linkid"`
	spid      string   `xml:"spid"`
	t         string   `xml:"t"`
}

func mmReceiver(r *http.Request, w http.ResponseWriter, db *sql.DB, log *log.Logger) (int, string) {
	// Process message
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("mm receive message failed:", err)
		return http.StatusBadRequest, "request failed"
	} else {
		var msg mmrequest

		if err := xml.Unmarshal(data, &msg); err != nil {
			log.Println("mm parse message failed:", err)
			return http.StatusBadRequest, "request failed"
		} else {
			stmtIn, err := db.Prepare("INSERT INTO mms_forward(spcode, srctermid, desttermid, msgcontent, linkid) VALUES(?, ?, ?, ?, ?)")
			if err != nil {
				panic(err.Error())
			}
			defer stmtIn.Close()

			// _, err = stmtIn.Exec(spid, srctermid, linkid, citycode, cmd, desttermid, fee, serviceid, time)
			res, err := stmtIn.Exec(msg.gateway, msg.from, msg.to, msg.msg, msg.linkid)

			if err != nil {
				panic(err.Error())
			}
			rowId, err := res.LastInsertId()
			if err != nil {
				panic(err.Error())
			}
			log.Printf("receive mm: %s", string(data))
			log.Printf("<%d> INSERT INTO mms_forward(spcode, srctermid, desttermid, msgcontent, linkid) VALUES('%s', '%s', '%s', '%s', '%s')", rowId, msg.gateway, msg.from, msg.to, msg.msg, msg.linkid)

			return http.StatusOK, fmt.Sprintf("<response><id>%s</id><command>sync_mo_resp</command><result>0</result></response>", msg.id)
		}
	}

}

func postRequest(reqURL string, data []byte) ([]byte, error) {
	r, err := http.Post(reqURL, "application/xml; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	reply, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func postMessage(w http.ResponseWriter) {
	msg := `
	<?xml version="1.0" encoding="utf-8"?>
	<request>
	        <id>14092209191300001</id>
	        <command>sync_mo_req</command>
	        <operator>CM</operator>
	        <type>0</type>
	        <gateway>801057</gateway>
	        <msgid>14092209191300001</msgid>
	        <from>13910422228</from>
	        <to>10669501</to>
	        <serviceid>115103</serviceid>
	        <msgfmt>0</msgfmt>
	        <msg>1</msg>
	        <linkid>55877283390400855976</linkid>
	        <spid>mms01</spid>
	        <t>20140922 09:19:12</t>
	</request>
	`
	data, err := xml.Marshal(msg)
	if err != nil {
		http.Error(w, "xml Marshal failed", http.StatusBadRequest)
		return
	}
	reply, err := postRequest("http://42.62.0.188:10087/mmReceiver", data)
	log.Printf("receive response:", string(reply))
	if err != nil {
		http.Error(w, "post request failed", http.StatusBadRequest)
		return
	} else {
		log.Printf("receive response:", string(reply))
	}

}

func main() {
	mtn := martini.Classic()
	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/receipt?charset=utf8")
	db.SetMaxOpenConns(10)
	if err != nil {
		panic(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
	}
	defer db.Close()
	mtn.Map(db)
	logger := log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile)
	mtn.Map(logger)

	mtn.Get("/mmRequest", postMessage)
	mtn.Post("mmReceiver", mmReceiver)
	// mtn.Get("/mrReview", mrReivew)
	http.ListenAndServe(":10087", mtn)
	// mtn.Run()
}
