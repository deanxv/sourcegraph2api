package cycletls

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	http "github.com/Danny-Dasilva/fhttp"
	"github.com/gorilla/websocket"
	"io"
	"log"
	nhttp "net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

// Options sets CycleTLS client options
type Options struct {
	URL                string            `json:"url"`
	Method             string            `json:"method"`
	Headers            map[string]string `json:"headers"`
	Body               string            `json:"body"`
	Ja3                string            `json:"ja3"`
	UserAgent          string            `json:"userAgent"`
	Proxy              string            `json:"proxy"`
	Cookies            []Cookie          `json:"cookies"`
	Timeout            int               `json:"timeout"`
	DisableRedirect    bool              `json:"disableRedirect"`
	HeaderOrder        []string          `json:"headerOrder"`
	OrderAsProvided    bool              `json:"orderAsProvided"` //TODO
	InsecureSkipVerify bool              `json:"insecureSkipVerify"`
	ForceHTTP1         bool              `json:"forceHTTP1"`
}

type cycleTLSRequest struct {
	RequestID string  `json:"requestId"`
	Options   Options `json:"options"`
}

// rename to request+client+options
type fullRequest struct {
	req     *http.Request
	client  http.Client
	options cycleTLSRequest
}

// Response contains Cycletls response data
type Response struct {
	RequestID string
	Status    int
	Body      string
	Headers   map[string]string
	Cookies   []*nhttp.Cookie
	FinalUrl  string
}

// JSONBody converts response body to json
func (re Response) JSONBody() map[string]interface{} {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(re.Body), &data)
	if err != nil {
		log.Print("Json Conversion failed " + err.Error() + re.Body)
	}
	return data
}

// CycleTLS creates full request and response
type CycleTLS struct {
	ReqChan  chan fullRequest
	RespChan chan Response
}

// ready Request
func processRequest(request cycleTLSRequest) (result fullRequest) {
	var browser = Browser{
		JA3:                request.Options.Ja3,
		UserAgent:          request.Options.UserAgent,
		Cookies:            request.Options.Cookies,
		InsecureSkipVerify: request.Options.InsecureSkipVerify,
		forceHTTP1:         request.Options.ForceHTTP1,
	}

	client, err := newClient(
		browser,
		request.Options.Timeout,
		request.Options.DisableRedirect,
		request.Options.UserAgent,
		request.Options.Proxy,
	)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest(strings.ToUpper(request.Options.Method), request.Options.URL, strings.NewReader(request.Options.Body))
	if err != nil {
		log.Fatal(err)
	}
	headerorder := []string{}
	//master header order, all your headers will be ordered based on this list and anything extra will be appended to the end
	//if your site has any custom headers, see the header order chrome uses and then add those headers to this list
	if len(request.Options.HeaderOrder) > 0 {
		//lowercase headers
		for _, v := range request.Options.HeaderOrder {
			lowercasekey := strings.ToLower(v)
			headerorder = append(headerorder, lowercasekey)
		}
	} else {
		headerorder = append(headerorder,
			"host",
			"connection",
			"cache-control",
			"device-memory",
			"viewport-width",
			"rtt",
			"downlink",
			"ect",
			"sec-ch-ua",
			"sec-ch-ua-mobile",
			"sec-ch-ua-full-version",
			"sec-ch-ua-arch",
			"sec-ch-ua-platform",
			"sec-ch-ua-platform-version",
			"sec-ch-ua-model",
			"upgrade-insecure-requests",
			"user-agent",
			"accept",
			"sec-fetch-site",
			"sec-fetch-mode",
			"sec-fetch-user",
			"sec-fetch-dest",
			"referer",
			"accept-encoding",
			"accept-language",
			"cookie",
		)
	}

	headermap := make(map[string]string)
	//TODO: Shorten this
	headerorderkey := []string{}
	for _, key := range headerorder {
		for k, v := range request.Options.Headers {
			lowercasekey := strings.ToLower(k)
			if key == lowercasekey {
				headermap[k] = v
				headerorderkey = append(headerorderkey, lowercasekey)
			}
		}

	}
	headerOrder := parseUserAgent(request.Options.UserAgent).HeaderOrder

	//ordering the pseudo headers and our normal headers
	req.Header = http.Header{
		http.HeaderOrderKey:  headerorderkey,
		http.PHeaderOrderKey: headerOrder,
	}
	//set our Host header
	u, err := url.Parse(request.Options.URL)
	if err != nil {
		panic(err)
	}

	//append our normal headers
	for k, v := range request.Options.Headers {
		if k != "Content-Length" {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("Host", u.Host)
	req.Header.Set("user-agent", request.Options.UserAgent)
	return fullRequest{req: req, client: client, options: request}

}

func dispatcher(res fullRequest) (response Response, err error) {
	defer res.client.CloseIdleConnections()
	finalUrl := res.options.Options.URL
	resp, err := res.client.Do(res.req)
	if err != nil {

		parsedError := parseError(err)

		headers := make(map[string]string)
		var cookies []*nhttp.Cookie
		return Response{RequestID: res.options.RequestID, Status: parsedError.StatusCode, Body: parsedError.ErrorMsg + "-> \n" + string(err.Error()), Headers: headers, Cookies: cookies, FinalUrl: finalUrl}, nil //normally return error here

	}
	defer resp.Body.Close()

	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		finalUrl = resp.Request.URL.String()
	}

	encoding := resp.Header["Content-Encoding"]
	content := resp.Header["Content-Type"]
	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return response, err
	}

	Body := DecompressBody(bodyBytes, encoding, content)
	headers := make(map[string]string)

	for name, values := range resp.Header {
		if name == "Set-Cookie" {
			headers[name] = strings.Join(values, "/,/")
		} else {
			for _, value := range values {
				headers[name] = value
			}
		}
	}
	cookies := convertFHTTPCookiesToNetHTTPCookies(resp.Cookies())
	return Response{
		RequestID: res.options.RequestID,
		Status:    resp.StatusCode,
		Body:      Body,
		Headers:   headers,
		Cookies:   cookies,
		FinalUrl:  finalUrl,
	}, nil

}

// Queue queues request in worker pool
func (client CycleTLS) Queue(URL string, options Options, Method string) {

	options.URL = URL
	options.Method = Method
	//TODO add timestamp to request
	opt := cycleTLSRequest{"Queued Request", options}
	response := processRequest(opt)
	client.ReqChan <- response
}

// Do creates a single request
func (client CycleTLS) Do(URL string, options Options, Method string) (response Response, err error) {

	options.URL = URL
	options.Method = Method
	// Set default values if not provided
	if options.Ja3 == "" {
		options.Ja3 = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,18-35-65281-45-17513-27-65037-16-10-11-5-13-0-43-23-51,29-23-24,0"
	}
	if options.UserAgent == "" {
		// Mac OS Chrome 121
		options.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"
	}
	opt := cycleTLSRequest{"cycleTLSRequest", options}

	res := processRequest(opt)
	response, err = dispatcher(res)
	if err != nil {
		return response, err
	}

	return response, nil
}

// Init starts the worker pool or returns a empty cycletls struct
func Init(workers ...bool) CycleTLS {
	if len(workers) > 0 && workers[0] {
		reqChan := make(chan fullRequest)
		respChan := make(chan Response)
		go workerPool(reqChan, respChan)
		log.Println("Worker Pool Started")

		return CycleTLS{ReqChan: reqChan, RespChan: respChan}
	}
	return CycleTLS{}

}

// Close closes channels
func (client CycleTLS) Close() {
	close(client.ReqChan)
	close(client.RespChan)

}

// Worker Pool
func workerPool(reqChan chan fullRequest, respChan chan Response) {
	//MAX
	for i := 0; i < 100; i++ {
		go worker(reqChan, respChan)
	}
}

// Worker
func worker(reqChan chan fullRequest, respChan chan Response) {
	for res := range reqChan {
		response, _ := dispatcher(res)
		respChan <- response
	}
}

func readSocket(reqChan chan fullRequest, c *websocket.Conn) {
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return
			}
			log.Print("Socket Error", err)
			return
		}
		request := new(cycleTLSRequest)

		err = json.Unmarshal(message, &request)
		if err != nil {
			log.Print("Unmarshal Error", err)
			return
		}

		reply := processRequest(*request)

		reqChan <- reply
	}
}

func writeSocket(respChan chan Response, c *websocket.Conn) {
	for {
		select {
		case r := <-respChan:
			message, err := json.Marshal(r)
			if err != nil {
				log.Print("Marshal Json Failed" + err.Error())
				continue
			}
			err = c.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Print("Socket WriteMessage Failed" + err.Error())
				continue
			}

		}

	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WSEndpoint exports the main cycletls function as we websocket connection that clients can connect to
func WSEndpoint(w nhttp.ResponseWriter, r *nhttp.Request) {
	upgrader.CheckOrigin = func(r *nhttp.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		//Golang Received a non-standard request to this port, printing request
		var data map[string]interface{}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Print("Invalid Request: Body Read Error" + err.Error())
		}
		err = json.Unmarshal(bodyBytes, &data)
		if err != nil {
			log.Print("Invalid Request: Json Conversion failed ")
		}
		body, err := PrettyStruct(data)
		if err != nil {
			log.Print("Invalid Request:", err)
		}
		headers, err := PrettyStruct(r.Header)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(headers)
		log.Println(body)
	} else {
		reqChan := make(chan fullRequest)
		respChan := make(chan Response)
		go workerPool(reqChan, respChan)

		go readSocket(reqChan, ws)
		//run as main thread
		writeSocket(respChan, ws)

	}

}

func setupRoutes() {
	nhttp.HandleFunc("/", WSEndpoint)
}

func main() {
	port, exists := os.LookupEnv("WS_PORT")
	var addr *string
	if exists {
		addr = flag.String("addr", ":"+port, "http service address")
	} else {
		addr = flag.String("addr", ":9112", "http service address")
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	setupRoutes()
	log.Fatal(nhttp.ListenAndServe(*addr, nil))
}

// 修改 SSEResponse 结构体，添加 FinalUrl 字段
type SSEResponse struct {
	RequestID string
	Status    int
	Data      string
	Done      bool
	FinalUrl  string // 添加 FinalUrl 字段
}

// 修改 dispatcher 函数以支持 SSE
//func dispatcherSSE(res fullRequest, sseChan chan<- SSEResponse) {
//	defer res.client.CloseIdleConnections()
//	finalUrl := res.options.Options.URL
//
//	resp, err := res.client.Do(res.req)
//	if err != nil {
//		parsedError := parseError(err)
//		sseChan <- SSEResponse{
//			RequestID: res.options.RequestID,
//			Status:    parsedError.StatusCode,
//			Data:      parsedError.ErrorMsg + "-> \n" + string(err.Error()),
//			Done:      true,
//			FinalUrl:  finalUrl,
//		}
//		return
//	}
//	defer resp.Body.Close()
//
//	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
//		finalUrl = resp.Request.URL.String()
//	}
//
//	scanner := bufio.NewScanner(resp.Body)
//	// 自定义分割函数，处理SSE格式
//	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
//		if atEOF && len(data) == 0 {
//			return 0, nil, nil
//		}
//
//		// 查找 "data: " 前缀
//		if i := bytes.Index(data, []byte("data: ")); i >= 0 {
//			// 查找下一个 "data: " 或结束
//			nextI := bytes.Index(data[i+6:], []byte("data: "))
//			if nextI >= 0 {
//				return i + 6 + nextI, data[i : i+6+nextI], nil
//			}
//			// 如果是最后一块数据
//			if atEOF {
//				return len(data), data[i:], nil
//			}
//		}
//
//		// 如果需要更多数据
//		if !atEOF {
//			return 0, nil, nil
//		}
//
//		// 处理剩余数据
//		return len(data), data, nil
//	})
//
//	for scanner.Scan() {
//		text := scanner.Text()
//		if strings.TrimSpace(text) != "" {
//			sseChan <- SSEResponse{
//				RequestID: res.options.RequestID,
//				Status:    resp.StatusCode,
//				Data:      text,
//				Done:      false,
//				FinalUrl:  finalUrl,
//			}
//		}
//	}
//
//	if err := scanner.Err(); err != nil {
//		sseChan <- SSEResponse{
//			RequestID: res.options.RequestID,
//			Status:    resp.StatusCode,
//			Data:      "Error reading stream: " + err.Error(),
//			Done:      true,
//			FinalUrl:  finalUrl,
//		}
//		return
//	}
//
//	sseChan <- SSEResponse{
//		RequestID: res.options.RequestID,
//		Status:    resp.StatusCode,
//		Data:      "",
//		Done:      true,
//		FinalUrl:  finalUrl,
//	}
//}

//func dispatcherSSE(res fullRequest, sseChan chan<- SSEResponse) {
//	defer res.client.CloseIdleConnections()
//	finalUrl := res.options.Options.URL
//
//	resp, err := res.client.Do(res.req)
//	if err != nil {
//		parsedError := parseError(err)
//		sseChan <- SSEResponse{
//			RequestID: res.options.RequestID,
//			Status:    parsedError.StatusCode,
//			Data:      parsedError.ErrorMsg + "-> \n" + string(err.Error()),
//			Done:      true,
//			FinalUrl:  finalUrl,
//		}
//		return
//	}
//	defer resp.Body.Close()
//
//	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
//		finalUrl = resp.Request.URL.String()
//	}
//
//	reader := bufio.NewReader(resp.Body)
//	var buf bytes.Buffer
//
//	const maxRetries = 3
//	retries := 0
//
//	for {
//		// 读取直到"data: "
//		line, err := reader.ReadString('\n')
//		if err != nil {
//			if err == io.EOF {
//				break
//			}
//
//			if retries < maxRetries {
//				retries++
//				time.Sleep(time.Second * time.Duration(retries))
//				continue
//			}
//
//			sseChan <- SSEResponse{
//				RequestID: res.options.RequestID,
//				Status:    resp.StatusCode,
//				Data:      "Error reading stream: " + err.Error(),
//				Done:      true,
//				FinalUrl:  finalUrl,
//			}
//			return
//		}
//
//		// 重置重试计数
//		retries = 0
//
//		// 处理数据
//		if strings.HasPrefix(line, "data: ") {
//			// 清空buffer
//			buf.Reset()
//
//			// 写入新数据（去掉"data: "前缀）
//			data := strings.TrimPrefix(line, "data: ")
//			data = strings.TrimSpace(data)
//
//			if data != "" {
//				sseChan <- SSEResponse{
//					RequestID: res.options.RequestID,
//					Status:    resp.StatusCode,
//					Data:      data,
//					Done:      false,
//					FinalUrl:  finalUrl,
//				}
//			}
//		}
//
//		// 可选：检查是否有结束标记
//		if strings.Contains(line, "[DONE]") {
//			break
//		}
//	}
//
//	// 发送完成信号
//	sseChan <- SSEResponse{
//		RequestID: res.options.RequestID,
//		Status:    resp.StatusCode,
//		Data:      "",
//		Done:      true,
//		FinalUrl:  finalUrl,
//	}
//}

func dispatcherSSE(res fullRequest, sseChan chan<- SSEResponse) {
	defer res.client.CloseIdleConnections()

	finalUrl := res.options.Options.URL

	resp, err := res.client.Do(res.req)
	if err != nil {
		parsedError := parseError(err)
		sseChan <- SSEResponse{
			RequestID: res.options.RequestID,
			Status:    parsedError.StatusCode,
			Data:      fmt.Sprintf("%s-> \n%s", parsedError.ErrorMsg, err.Error()),
			Done:      true,
			FinalUrl:  finalUrl,
		}
		return
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx状态码可能表示错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorMsg := string(bodyBytes)
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("HTTP error status: %d", resp.StatusCode)
		}

		sseChan <- SSEResponse{
			RequestID: res.options.RequestID,
			Status:    resp.StatusCode,
			Data:      errorMsg,
			Done:      true,
			FinalUrl:  finalUrl,
		}
		return
	}

	// 更新最终URL（考虑重定向）
	if resp.Request != nil && resp.Request.URL != nil {
		finalUrl = resp.Request.URL.String()
	}

	reader := bufio.NewReader(resp.Body)
	const maxRetries = 3
	retries := 0

	for {
		// 读取直到换行符
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			if retries < maxRetries {
				retries++
				time.Sleep(time.Second * time.Duration(retries))
				continue
			}

			sseChan <- SSEResponse{
				RequestID: res.options.RequestID,
				Status:    resp.StatusCode,
				Data:      "Error reading stream: " + err.Error(),
				Done:      true,
				FinalUrl:  finalUrl,
			}
			return
		}

		// 重置重试计数
		retries = 0

		// 去除行尾的空白字符
		line = strings.TrimSpace(line)

		// 跳过空行
		if line == "" {
			continue
		}

		// 处理数据行
		//if strings.HasPrefix(line, "data: ") {
		data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if data != "" {
			sseChan <- SSEResponse{
				RequestID: res.options.RequestID,
				Status:    resp.StatusCode,
				Data:      data,
				Done:      false,
				FinalUrl:  finalUrl,
			}
		}
		//}

		// 检查是否有结束标记
		if strings.Contains(line, "[DONE]") {
			break
		}
	}

	// 发送完成信号
	sseChan <- SSEResponse{
		RequestID: res.options.RequestID,
		Status:    resp.StatusCode,
		Data:      "[DONE]",
		Done:      true,
		FinalUrl:  finalUrl,
	}
}

// 修改 Do 方法以支持 SSE
func (client CycleTLS) DoSSE(URL string, options Options, Method string) (<-chan SSEResponse, error) {
	sseChan := make(chan SSEResponse)

	options.URL = URL
	options.Method = Method
	if options.Ja3 == "" {
		options.Ja3 = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,18-35-65281-45-17513-27-65037-16-10-11-5-13-0-43-23-51,29-23-24,0"
	}
	if options.UserAgent == "" {
		options.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"
	}

	opt := cycleTLSRequest{"cycleTLSRequest", options}
	res := processRequest(opt)

	go func() {
		defer close(sseChan)
		dispatcherSSE(res, sseChan)
	}()

	return sseChan, nil
}
