package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
)

var config Config

type Config struct {
	MongoURI             string   `json:"mongoURI"`
	DBName               string   `json:"dbName"`
	CollectionName       string   `json:"collectionName"`
	Admins               []string `json:"admins"`
	RefreshInterval      int      `json:"refreshInterval"`
	InterfaceName        string   `json:"interfaceName"`
	Collection           *mongo.Collection
	Peers                map[string]*Peer
	TotalRx              uint64
	TotalTx              uint64
	CurrentRx            uint64
	CurrentTx            uint64
	ServerEndpoint       string `json:"serverEndpoint"`
	ServerPublicKey      string `json:"serverPublicKey"`
	ServerNetworkAddress string `json:"serverNetworkAddress"`
}

type Peer struct {
	ID              primitive.ObjectID `bson:"_id" json:"id,omitempty"`
	Name            string             `bson:"name,omitempty" json:"name"`
	PrivateKey      string             `bson:"privatekey,omitempty" json:"privatekey"`
	PublicKey       string             `bson:"publicKey,omitempty" json:"publicKey"`
	PresharedKey    string             `bson:"presharedKey,omitempty" json:"presharedKey"`
	AllowedIps      string             `bson:"allowedIps,omitempty" json:"allowedIps"`
	ExpiresAt       uint64             `bson:"expiresAt,omitempty" json:"expiresAt"`
	LatestHandshake uint64             `bson:"-" json:"latestHandshake"`
	TotalRx         uint64             `json:"totalRx"`
	TotalTx         uint64             `json:"totalTx"`
	CurrentRx       uint64             `bson:"-" json:"currentRx"`
	CurrentTx       uint64             `bson:"-" json:"currentTx"`
	Suspended       bool               `bson:"suspended,omitempty" json:"suspended"`
	PreviousTotalRx uint64
	PreviousTotalTx uint64
}

type IPAddress struct {
	Octets [4]int
}

func (a *IPAddress) Increment() {
	if a.Octets[3] < 254 {
		a.Octets[3]++
	} else {
		a.Octets[3] = 1
		if a.Octets[2] < 254 {
			a.Octets[2]++
		} else {
			a.Octets[2] = 1
			if a.Octets[1] < 254 {
				a.Octets[1]++
			} else {
				a.Octets[1] = 1
				if a.Octets[0] < 254 {
					a.Octets[0]++
				} else {
					panic(fmt.Sprintf("cant increment address, %d", a.Octets))
				}
			}
		}
	}
}

func (a *IPAddress) ToString() string {
	return fmt.Sprintf("%d.%d.%d.%d", a.Octets[0], a.Octets[1], a.Octets[2], a.Octets[3])
}

func (a *IPAddress) Parse(address string) {
	serverNetworkAddressOctets := strings.Split(address, ".")
	for i, o := range serverNetworkAddressOctets {
		a.Octets[i], _ = strconv.Atoi(o)
	}
}

func createPeer(name string) (*Peer, error) {
	// check if name is already taken
	for _, peer := range config.Peers {
		if name == peer.Name {
			return nil, errors.New("duplicate name")
		}
	}

	// find unused network address for peer
	var a IPAddress
	a.Parse(config.ServerNetworkAddress)
	a.Increment()
	cmd := exec.Command("wg-quick", "strip", config.InterfaceName)
	allPeersBytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	allPeers := string(allPeersBytes)
	for strings.Contains(allPeers, a.ToString()) {
		a.Increment()
	}

	// create private key
	cmd = exec.Command("wg", "genkey")
	privateKeyBytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	clientPrivateKey := strings.TrimSpace(string(privateKeyBytes))

	// create publick key
	echoCommand := exec.Command("echo", clientPrivateKey)
	genkeyCommand := exec.Command("wg", "pubkey")
	genkeyCommand.Stdin, _ = echoCommand.StdoutPipe()
	err = echoCommand.Start()
	if err != nil {
		return nil, err
	}
	publicKeyBytes, err := genkeyCommand.Output()
	if err != nil {
		return nil, err
	}
	clientPublicKey := strings.TrimSpace(string(publicKeyBytes))

	// create preshared key
	cmd = exec.Command("wg", "genpsk")
	presharedKeyBytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	presharedKey := strings.TrimSpace(string(presharedKeyBytes))

	// add peer
	config.Peers[clientPublicKey] = &Peer{
		ID:           primitive.NewObjectID(),
		Name:         name,
		PublicKey:    clientPublicKey,
		PrivateKey:   clientPrivateKey,
		PresharedKey: presharedKey,
		AllowedIps:   a.ToString() + "/32",
		ExpiresAt:    uint64(time.Now().Unix() + 60*60*24*30),
	}

	// update config file
	f, err := os.OpenFile(fmt.Sprintf("/etc/wireguard/%s.conf", config.InterfaceName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write([]byte(fmt.Sprintf("\n[Peer]\nPublicKey = %s\nPresharedKey = %s\nAllowedIPs = %s\n", clientPublicKey, presharedKey, a.ToString()+"/32"))); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	// get striped config
	cmd = exec.Command("wg-quick", "strip", "wg0")
	configBytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// write striped config to a file
	err = os.WriteFile("/root/wireguard-ui/wg0.conf", configBytes, 0644)
	if err != nil {
		panic(err)
	}

	// save chagnes to main config file
	cmd = exec.Command("wg", "syncconf", config.InterfaceName, fmt.Sprintf("/root/wireguard-ui/%s.conf", config.InterfaceName))
	_, err = cmd.Output()
	if err != nil {
		return nil, err
	}

	// add peer to database
	_, err = config.Collection.InsertOne(context.TODO(), config.Peers[clientPublicKey])
	if err != nil {
		return nil, err
	}
	return config.Peers[clientPublicKey], nil
}

// func deletePeer(name string) {}

// func renamePeer(name string, newName string) {}

func getPeers() {
	// get peers info from wg
	cmd := exec.Command("wg", "show", config.InterfaceName, "dump")
	bytes, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
	}

	// define temp vars
	var totalRx uint64
	var totalTx uint64
	var currentRx uint64
	var currentTx uint64

	// each line contains a peer's info, excluding the first line whichis the interface info
	peerLines := strings.Split(strings.TrimSpace(string(bytes)), "\n")[1:]

	var publicKey string
	var newTotalTx uint64
	var newTotalRx uint64
	for _, p := range peerLines {
		info := strings.Split(p, "\t")
		publicKey = info[0]

		if config.Peers[publicKey] == nil {
			config.Peers[publicKey] = &Peer{}
		}

		// find public key
		publicKey = info[0]

		// update preshared key
		config.Peers[publicKey].PresharedKey = strings.TrimSpace(info[1])

		// update current rx and tx
		newTotalTx, _ = strconv.ParseUint(string(info[5]), 10, 64)
		newTotalRx, _ = strconv.ParseUint(string(info[6]), 10, 64)
		config.Peers[publicKey].CurrentRx = newTotalRx - config.Peers[publicKey].TotalRx
		config.Peers[publicKey].CurrentTx = newTotalTx - config.Peers[publicKey].TotalTx

		// update latest handshake
		config.Peers[publicKey].LatestHandshake, _ = strconv.ParseUint(string(info[4]), 10, 64)

		// update total rx and tx
		config.Peers[publicKey].TotalTx = newTotalTx + config.Peers[publicKey].PreviousTotalTx
		config.Peers[publicKey].TotalRx = newTotalRx + config.Peers[publicKey].PreviousTotalRx

		// update servers total and current rx and tx
		totalRx += config.Peers[publicKey].TotalRx
		totalTx += config.Peers[publicKey].TotalTx
		currentRx += config.Peers[publicKey].CurrentRx
		currentTx += config.Peers[publicKey].CurrentTx

		// suspend expired peers
		if config.Peers[publicKey].ExpiresAt < uint64(time.Now().Unix()) && !config.Peers[publicKey].Suspended {
			// create invalid preshared key
			invalid := config.Peers[publicKey].ID.Hex() + "AAAAAAAAAAAAAAAAAAA="

			// replace peer's preshared key with the invalid one
			cmd := exec.Command("sh", "/root/wg-stats/scripts/replace-string.sh", fmt.Sprintf("/etc/wireguard/%s.conf", config.InterfaceName), config.Peers[publicKey].PresharedKey, invalid, "&&", "sh", "/root/wg-stats/scripts/restart-wireguard.sh")
			_, err := cmd.Output()
			if err != nil {
				fmt.Println(err)
				continue
			}

			// save chagnes to main config file
			cmd = exec.Command("wg", "syncconf", config.InterfaceName, fmt.Sprintf("/root/wg-stats/%s.conf", config.InterfaceName))
			_, err = cmd.Output()
			if err != nil {
				fmt.Println(err)
				continue
			}

			// update database
			config.Peers[publicKey].Suspended = true
			_, err = config.Collection.UpdateOne(context.TODO(), bson.M{"publicKey": config.Peers[publicKey].PublicKey}, bson.M{"$set": bson.M{"suspended": true}})
			if err != nil {
				fmt.Println(err)
				continue
			}
		}

		// revive suspended peers
		if config.Peers[publicKey].Suspended && config.Peers[publicKey].ExpiresAt > uint64(time.Now().Unix()) {
			// create invalid preshared key
			invalid := config.Peers[publicKey].ID.Hex() + "AAAAAAAAAAAAAAAAAAA="

			// get peer's info from database
			p := Peer{}
			res := config.Collection.FindOne(context.Background(), bson.M{"publicKey": config.Peers[publicKey].PublicKey})
			err = res.Decode(&p)
			if err != nil {
				panic(err)
			}

			// replace invalid preshared key with the correct one from database
			cmd := exec.Command("sh", "/root/wg-stats/scripts/replace-string.sh", fmt.Sprintf("/etc/wireguard/%s.conf", config.InterfaceName), invalid, p.PresharedKey, "&&", "sh", "/root/wg-stats/scripts/restart-wireguard.sh")
			_, err := cmd.Output()
			if err != nil {
				panic(err)
			}

			// save chagnes to main config file
			cmd = exec.Command("wg", "syncconf", config.InterfaceName, fmt.Sprintf("/root/wg-stats/%s.conf", config.InterfaceName))
			_, err = cmd.Output()
			if err != nil {
				panic(err)
			}

			// update database
			config.Peers[publicKey].Suspended = false
			_, err = config.Collection.UpdateOne(context.TODO(), bson.M{"publicKey": config.Peers[publicKey].PublicKey}, bson.M{"$set": bson.M{"suspended": true}})
			if err != nil {
				panic(err)
			}
		}
	}

	// set servers total and current rx and tx
	config.TotalRx = totalRx
	config.TotalTx = totalTx
	config.CurrentRx = currentRx
	config.CurrentTx = currentTx
}

func findPeerNameByIp(ip string) string {
	for _, p := range config.Peers {
		for _, aip := range strings.Split(p.AllowedIps, ",") {
			if strings.Split(aip, "/")[0] == ip {
				return p.Name
			}
		}
	}
	return ""
}

func findPeerPublicKeyByName(name string) string {
	for pk, p := range config.Peers {
		if strings.Contains(p.Name, name) {
			return pk
		}
	}
	return ""
}

func init() {
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1] + configPath
	}
	bytes, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		panic(err)
	}

	config.Peers = make(map[string]*Peer)

	client, err := mongo.Connect(
		context.TODO(),
		options.Client().ApplyURI(config.MongoURI).SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)))
	if err != nil {
		panic(err)
	}
	config.Collection = client.Database(config.DBName).Collection(config.CollectionName)
	var data []Peer
	cursor, err := config.Collection.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	if err = cursor.All(context.TODO(), &data); err != nil {
		panic(err)
	}
	for _, p := range data {
		config.Peers[p.PublicKey] = &p
		config.Peers[p.PublicKey].PreviousTotalRx = p.TotalRx
		config.Peers[p.PublicKey].PreviousTotalTx = p.TotalTx
	}
}

func main() {
	// get peers info every second
	go func() {
		for range time.NewTicker(time.Second).C {
			getPeers()
		}
	}()

	// update peers totoal usages in datebase every minute
	go func() {
		for range time.NewTicker(time.Minute).C {
			var err error
			var p *Peer
			for _, p = range config.Peers {
				_, err = config.Collection.UpdateOne(
					context.TODO(),
					bson.M{"publicKey": p.PublicKey},
					bson.M{"$set": bson.M{"totalRx": p.TotalRx, "totalTx": p.TotalTx}})
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
	r := gin.Default()
	r.LoadHTMLGlob("/root/wireguard-ui/templates/*")
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Next()
	})
	r.GET("/api/stats", func(c *gin.Context) {
		ra := c.Request.Header.Get("X-Real-IP")
		if ra == "" {
			ra = c.Request.RemoteAddr
		}
		name := findPeerNameByIp(strings.Split(ra, ":")[0])
		tempPeers := make(map[string]*Peer)
		isAdmin := slices.Contains(config.Admins, name)
		if isAdmin {
			tempPeers = config.Peers
		} else {
			for pk, p := range config.Peers {
				if strings.Contains(p.Name, strings.Split(name, "-")[0]+"-") {
					tempPeers[pk] = p
				}
			}
		}
		data := make(map[string]interface{})
		data["peers"] = tempPeers
		data["totalRx"] = config.TotalRx
		data["totalTx"] = config.TotalTx
		data["currentRx"] = config.CurrentRx
		data["currentTx"] = config.CurrentTx
		data["isAdmin"] = isAdmin
		data["name"] = name
		c.JSON(200, data)
	})
	r.POST("/api/peers", func(c *gin.Context) {
		ra := c.Request.Header.Get("X-Real-IP")
		if ra == "" {
			ra = c.Request.RemoteAddr
		}
		name := findPeerNameByIp(strings.Split(ra, ":")[0])
		isAdmin := slices.Contains(config.Admins, name)
		if !isAdmin {
			c.AbortWithStatus(403)
			return
		}
		p := Peer{}
		err := c.BindJSON(&p)
		if err != nil {
			fmt.Println(err)
			c.AbortWithStatus(400)
			return
		}
		if p.Name != "" && p.Name != config.Peers[p.PublicKey].Name {
			cmd := exec.Command("sh", "/root/wg-stats/scripts/replace-string.sh", "/etc/wireguard/wg0.conf", config.Peers[p.PublicKey].Name, p.Name)
			fmt.Println(cmd)
			_, err := cmd.Output()
			if err != nil {
				fmt.Println(err)
				c.AbortWithStatus(400)
				return
			}
			c.AbortWithStatus(200)
		} else {
			config.Peers[p.PublicKey].ExpiresAt = p.ExpiresAt
			_, err = config.Collection.UpdateOne(context.TODO(), bson.M{"publicKey": p.PublicKey}, bson.M{"$set": bson.M{"expiresAt": p.ExpiresAt}})
			if err != nil {
				fmt.Println(err)
				c.AbortWithStatus(400)
				return
			}
			c.AbortWithStatus(200)
		}
	})
	r.GET("/api/peers/:name", func(c *gin.Context) {
		name := c.Param("name")
		if p, ok := config.Peers[findPeerPublicKeyByName(name)]; ok {
			c.JSON(200, p)
		} else {
			c.AbortWithStatus(400)
		}
	})
	r.POST("/api/peers/:name", func(c *gin.Context) {
		name := c.Param("name")
		p, err := createPeer(name)
		if err != nil {
			fmt.Println(err)
			c.JSON(500, map[string]interface{}{"error": err.Error()})
		} else {
			c.JSON(201, p)
		}
	})
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": "Home Page",
		})
	})
	r.GET("/peers", func(c *gin.Context) {
		c.HTML(http.StatusOK, "peers.tmpl", gin.H{
			"title": "peers",
			"peers": config.Peers,
		})
	})

	if err := r.Run(":5051"); err != nil {
		panic(err)
	}
}
