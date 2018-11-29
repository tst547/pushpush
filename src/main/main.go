package main

import (
	"bufio"
	"bytes"
	"cm"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

/**
列表文件信息
*/
func listFile(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	req.ParseForm()
	filePath := req.Form.Get("filePath")
	if filePath == "" {
		fList := GetLogicalDrives()
		var files [20]cm.File
		for i, v := range fList {
			files[i] = cm.File{
				Name:  v,
				Path:  v+"\\",
				Size:  0,
				IsDir: true,
			}
		}
		genList := cm.Base{
			Err: 0,
			Msg: files[:len(fList)],
		}
		results, _ := json.Marshal(genList)
		w.Write(results)
		fmt.Print(string(results))
		return
	}
	f, _ := os.Open(filePath)
	fInfo, _ := f.Stat()
	if fInfo.IsDir() {
		osFList, _ := ioutil.ReadDir(filePath)
		var files [1024]cm.File
		for i, v := range osFList {
			bf := bytes.Buffer{}
			bf.WriteString(filePath)
			bf.WriteRune(os.PathSeparator)
			bf.WriteString(v.Name())
			fl,_ :=filepath.Abs(bf.String())
			files[i] = cm.File{
				Name:  v.Name(),
				Path:  fl,
				Size:  v.Size(),
				IsDir: v.IsDir(),
			}
		}
		genList := cm.Base{
			Err: 0,
			Msg: files[:len(osFList)],
		}
		results, _ := json.Marshal(genList)
		w.Write(results)
		fmt.Print(string(results))
		return
	}
}

/**
接收文件
*/
func fileRecv(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	e := req.ParseForm()
	checkErr(e)
	var rm = new(cm.RecvMsg)
	err := json.Unmarshal([]byte(req.Form.Get("RecvMsg")), rm)
	checkErr(err)
	go cm.FileRecv(rm)
	results, _ := json.Marshal(cm.Base{
		Err: 0,
		Msg: "Is connected",
	})
	fmt.Print(string(results))
	w.Write(results)
}

/**
get参数 : filePath 文件路径
下载指定文件
*/
func fileDownLoad(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	filePath := req.PostForm.Get("filePath")
	file, err := os.Open(filePath)
	log.Println(filePath)
	defer file.Close()
	checkErr(err)
	fileInfo, errs := file.Stat()
	checkErr(errs)
	var offset int64 = 0
	var fileLen = fileInfo.Size()
	rag := req.Header.Get("Range")
	var index int
	if len(rag) > 0 && strings.Contains(rag, "-") {
		index, err = strconv.Atoi(rag[:strings.Index(rag, "-")])
		checkErr(err)
	} else {
		index = -1
	}
	log.Println(index)
	switch strings.Index(rag, "-") {
	case 0:
		r, _ := strconv.ParseInt(rag, 10, 64)
		offset = fileLen + r
	case len(rag) - 1:
		r, _ := strconv.ParseInt(strings.Replace(rag, "-", "", -1),
			10, 64)
		offset = r
	case -1:
		offset = 0
	default:
		strSlice := strings.Split(rag, "-")[1:]
		offset, _ = strconv.ParseInt(strSlice[0], 10, 64)
		fileLen, _ = strconv.ParseInt(strSlice[1], 10, 64)
	}
	w.Header().Add("Accept-Ranges", "bytes")
	w.Header().Add("Content-disposition", "attachment;filename="+fileInfo.Name())
	w.Header().Add("Content-Length", strconv.FormatInt(fileLen-offset, 10))
	file.Seek(offset, 0)
	bfRd := bufio.NewReader(file)
	for {
		data := make([]byte, 1024*128)
		n, err := bfRd.Read(data)
		if nil != err && err != io.EOF {
			log.Println(err.Error())
			return
		}
		if nil != err || n == 0 {
			break
		}
		_, errw := w.Write(data[:n])
		if nil != errw {
			log.Println(errw.Error())
			return
		}
	}
}

/**
连接测试
*/
func test(w http.ResponseWriter, req *http.Request) {
	results, _ := json.Marshal(cm.Base{
		Err: 0,
		Msg: "test finish",
	})
	w.Write(results)
}

func main() {
	httpPort := 8888
	go cm.ScanPort(22555, httpPort)
	http.HandleFunc("/test", test)
	http.HandleFunc("/video", video)
	http.HandleFunc("/recv", fileRecv)
	http.HandleFunc("/list", listFile)
	http.HandleFunc("/fileDL", fileDownLoad)
	err := http.ListenAndServe(":"+strconv.Itoa(httpPort), nil)
	if err != nil {
		log.Println("ListenAndServe: ", err.Error())
		return
	}
}
func video(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	filePath := req.Form.Get("filePath")
	fmt.Print(filePath)
	http.ServeFile(w,req,filePath)
}

/**
获取系统盘符
*/
func GetLogicalDrives() []string {
	kernel32 := syscall.MustLoadDLL("kernel32.dll")
	GetLogicalDrives := kernel32.MustFindProc("GetLogicalDrives")
	n, _, _ := GetLogicalDrives.Call()
	s := strconv.FormatInt(int64(n), 2)
	var drivesAll = []string{"A:", "B:", "C:", "D:", "E:", "F:", "G:", "H:", "I:", "J:", "K:", "L:", "M:", "N:", "O:", "P：", "Q：", "R：", "S：", "T：", "U：", "V：", "W：", "X：", "Y：", "Z："}
	temp := drivesAll[0:len(s)]
	var d []string
	for i, v := range s {
		if v == 49 {
			l := len(s) - i - 1
			d = append(d, temp[l])
		}
	}

	var drives []string
	for i, v := range d {
		drives = append(drives[i:], append([]string{v}, drives[:i]...)...)
	}
	return drives

}

func checkErr(err error) {
	if nil != err {
		return
	}
}
