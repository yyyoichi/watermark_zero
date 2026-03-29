sudo apt-get update && sudo apt-get install -y libopencv-dev

if [ -z "$(ls -A /tmp/cargo/target)" ]; then
  sudo chown -R vscode:vscode /tmp/cargo/target
fi