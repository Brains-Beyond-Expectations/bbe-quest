# BBE-Quest: Big Brain Energy Quest

![BBE-Quest Banner](./assets/banner.webp)

## Welcome to BBE-Quest

BBE-Quest (Big Brain Energy Quest) is your ultimate guide to building and
automating a home lab Kubernetes (k8s) cluster. This project offers a fully
guided and automated journey, taking the complexity out of the process and
making it accessible even for those new to the world of home labs.

### Project Objective

Our goal is to provide:

- **Streamlined Home Lab Automation**: Automate the setup and management of your
  Kubernetes cluster with minimal manual intervention.
- **Guided Learning Journey**: Each step is designed to teach and empower you,
  turning complex tasks into achievable milestones.
- **Collaborative Excellence**: Crafted by two passionate nerds who love to
  share knowledge and solve challenges together.

---

### Key Features

- End-to-end scripts for deploying and managing a k8s cluster.
- Documentation and guidance for understanding each phase of the process.
- Tools and tips for monitoring, scaling, and maintaining your setup.

### Getting Started

## Requirements

- [Crane](https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md)
- [balenaEtcher](https://www.balena.io/etcher/)
- [talosctl](https://www.talos.dev/v1.8/learn-more/talosctl/)

## Usage on Intel NUC

> [!NOTE]  
> Since Talos does not support secure boot on x86, you will need to disable
> secureboot in the BIOS settings of the Intel NUC.

1. Run the talos-iso.sh script to create the ISO file.

```bash
bash ./talos-iso.sh
```

2. Use balenaEtcher to flash the ISO file to a USB drive.

3. Boot the Intel NUC from the USB drive.

4. ???

5. Profit!
