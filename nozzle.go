package main
import (
	"code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"context"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"log"
	"net/http"
	"os"
)
var counter = Counter{0}
func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/reset", reset)
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
	rlpAddr := os.Getenv("LOG_STREAM_ADDR")
	if rlpAddr == "" {
		log.Fatal("LOG_STREAM_ADDR is required")
	}
	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatalf("TOKEN is required")
	}
	c := loggregator.NewRLPGatewayClient(
		rlpAddr,
		loggregator.WithRLPGatewayClientLogger(log.New(os.Stderr, "", log.LstdFlags)),
		loggregator.WithRLPGatewayHTTPClient(&tokenAttacher{
			token: token,
		}),
	)
	var selectors = []*loggregator_v2.Selector{
		{Message: &loggregator_v2.Selector_Log{Log: &loggregator_v2.LogSelector{}}},
		{Message: &loggregator_v2.Selector_Counter{Counter: &loggregator_v2.CounterSelector{}}},
		{Message: &loggregator_v2.Selector_Event{Event: &loggregator_v2.EventSelector{}}},
		{Message: &loggregator_v2.Selector_Gauge{Gauge: &loggregator_v2.GaugeSelector{}}},
		{Message: &loggregator_v2.Selector_Timer{Timer: &loggregator_v2.TimerSelector{}}},
	}
	es := c.Stream(context.Background(), &loggregator_v2.EgressBatchRequest{
		ShardId: "myshardid",
		Selectors: selectors,
	})
	marshaler := jsonpb.Marshaler{}
	for {
		for _, e := range es() {
			if err := marshaler.Marshal(os.Stdout, e); err != nil {
				log.Fatal(err)
			}else{
				counter.increment()
			}
		}
	}
}
type tokenAttacher struct {
	token string
}
func (a *tokenAttacher) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", a.token)
	return http.DefaultClient.Do(req)
}
type Counter struct {
	count int
}
func (self Counter) currentValue() int {
	return self.count
}
func (self *Counter) increment() {
	self.count++
}
func (self *Counter) reset() {
	self.count = 0
}
func handler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "Message, %s %d", "count: ", counter.currentValue())
}
func reset(w http.ResponseWriter, r *http.Request) {
	counter.reset()
}
