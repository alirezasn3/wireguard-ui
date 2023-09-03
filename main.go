package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var config Config

type Config struct {
	MongoURI             string `json:"mongoURI"`
	DBName               string `json:"dbName"`
	CollectionName       string `json:"collectionName"`
	RefreshInterval      int    `json:"refreshInterval"`
	InterfaceName        string `json:"interfaceName"`
	Collection           *mongo.Collection
	Peers                map[string]*Peer
	TotalRx              uint64
	TotalTx              uint64
	CurrentRx            uint64
	CurrentTx            uint64
	ServerEndpoint       string `json:"serverEndpoint"`
	ServerPublicKey      string `json:"serverPublicKey"`
	ServerNetworkAddress string `json:"serverNetworkAddress"`
	Path                 string `json:"path"`
	DNSServers           string `json:"dnsServers"`
}

type Peer struct {
	ID              primitive.ObjectID `bson:"_id" json:"id"`
	Name            string             `bson:"name" json:"name"`
	PrivateKey      string             `bson:"privatekey" json:"privatekey"`
	PublicKey       string             `bson:"publicKey" json:"publicKey"`
	PresharedKey    string             `bson:"presharedKey" json:"presharedKey"`
	Address         string             `bson:"address" json:"address"`
	ExpiresAt       uint64             `bson:"expiresAt" json:"expiresAt"`
	LatestHandshake uint64             `bson:"-" json:"latestHandshake"`
	TotalRx         uint64             `bson:"-" json:"-"`
	TotalTx         uint64             `bson:"-" json:"-"`
	CurrentRx       uint64             `bson:"-" json:"currentRx"`
	CurrentTx       uint64             `bson:"-" json:"currentTx"`
	Suspended       bool               `bson:"suspended" json:"suspended"`
	AllowedUsage    uint64             `bson:"allowedUsage" json:"allowedUsage"`
	TotalUsage      uint64             `bson:"totalUsage" json:"totalUsage"`
	IsAdmin         bool               `bson:"isAdmin" json:"isAdmin"`
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

func createPeer(name string, isAdmin bool) (*Peer, error) {
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
		Address:      a.ToString() + "/32",
		ExpiresAt:    uint64(time.Now().Unix() + 60*60*24*30),
		AllowedUsage: 50 * 1024000000,
		IsAdmin:      isAdmin,
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
	err = os.WriteFile(config.Path+"/wg0.conf", configBytes, 0644)
	if err != nil {
		panic(err)
	}

	// save chagnes to main config file
	cmd = exec.Command("wg", "syncconf", config.InterfaceName, fmt.Sprintf("%s/%s.conf", config.Path, config.InterfaceName))
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

func deletePeer(name string) error {
	peer := findPeerByName(name)
	if peer == nil {
		return errors.New("peer not found")
	}
	configBytes, err := os.ReadFile(fmt.Sprintf("/etc/wireguard/%s.conf", config.InterfaceName))
	if err != nil {
		return err
	}

	newConfig := strings.Replace(
		string(configBytes),
		fmt.Sprintf("\n[Peer]\nPublicKey = %s\nPresharedKey = %s\nAllowedIPs = %s\n", peer.PublicKey, peer.PresharedKey, peer.Address),
		"",
		1,
	)

	err = os.WriteFile(fmt.Sprintf("/etc/wireguard/%s.conf", config.InterfaceName), []byte(newConfig), 0644)
	if err != nil {
		return err
	}

	_, err = config.Collection.DeleteOne(
		context.TODO(),
		bson.M{"name": name},
	)

	if err == nil {
		delete(config.Peers, peer.PublicKey)
	}

	return err
}

func updatePeers() {
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

	var operations []mongo.WriteModel
	var publicKey string
	var newTotalTx uint64
	var newTotalRx uint64
	for _, p := range peerLines {
		info := strings.Split(p, "\t")

		// find public key
		publicKey = info[0]

		if config.Peers[publicKey] == nil {
			continue
		}

		newTotalTx, _ = strconv.ParseUint(string(info[5]), 10, 64)
		newTotalRx, _ = strconv.ParseUint(string(info[6]), 10, 64)

		// update current rx and tx
		config.Peers[publicKey].CurrentRx = newTotalRx - config.Peers[publicKey].TotalRx
		config.Peers[publicKey].CurrentTx = newTotalTx - config.Peers[publicKey].TotalTx

		// update total rx and tx
		config.Peers[publicKey].TotalRx = newTotalRx
		config.Peers[publicKey].TotalTx = newTotalTx

		// update peer's total usage
		config.Peers[publicKey].TotalUsage += config.Peers[publicKey].CurrentRx
		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(bson.M{"publicKey": publicKey})
		operation.SetUpdate(bson.M{"$set": bson.M{"totalUsage": config.Peers[publicKey].TotalUsage}})
		operations = append(operations, operation)

		// update latest handshake
		config.Peers[publicKey].LatestHandshake, _ = strconv.ParseUint(string(info[4]), 10, 64)

		// update servers total and current rx and tx
		totalRx += config.Peers[publicKey].TotalRx
		totalTx += config.Peers[publicKey].TotalTx
		currentRx += config.Peers[publicKey].CurrentRx
		currentTx += config.Peers[publicKey].CurrentTx

		// suspend expired peers
		if (config.Peers[publicKey].ExpiresAt < uint64(time.Now().Unix()) ||
			config.Peers[publicKey].TotalUsage > config.Peers[publicKey].AllowedUsage) && !config.Peers[publicKey].Suspended {
			fmt.Println("suspending " + config.Peers[publicKey].Name)
			// create invalid preshared key
			invalid := config.Peers[publicKey].ID.Hex() + "AAAAAAAAAAAAAAAAAAA="

			// replace peer's preshared key with the invalid one
			cmd := exec.Command("sh", config.Path+"/scripts/replace-string.sh", fmt.Sprintf("/etc/wireguard/%s.conf", config.InterfaceName), config.Peers[publicKey].PresharedKey, invalid)
			_, err := cmd.Output()
			if err != nil {
				fmt.Println(err)
				continue
			}

			// get striped config
			cmd = exec.Command("wg-quick", "strip", "wg0")
			configBytes, err := cmd.Output()
			if err != nil {
				fmt.Println(err)
			}

			// write striped config to a file
			err = os.WriteFile(config.Path+"/wg0.conf", configBytes, 0644)
			if err != nil {
				fmt.Println(err)
			}

			// save chagnes to main config file
			cmd = exec.Command("wg", "syncconf", config.InterfaceName, fmt.Sprintf("%s/%s.conf", config.Path, config.InterfaceName))
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
		if config.Peers[publicKey].Suspended && (config.Peers[publicKey].ExpiresAt > uint64(time.Now().Unix()) &&
			config.Peers[publicKey].TotalUsage < config.Peers[publicKey].AllowedUsage) {
			fmt.Println("reviving " + config.Peers[publicKey].Name)

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
			cmd := exec.Command("sh", config.Path+"/scripts/replace-string.sh", fmt.Sprintf("/etc/wireguard/%s.conf", config.InterfaceName), invalid, p.PresharedKey)
			_, err := cmd.Output()
			if err != nil {
				panic(err)
			}

			// get striped config
			cmd = exec.Command("wg-quick", "strip", "wg0")
			configBytes, err := cmd.Output()
			if err != nil {
				fmt.Println(err)
			}

			// write striped config to a file
			err = os.WriteFile(config.Path+"/wg0.conf", configBytes, 0644)
			if err != nil {
				fmt.Println(err)
			}

			// save chagnes to main config file
			cmd = exec.Command("wg", "syncconf", config.InterfaceName, fmt.Sprintf("%s/%s.conf", config.Path, config.InterfaceName))
			_, err = cmd.Output()
			if err != nil {
				panic(err)
			}

			// update database
			config.Peers[publicKey].Suspended = false
			_, err = config.Collection.UpdateOne(context.TODO(), bson.M{"publicKey": config.Peers[publicKey].PublicKey}, bson.M{"$set": bson.M{"suspended": false}})
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

	_, err = config.Collection.BulkWrite(context.TODO(), operations, &options.BulkWriteOptions{})
	if err != nil {
		fmt.Println(err)
	}
}

func findPeerByIp(ip string) *Peer {
	for _, p := range config.Peers {
		for _, cidr := range strings.Split(p.Address, ",") {
			if strings.Split(cidr, "/")[0] == ip {
				return p
			}
		}
	}
	return nil
}

func findPeerByName(name string) *Peer {
	for _, p := range config.Peers {
		if strings.Contains(p.Name, name) {
			return p
		}
	}
	return nil
}

func generateConfig(peer *Peer) string {
	return fmt.Sprintf("[Interface]\nPrivateKey = %s\nAddress = %s\nDNS = %s\n[Peer]\nPublicKey = %s\nPresharedKey = %s\nAllowedIPs = 0.0.0.0/0\nEndpoint = %s\n", peer.PrivateKey, peer.Address, config.DNSServers, config.ServerPublicKey, peer.PresharedKey, config.ServerEndpoint)
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
	if len(data) == 0 {
		err := os.MkdirAll("/root/configs", 0700)
		if err != nil {
			panic(err)
		}
		p, err := createPeer("Admin-0", true)
		if err != nil {
			panic(err)
		}
		config := generateConfig(p)
		err = os.WriteFile("/root/configs/Admin-0.conf", []byte(config), 0644)
		if err != nil {
			panic(err)
		}
		fmt.Println("Created new peer in /root/configs/Admin-0.conf\nUse it to connect Wireguard UI admin panel.")
	}

	for i, p := range data {
		config.Peers[p.PublicKey] = &data[i]
	}

	// get peers info from wg
	cmd := exec.Command("wg", "show", config.InterfaceName, "dump")
	bytes, err = cmd.Output()
	if err != nil {
		panic(err)
	}

	// each line contains a peer's info, excluding the first line whichis the interface info
	peerLines := strings.Split(strings.TrimSpace(string(bytes)), "\n")[1:]

	var publicKey string
	var newTotalTx uint64
	var newTotalRx uint64
	for _, p := range peerLines {
		info := strings.Split(p, "\t")

		// find public key
		publicKey = info[0]

		if config.Peers[publicKey] == nil {
			continue
		}

		// update total rx and tx
		newTotalTx, _ = strconv.ParseUint(string(info[5]), 10, 64)
		newTotalRx, _ = strconv.ParseUint(string(info[6]), 10, 64)
		config.Peers[publicKey].TotalRx = newTotalRx
		config.Peers[publicKey].TotalTx = newTotalTx
	}
}

func main() {
	// get peers info every second
	go func() {
		for range time.NewTicker(time.Second).C {
			updatePeers()
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
	r := gin.Default()
	r.Use(static.Serve("/", static.LocalFile(config.Path+"/public/build", false)))
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Next()
	})
	r.GET("/api/stats", func(c *gin.Context) {
		ra := c.Request.RemoteAddr
		peer := findPeerByIp(strings.Split(ra, ":")[0])
		if peer == nil {
			c.AbortWithStatus(403)
			return
		}

		tempPeers := make(map[string]*Peer)

		if peer.IsAdmin {
			tempPeers = config.Peers
		} else {
			for pk, p := range config.Peers {
				if strings.HasPrefix(p.Name, strings.Split(peer.Name, "-")[0]+"-") {
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
		data["isAdmin"] = peer.IsAdmin
		data["name"] = peer.Name
		c.JSON(200, data)
	})
	r.PATCH("/api/peers/:name", func(c *gin.Context) {
		ra := c.Request.RemoteAddr
		client := findPeerByIp(strings.Split(ra, ":")[0])
		if client == nil || !client.IsAdmin {
			c.AbortWithStatus(403)
			return
		}
		peer := findPeerByName(c.Param("name"))
		if peer == nil {
			c.AbortWithStatus(400)
			return
		}
		newPeer := &Peer{}
		err := c.BindJSON(&newPeer)
		if err != nil {
			fmt.Println(err)
			c.AbortWithStatus(400)
			return
		}
		if newPeer.ExpiresAt != 0 {
			peer.ExpiresAt = newPeer.ExpiresAt
		}
		if newPeer.Name != "" {
			peer.Name = newPeer.Name
		}
		if newPeer.AllowedUsage != 0 {
			peer.AllowedUsage = newPeer.AllowedUsage
		}
		_, err = config.Collection.UpdateOne(
			context.TODO(),
			bson.M{"publicKey": peer.PublicKey},
			bson.M{"$set": bson.M{"expiresAt": peer.ExpiresAt, "name": peer.Name, "allowedUsage": peer.AllowedUsage}})
		if err != nil {
			fmt.Println(err)
			c.AbortWithStatus(400)
			return
		}
		c.AbortWithStatus(200)
	})
	r.GET("/api/peers/:name", func(c *gin.Context) {
		name := c.Param("name")
		if p := findPeerByName(name); p != nil {
			c.JSON(200, p)
		} else {
			c.AbortWithStatus(400)
		}
	})
	r.POST("/api/peers/:name", func(c *gin.Context) {
		ra := c.Request.RemoteAddr
		client := findPeerByIp(strings.Split(ra, ":")[0])
		if client == nil || !client.IsAdmin {
			c.AbortWithStatus(403)
			return
		}
		name := c.Param("name")
		p := &Peer{}
		err := c.BindJSON(&p)
		if err != nil {
			fmt.Println(err)
			c.AbortWithStatus(400)
			return
		}
		p, err = createPeer(name, p.IsAdmin)
		if err != nil {
			fmt.Println(err)
			c.JSON(400, map[string]interface{}{"error": err.Error()})
		} else {
			c.JSON(201, p)
		}
	})
	r.DELETE("/api/peers/:name", func(c *gin.Context) {
		ra := c.Request.RemoteAddr
		client := findPeerByIp(strings.Split(ra, ":")[0])
		if client == nil || !client.IsAdmin {
			c.AbortWithStatus(403)
			return
		}
		err := deletePeer(c.Param("name"))
		if err != nil {
			if err.Error() == "peer not found" {
				c.AbortWithStatus(400)
			} else {
				fmt.Println(err)
				c.AbortWithStatus(500)
			}
			return
		}
		c.AbortWithStatus(200)
	})
	r.GET("/api/reset-usage/:name", func(c *gin.Context) {
		ra := c.Request.RemoteAddr
		client := findPeerByIp(strings.Split(ra, ":")[0])
		if client == nil || !client.IsAdmin {
			c.AbortWithStatus(403)
			return
		}
		peer := findPeerByName(c.Param("name"))
		if peer == nil {
			c.AbortWithStatus(400)
			return
		}
		peer.TotalUsage = 0
		_, err := config.Collection.UpdateOne(
			context.TODO(),
			bson.M{"publicKey": peer.PublicKey},
			bson.M{"$set": bson.M{"totalUsage": 0}})
		if err != nil {
			fmt.Println(err)
			c.AbortWithStatus(400)
			return
		}
		c.AbortWithStatus(200)
	})
	r.GET("/api/configs/:name", func(c *gin.Context) {
		name := c.Param("name")
		if p := findPeerByName(name); p != nil {
			c.Data(200, "text/plain", []byte(generateConfig(p)))
		} else {
			c.AbortWithStatus(400)
		}
	})
	if err := r.Run(":80"); err != nil {
		panic(err)
	}
}
