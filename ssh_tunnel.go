package main

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SSHConnConfig struct {
	Host       string
	Port       string
	User       string
	Knownhosts string
	Keyfile    string
}

func NewSSHClient(config *SSHConnConfig) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: config.User,
	}

	if auth := SSHAgent(); auth != nil {
		sshConfig.Auth = append(sshConfig.Auth, auth)
	}

	if hostKeyCallback, err := knownhosts.New(config.Knownhosts); err == nil {
		sshConfig.HostKeyCallback = hostKeyCallback
	}
	if config.Keyfile != "" {
		if auth := PrivateKey(config.Keyfile); auth != nil {
			sshConfig.Auth = append(sshConfig.Auth, auth)
		}
	}

	return ssh.Dial("tcp", net.JoinHostPort(config.Host, config.Port), sshConfig)
}

func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func PrivateKey(path string) ssh.AuthMethod {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(signer)
}
