# ⚡ Infrakit

[![Go Report Card](https://goreportcard.com/badge/github.com/rahulwagh/infrakit)](https://goreportcard.com/report/github.com/rahulwagh/infrakit)
[![Build Status](https://github.com/rahulwagh/infrakit/actions/workflows/go.yml/badge.svg)](https://github.com/rahulwagh/infrakit/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**A blazing fast CLI and Web UI for interactive fuzzy-searching of your AWS, GCP, and Azure cloud resources. The missing `grep` for your cloud infrastructure.**

Tired of clicking through endless pages in the slow AWS console just to find an EC2 instance's IP or an IAM role's permissions? **Infrakit** keeps you in your terminal and gives you the power to find any resource in milliseconds.

---

## 🚀 Demo

See `infrakit` in action! The interactive fuzzy finder makes searching for resources across your entire cloud account feel instantaneous.

![Infrakit CLI Demo](https://user-images.githubusercontent.com/rahulwagh/your-image-id/infrakit-demo.gif)

*(To create a GIF like this, you can use tools like [Terminalizer](https://github.com/faressoft/terminalizer) or [asciinema](https://asciinema.org/). After uploading it to a GitHub issue or comment, you can copy the image link.)*

---

## 🤔 Why Infrakit?

Cloud providers offer powerful services, but their web consoles are often slow, cumbersome, and require dozens of clicks for simple tasks. As a developer or DevOps engineer, context switching between your code and a browser is a major drag on productivity.

**Infrakit** solves this by treating your cloud infrastructure as a local, searchable database. It fetches metadata from your cloud accounts, caches it locally, and provides a powerful, `fzf`-style interface to find what you need instantly.

| Feature | Description |
| :-- | :-- |
| 🚀 **Blazing Fast** | By using a local cache, searches are performed in milliseconds, not seconds. |
| 🔍 **Fuzzy Search** | No need to remember exact IDs or names. Just type a few characters and `infrakit` will find matching resources instantly. |
| 💻 **CLI & Web UI** | Use the powerful interactive finder in your terminal or start a local web server to search from your browser. |
| 🌐 **Multi-Cloud Ready** | Built on a standardized resource model, `infrakit` is designed to support AWS, GCP, and Azure from day one. (GCP/Azure coming soon!) |
| ✅ **Simple & Focused** | Does one thing and does it well: finds resources. No complex query languages to learn. |
| 🐹 **Written in Go** | Compiles to a single, dependency-free binary that is fast and easy to install on any system. |

---

## 🛠️ Installation

### Install `go` if you don't have already install 

#### Ubuntu
1. Update the package manager
```bash
sudo apt-get update
```

2. Install Go
```bash
sudo apt install golang-go
```


### With `go install`

If you have Go installed, you can install `infrakit` with a single command:

```bash
go install github.com/rahulwagh/infrakit@latest
```

#### Add the Go Path to your Shell's Configuration File
Run the following command to add the Go binary path to your .bashrc file. This file runs every time you start a new terminal session.
```bash 
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc 
```

#### Apply the Changes to Your Current Session
```bash 
source ~/.bashrc
```

## 🏁 Quick Start
### Step 1: Configure Your Cloud Credentials

infrakit uses the official AWS SDK, which automatically detects your credentials.
The easiest way to set this up is by installing the AWS CLI and running:

```bash
aws configure
```

### Step 2: Sync Your Resources

Before you can search, you need to build the local cache.
The sync command fetches metadata for all supported resources and saves it to ~/.infrakit/cache.json.

```bash
infrakit sync
```

Note: Run this command periodically to keep your cache up-to-date with your live infrastructure.

### Step 3: Search!

You have two ways to search your resources:

A) Interactive CLI Search

Run the search command to launch the full-screen interactive fuzzy finder:
```bash
infrakit search
```
B) Local Web UI Search

Run the serve command to start a local web server:
```bash
infrakit serve
```
Then, open your browser and go to http://localhost:8080

## 🗺️ Supported Services

| Provider | Service          |    Status   |
| :------- | :--------------- | :---------: |
| AWS      | EC2 Instances    | ✅ Supported |
| AWS      | IAM Roles        | ✅ Supported |
| AWS      | S3 Buckets       |  ⏳ Planned  |
| AWS      | RDS Databases    |  ⏳ Planned  |
| GCP      | Compute Engine   |  ⏳ Planned  |
| Azure    | Virtual Machines |  ⏳ Planned  |


## 🚧 Project Roadmap

This project is actively being developed. Here's what's planned for the future:
More AWS Services: S3, RDS, Lambda, VPCs, and Security Groups
GCP & Azure Support: Add fetchers for the other major cloud providers
Profile Support: Ability to switch between different AWS profiles
Advanced Output: Option to output search results as JSON or YAML for scripting
Automated Sync: A background daemon to keep the cache fresh automatically

## 🙌 Contributing

Contributions are welcome! If you have an idea for a new feature or want to add support for a new service, please open an issue to discuss it first.

1. Fork the repository

2. Create your feature branch:
```bash
git checkout -b feature/AmazingFeature
```
3. Commit your changes:
```bash
git commit -m 'Add some AmazingFeature'
```
4. Push to the branch:
```bash
git push origin feature/AmazingFeature
```
5. Open a Pull Request

## 📜 License

This project is licensed under the MIT License — see the LICENSE

✅ This version:
- Uses **pure Markdown**, no mixed syntax.
- Keeps heading levels consistent.
- Works 100% when previewed on **GitHub** or in VS Code’s Markdown viewer.

Would you like me to include a small **"Build from Source"** section (with `git clone`, `go build`, etc.) after *Installation* for developer contributors?

