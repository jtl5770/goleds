package hardware

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	p "lautenbacher.net/goleds/producer"
)

func simulateLed(segmentID int, values []p.Led) {
	var buf strings.Builder
	buf.Grow(len(values))

	fmt.Print("[")
	for _, v := range values {
		if v.IsEmpty() {
			buf.WriteString(" ")
		} else {
			value := byte(math.Round(float64(v.Red+v.Green+v.Blue) / 3.0))
			if value == 1 {
				buf.WriteString("▁")
			} else if value == 2 {
				buf.WriteString("▂")
			} else if value <= 4 {
				buf.WriteString("▃")
			} else if value <= 8 {
				buf.WriteString("▄")
			} else if value <= 16 {
				buf.WriteString("▅")
			} else if value <= 24 {
				buf.WriteString("▆")
			} else if value <= 32 {
				buf.WriteString("▇")
			} else {
				buf.WriteString("█")
			}
		}
	}
	fmt.Print(buf.String())
	if segmentID == 0 {
		fmt.Print("]       ")
	} else {
		fmt.Print("]\r")
	}
}

func readSingle(reader *bufio.Reader, w chan rune) {
	r, _, err := reader.ReadRune()
	if err != nil {
		panic(err)
	}
	w <- r
}

func simulateSensors(sensorReader chan Trigger, sig chan bool) {
	reader := bufio.NewReader(os.Stdin)
	work := make(chan rune)
	defer close(work)

	for {
		go readSingle(reader, work)
		select {
		case <-sig:
			log.Println("Ending SensorDriver simulation go-routine")
			return
		case r := <-work:
			if r == '1' {
				sensorReader <- Trigger{"S0", 80, time.Now()}
			} else if r == '2' {
				sensorReader <- Trigger{"S1", 80, time.Now()}
			} else if r == '3' {
				sensorReader <- Trigger{"S2", 80, time.Now()}
			} else if r == '4' {
				sensorReader <- Trigger{"S3", 80, time.Now()}
			}
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
