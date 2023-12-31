package hail

type operation int

const (
	// Subscribe 訂閱 (subscribe new topic)
	Subscribe operation = iota
	// Publish 發布訊息 (publish the topic[...] subscriber)
	Publish
	// AsyncPublish 非同步發布訊息  (async publish the topic[...] subscriber)
	AsyncPublish
	// Unsubscribe 取消訂閱 (unsubscribe the topic)
	Unsubscribe
	// UnSubscribeAll 全部取消訂閱  (unsubscribe all topic)
	UnSubscribeAll
	// CloseTopic 關閉此標題 (close the topic)
	CloseTopic
	// ShutDown 此訂閱服務關機 (shutdown this pub/sub service)
	ShutDown
)

// pubSubPattern 集合topic,capacity是容量 (topic set, buffer channel size)
type pubSub struct {
	commandChan chan cmd // 接收指令的channel
	bufferSize  int
}

type cmd struct {
	opCode operation // 指令 (command)
	topics []string  // 訂閱的主題 (subscribe topics)
	ch     chan *box // 使用的channel (channel used by subscriber)
	msg    *box      // 訊息內文 (msg data)
}

// pubSubNew 創建一個訂閱者模式 (create a new pub/sub pattern)
func pubSubNew(bufferSize int) *pubSub {
	ps := &pubSub{make(chan cmd), bufferSize}
	go ps.start()
	return ps
}

// Sub 創建一個新的訂閱頻道, 並將channel回傳 (create a channel for subscribe topic, and return it)
func (ps *pubSub) Sub(topics ...string) chan *box {
	return ps.sub(Subscribe, topics...)
}

func (ps *pubSub) sub(op operation, topics ...string) chan *box {
	ch := make(chan *box, ps.bufferSize)
	ps.commandChan <- cmd{opCode: op, topics: topics, ch: ch}
	return ch
}

// AddSub 將要訂閱的Topic加到現有的channel (add topics into the subscribe channel)
func (ps *pubSub) AddSub(ch chan *box, topics ...string) {
	ps.commandChan <- cmd{opCode: Subscribe, topics: topics, ch: ch}
}

// Pub 發布訊息 (publish message to subscribe channels)
func (ps *pubSub) Pub(msg *box, topics ...string) {
	ps.commandChan <- cmd{opCode: Publish, topics: topics, msg: msg}
}

// AsyncPub 非同步的發布訊息，內部機制 (async publish message to subscribe channels)
func (ps *pubSub) AsyncPub(msg *box, topics ...string) {
	ps.commandChan <- cmd{opCode: AsyncPublish, topics: topics, msg: msg}
}

// Unsub 取消訂閱  (unsubscribe topic, if topics is null, it will unsubscribe all)
func (ps *pubSub) Unsub(ch chan *box, topics ...string) {
	// 如果不寫topic，視為將全部topic都取消訂閱
	if len(topics) == 0 {
		ps.commandChan <- cmd{opCode: UnSubscribeAll, ch: ch}
		return
	}

	ps.commandChan <- cmd{opCode: Unsubscribe, topics: topics, ch: ch}
}

// Close 關閉Topic, 相關有訂閱的channel都會被取消 (close topics, if channel subscribe it, will auto unsubscribe)
func (ps *pubSub) Close(topics ...string) {
	ps.commandChan <- cmd{opCode: CloseTopic, topics: topics}
}

// Shutdown 關閉所有有訂閱的Channel (close all channels)
func (ps *pubSub) Shutdown() {
	ps.commandChan <- cmd{opCode: ShutDown}
}

func (ps *pubSub) start() {

	// 初始化暫存在記憶體的資料(topicsMap & revertTopicsOfChannelMap)
	// init register data
	reg := register{
		topics:    make(map[string]map[chan *box]bool),
		revTopics: make(map[chan *box]map[string]bool),
	}

loop:
	for cmd := range ps.commandChan {
		if cmd.topics == nil {
			switch cmd.opCode {
			case UnSubscribeAll:
				reg.removeChannel(cmd.ch)

			case ShutDown:
				break loop
			}

			continue loop
		}

		for _, topic := range cmd.topics {
			switch cmd.opCode {
			case Subscribe:
				reg.add(topic, cmd.ch)

			case Publish:
				reg.send(topic, cmd.msg)

			case AsyncPublish:
				reg.sendAsync(topic, cmd.msg)

			case Unsubscribe:
				reg.remove(topic, cmd.ch)

			case CloseTopic:
				reg.removeTopic(topic)
			}
		}
	}

	// 當跳出迴圈要結束時，將所有未關閉的topic channel進行移除
	// while break loop, close all topic channel ,release all
	for topic, chans := range reg.topics {
		for ch := range chans {
			reg.remove(topic, ch)
		}
	}
}
