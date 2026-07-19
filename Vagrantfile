# FreeZenith local test server.
#
# Spins up a clean Ubuntu VM with Docker — a real "server" you can install
# FreeZenith into and reach from your host, without paying for a VPS.
#
#   vagrant up
#   vagrant ssh
#   # inside the VM:
#   curl -fsSL https://raw.githubusercontent.com/taikuri-infra/Zenith/main/infra/scripts/install.sh | bash
#
# Then open http://localhost:3000 on your host (forwarded from the VM).
#
# For a domain + HTTPS test without a public domain, use a wildcard nip.io host
# that resolves to the VM's private IP, e.g. ZENITH_DOMAIN=zenith.192.168.56.20.nip.io

Vagrant.configure("2") do |config|
  config.vm.box = "bento/ubuntu-22.04"
  config.vm.hostname = "freezenith-test"

  # Private IP so you can reach the VM (and app subdomains via nip.io) directly.
  config.vm.network "private_network", ip: "192.168.56.20"

  # Forward the main ports to the host so http://localhost:<port> works.
  config.vm.network "forwarded_port", guest: 3000, host: 3000, auto_correct: true # dashboard
  config.vm.network "forwarded_port", guest: 8080, host: 8080, auto_correct: true # api
  config.vm.network "forwarded_port", guest: 80,   host: 8888, auto_correct: true # http (Caddy)
  config.vm.network "forwarded_port", guest: 443,  host: 8443, auto_correct: true # https (Caddy)

  config.vm.provider "virtualbox" do |vb|
    vb.name   = "freezenith-test"
    vb.memory = 8192   # of your 16 GB — comfortable headroom for the stack + a test app
    vb.cpus   = 4
  end

  # Install Docker so the VM is a ready-to-use server.
  config.vm.provision "shell", inline: <<-SHELL
    set -e
    if ! command -v docker >/dev/null 2>&1; then
      echo "==> Installing Docker..."
      curl -fsSL https://get.docker.com | sh
      usermod -aG docker vagrant
    fi
    echo ""
    echo "============================================================"
    echo "  FreeZenith test VM ready."
    echo "  vagrant ssh, then run the installer:"
    echo "    curl -fsSL https://raw.githubusercontent.com/taikuri-infra/Zenith/main/infra/scripts/install.sh | bash"
    echo "  Dashboard will be at http://localhost:3000 on your host."
    echo "============================================================"
  SHELL
end
