package ping

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"text/template"
	"time"
)

type Target struct {
	Protocol Protocol

	Name string
	Host string
	Port int

	Counter  int
	Interval time.Duration
	Timeout  time.Duration
}

func (t Target) String() string {
	return fmt.Sprintf("%s (%s://%s:%d)", t.Name, t.Protocol, t.Host, t.Port)
}

type Result struct {
	Total   int
	Success int

	Target *Target

	MinDuration   time.Duration
	MaxDuration   time.Duration
	TotalDuration time.Duration
}

// Avg return the average time of ping
func (r Result) Avg() time.Duration {
	if r.Success == 0 {
		return 0
	}
	return r.TotalDuration / time.Duration(r.Success)
}

// Failed return failed counter
func (r Result) Failed() int {
	return r.Total - r.Success
}

func (r Result) String() string {
	const resultTpl = `
Ping statistics {{.Target}}
	{{.Total}} probes sent.
	{{.Success}} successful, {{.Failed}} failed.
Approximate trip times:
	Minimum = {{.MinDuration}}, Maximum = {{.MaxDuration}}, Average = {{.Avg}}`

	tpl := template.New("result")
	t := template.Must(tpl.Parse(resultTpl))
	res := bytes.NewBufferString("")
	if err := t.Execute(res, r); err != nil {
		log.Printf("render result template err: %s", err)
	}
	return res.String()
}

/**
fastest
list name order by speed
*/
func NewTCPing(bufferSize int) *TCPing {
	p := &TCPing{}
	if bufferSize < 2 {
		p.resultCh = make(chan *Result)
	} else {
		p.resultCh = make(chan *Result, bufferSize)
	}
	return p
}

type TCPing struct {
	resultCh chan *Result
}

func (p *TCPing) GetResult() <-chan *Result {
	return p.resultCh
}

func (p *TCPing) Close() {
	close(p.resultCh)
}

func (p *TCPing) ping(host string, port int, timeout time.Duration) (duration time.Duration, remoteAddr net.Addr, err error) {
	dur, err := timeIt(func() error {
		conn, err := net.DialTimeout(TCP.String(), fmt.Sprintf("%s:%d", host, port), timeout)
		if err != nil {
			return err
		}
		remoteAddr = conn.RemoteAddr()

		conn.Close()
		return nil
	})

	if err != nil {
		return
	}
	duration = time.Duration(dur)
	return
}

func (p *TCPing) AsyncPing(ctx context.Context, target *Target) {
	t := time.NewTicker(target.Interval)
	defer t.Stop()
	fmt.Printf("Start Ping '%s'\n", target)
	result := &Result{Target: target}

	for {
		select {
		case <-ctx.Done():
			log.Printf("stop ping '%s' coz context deadlind exceed: %s\n", target, ctx.Err())
			return
		case <-t.C:
			if result.Total >= target.Counter && target.Counter != 0 {
				p.resultCh <- result
				return
			}
			duration, _, err := p.ping(target.Host, target.Port, target.Timeout)
			result.Total++

			if err != nil {
				log.Printf("Ping %s - failed: %s\n", target, err)
			} else {
				log.Printf("Ping %s - Connected - time=%s\n", target, duration)

				if result.MinDuration == 0 {
					result.MinDuration = duration
				}
				if result.MaxDuration == 0 {
					result.MaxDuration = duration
				}
				result.Success++
				if duration > result.MaxDuration {
					result.MaxDuration = duration
				} else if duration < result.MinDuration {
					result.MinDuration = duration
				}
				result.TotalDuration += duration
			}
		}
	}
}

func timeIt(f func() error) (int64, error) {
	startAt := time.Now()
	err := f()
	endAt := time.Now()
	return endAt.UnixNano() - startAt.UnixNano(), err
}
