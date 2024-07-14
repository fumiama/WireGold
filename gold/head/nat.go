package head

// Notify 是 map[peerip]{network, endpoint}
type Notify = map[string][2]string

// Query 是 peerips 组成的数组
type Query = []string
