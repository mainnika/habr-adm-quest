package lib

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"sync/atomic"
	"time"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type Server struct {
	LetterPath        string
	LocalPostboxPath  string
	RemotePostboxPath string

	Alg  string
	Pub  *ecdsa.PublicKey
	Priv *ecdsa.PrivateKey

	Docker *client.Client

	WinnersKey string
	Redis      *redis.Client

	ClientsLimit uint32
	connected    uint32
}

func (s *Server) Serve(lis net.Listener) (err error) {

	for {
		var conn net.Conn

		conn, err = lis.Accept()
		if err != nil {
			return
		}

		go s.HandleClient(conn)
	}
}

func (s *Server) HandleClient(conn net.Conn) {

	var err error

	defer log.Debugf("done client %v", conn)
	defer atomic.AddUint32(&s.connected, ^uint32(0))

	defer func() {
		if err != nil {
			log.Debug(err)
		}

		_ = conn.Close()
	}()

	log.Debugf("new client %v", conn)

	connected := atomic.AddUint32(&s.connected, 1)
	if connected > s.ClientsLimit {
		return
	}

	now := time.Now().UnixNano()
	claims := &jwtgo.StandardClaims{}

	go func() {
		time.Sleep(time.Minute)
		err = conn.Close()
	}()

	key, err := s.readKey(conn)
	if err != nil {
		return
	}

	_, err = jwtgo.ParseWithClaims(key, claims, s.getKey)
	if err != nil {
		return
	}

	if !s.isWinner(claims.Id) {
		return
	}

	log.Debugf("connected with id %s", claims.Id)

	conn.Write([]byte("loading\n"))

	jailId, err := s.createJail(now, claims)
	if err != nil {
		return
	}

	defer s.removeJailNoErr(jailId)

	log.Debugf("jail created %s → %s", claims.Id, jailId)

	err = s.startJail(jailId)
	if err != nil {
		return
	}

	jailConn, err := s.attachJail(jailId)
	if err != nil {
		return
	}

	greetings := fmt.Sprintf("welcome %s! I have a letter for you, just read it!", claims.Subject)
	conn.Write([]byte(fmt.Sprintf("echo %s\n", greetings)))
	cmd := fmt.Sprintf(`echo %s 2>&1 | tee msg /dev/console`, greetings)

	err = s.execJail(jailId, []string{"sh", "-c", cmd})
	if err != nil {
		return
	}

	err = s.startProxyJail(jailConn, conn)
	if err != nil {
		return
	}
}

func (s *Server) getPrivKey(t *jwtgo.Token) (interface{}, error) {
	if _, ok := t.Method.(*jwtgo.SigningMethodECDSA); !ok {
		return nil, errors.New("unexpected signing method")
	}
	return s.Priv, nil
}

func (s *Server) getKey(t *jwtgo.Token) (interface{}, error) {
	if _, ok := t.Method.(*jwtgo.SigningMethodECDSA); !ok {
		return nil, errors.New("unexpected signing method")
	}
	return s.Pub, nil
}

func (s *Server) isWinner(id string) (winner bool) {
	winner, _ = s.Redis.SIsMember(s.WinnersKey, id).Result()
	return
}

func (s *Server) readKey(conn net.Conn) (key string, err error) {

	buf := make([]byte, 4096)
	scanner := bufio.NewScanner(conn)

	scanner.Buffer(buf, 0)

	for {
		err = scanner.Err()
		if err != nil {
			return
		}

		if !scanner.Scan() {
			continue
		}

		key = scanner.Text()
		break
	}

	return
}

func (s *Server) createJail(now int64, claims *jwtgo.StandardClaims) (id string, err error) {

	letterPath, _, err := s.createLetter()
	if err != nil {
		return
	}

	remoteLetterPath := path.Join(s.RemotePostboxPath, letterPath)
	name := fmt.Sprintf("%d-%s", now, claims.Id)
	cfg := &container.Config{
		NetworkDisabled: true,
		Labels: map[string]string{
			"id": claims.Id,
		},
		Image:     "alpine",
		User:      "nobody",
		OpenStdin: true,
		Tty:       true,
		Cmd:       []string{"/bin/sh"},
	}
	letter := mount.Mount{
		Type:   mount.TypeBind,
		Source: remoteLetterPath,
		Target: fmt.Sprintf("/%s", letterPath),
	}
	hcfg := &container.HostConfig{
		NetworkMode: "none",
		Mounts:      []mount.Mount{letter},
	}
	ncfg := &network.NetworkingConfig{}
	created, err := s.Docker.ContainerCreate(context.Background(), cfg, hcfg, ncfg, name)
	if err != nil {
		return
	}

	id = created.ID

	return
}

func (s *Server) removeJailNoErr(id string) {
	_ = s.Docker.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true})
}

func (s *Server) startJail(id string) (err error) {
	return s.Docker.ContainerStart(context.Background(), id, types.ContainerStartOptions{})
}

func (s *Server) execJail(id string, cmd []string) (err error) {

	ctx := context.Background()
	exec, err := s.Docker.ContainerExecCreate(ctx, id, types.ExecConfig{User: "root", Cmd: cmd, Detach: true})
	if err != nil {
		return
	}

	err = s.Docker.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{Detach: true})

	return
}

func (s *Server) attachJail(id string) (conn net.Conn, err error) {

	containerio, err := s.Docker.ContainerAttach(context.Background(), id, types.ContainerAttachOptions{Stdin: true, Stdout: true, Stderr: true, Stream: true})
	if err != nil {
		return
	}

	conn = containerio.Conn

	return
}

func (s *Server) startProxyJail(jailconn net.Conn, userconn net.Conn) (err error) {

	closed := new(uint32)
	stdin := createProxyChan(userconn, closed)
	stdout := createProxyChan(jailconn, closed)

	defer jailconn.Close()

Proxying:
	for {
		select {
		case data, ok := <-stdin:
			if !ok {
				break Proxying
			}
			_, err = jailconn.Write(data)
		case data, ok := <-stdout:
			if !ok {
				break Proxying
			}
			_, err = userconn.Write(data)
		}

		if err != nil {
			break
		}
	}

	return
}

func (s *Server) createLetter() (letterPath string, localFullPath string, err error) {

	source, err := os.Open(s.LetterPath)
	if err != nil {
		return
	}
	defer source.Close()

	letterPath = fmt.Sprintf("%s.letter", uuid.New().String())
	localFullPath = path.Join(s.LocalPostboxPath, letterPath)

	letter, err := os.OpenFile(localFullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0000)
	if err != nil {
		return
	}
	defer letter.Close()

	_, err = io.Copy(letter, source)
	if err != nil {
		return
	}

	unix.Sync()

	return
}

func createProxyChan(conn net.Conn, closed *uint32) (out chan []byte) {

	out = make(chan []byte)
	b := make([]byte, 1024)

	go func() {

		defer close(out)

		for {
			deadline := time.Now().Add(time.Minute)
			err := conn.SetReadDeadline(deadline)
			if err != nil {
				log.Debugf("conn deadline failed: %v", err)
				return
			}

			if !atomic.CompareAndSwapUint32(closed, 0, 0) {
				return
			}

			n, err := conn.Read(b)
			if n > 0 {
				res := make([]byte, n)
				copy(res, b[:n])
				out <- res
			}
			if err != nil {
				log.Debugf("conn read failed: %v", err)
				return
			}
		}
	}()

	return
}
