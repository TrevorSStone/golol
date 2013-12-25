package rtmpsclient

func InvokeIdGenerator(idChannel chan<- int) {
	i := 2
	for {
		idChannel <- i
		i++
	}
}

func (client RTMPSClient) GetNextID() int {
	return <-client.idGeneratorChannel
}
