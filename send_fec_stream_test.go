package quic

import (
	"crypto/rand"
	mrand "math/rand"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/quic-go/quic-go/internal/mocks"
	"github.com/quic-go/quic-go/internal/protocol"
	"go.uber.org/mock/gomock"
)

/*
TODO
- Use the test from send_stream_test.go It("retransmits data until everything has been acknowledged", func() { ... } to drop random source symbols in order to trigger the repair functionality
go run github.com/onsi/ginkgo/v2/ginkgo -r -v -cover -randomize-all -randomize-suites -trace -skip-package integrationtests

Expecting 1357 but got 1343. That's a difference of 14 bytes.

maybeGetShortHeaderPacket

*/

var _ = Describe("Send FEC Stream", func() {
	// TODO (ddritzenhoff) used to be 1337
	const streamID protocol.StreamID = 1338

	var (
		str *sendStream
		// strWithTimeout io.Writer // str wrapped with gbytes.TimeoutWriter
		mockFC     *mocks.MockStreamFlowController
		mockSender *MockStreamSender
	)

	BeforeEach(func() {
		mockSender = NewMockStreamSender(mockCtrl)
		mockFC = mocks.NewMockStreamFlowController(mockCtrl)
		// str = newSendStream(streamID, mockSender, mockFC)
		str = newSendStreamWithFEC(streamID, mockSender, mockFC, true)

		// timeout := scaleDuration(250 * time.Millisecond)
		// strWithTimeout = gbytes.TimeoutWriter(str, timeout)
	})

	// This test is kind of an integration test.
	// It writes 4 MB of data, and pops STREAM frames that sometimes are and sometimes aren't limited by flow control.
	// Half of these STREAM frames are then received and their content saved, while the other half is reported lost
	// and has to be retransmitted.
	It("retransmits data until everything has been acknowledged", func() {
		const dataLen = 1 << 22 // 4 MB
		mockSender.EXPECT().onHasStreamData(streamID).AnyTimes()
		mockFC.EXPECT().SendWindowSize().DoAndReturn(func() protocol.ByteCount {
			return protocol.ByteCount(mrand.Intn(500)) + 50
		}).AnyTimes()
		mockFC.EXPECT().AddBytesSent(gomock.Any()).AnyTimes()

		data := make([]byte, dataLen)
		_, err := rand.Read(data)
		Expect(err).ToNot(HaveOccurred())
		done := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			_, err := str.Write(data)
			Expect(err).ToNot(HaveOccurred())
			str.Close()
		}()

		var completed bool
		mockSender.EXPECT().onStreamCompleted(streamID).Do(func(protocol.StreamID) { completed = true })

		received := make([]byte, dataLen)
		for {
			if completed {
				break
			}
			f, ok, _ := str.popStreamFrame(protocol.ByteCount(mrand.Intn(300)+100), protocol.Version1)
			if !ok {
				continue
			}
			sf := f.Frame
			// 50%: acknowledge the frame and save the data
			// 50%: lose the frame
			if mrand.Intn(100) < 50 {
				copy(received[sf.Offset:sf.Offset+sf.DataLen()], sf.Data)
				f.Handler.OnAcked(f.Frame)
			} else {
				f.Handler.OnLost(f.Frame)
			}
		}
		Expect(received).To(Equal(data))
	})

	// This test is kind of an integration test.
	// It writes 4 MB of data, and pops STREAM frames that sometimes are and sometimes aren't limited by flow control.
	// Half of these STREAM frames are then received and their content saved, while the other half is reported lost
	// and has to be retransmitted.
	It("purposely drop the second packet to force a repair", func() {
		const dataLen = 2000 // 4 MB
		mockSender.EXPECT().onHasStreamData(streamID).AnyTimes()
		mockFC.EXPECT().SendWindowSize().DoAndReturn(func() protocol.ByteCount {
			return protocol.ByteCount(mrand.Intn(500)) + 50
		}).AnyTimes()
		mockFC.EXPECT().AddBytesSent(gomock.Any()).AnyTimes()

		data := make([]byte, dataLen)
		_, err := rand.Read(data)
		Expect(err).ToNot(HaveOccurred())
		done := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			_, err := str.Write(data)
			Expect(err).ToNot(HaveOccurred())
			str.Close()
		}()

		var completed bool
		mockSender.EXPECT().onStreamCompleted(streamID).Do(func(protocol.StreamID) { completed = true })

		received := make([]byte, dataLen)
		for {
			if completed {
				break
			}
			f, ok, _ := str.popStreamFrame(protocol.ByteCount(mrand.Intn(300)+100), protocol.Version1)
			if !ok {
				continue
			}
			sf := f.Frame
			// 50%: acknowledge the frame and save the data
			// 50%: lose the frame
			if mrand.Intn(100) < 50 {
				copy(received[sf.Offset:sf.Offset+sf.DataLen()], sf.Data)
				f.Handler.OnAcked(f.Frame)
			} else {
				f.Handler.OnLost(f.Frame)
			}
		}
		Expect(received).To(Equal(data))
	})

})
