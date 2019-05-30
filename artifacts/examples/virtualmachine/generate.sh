for i in {1..50}; do
cat << EOF > vm-demo$i.yaml
apiVersion: hobbyfarm.io/v1
kind: VirtualMachine
metadata:
  name: vm-demo$i
spec:
  id: vm-demo$i
  user:
  vm_template_id: ubuntu1804-docker1
  vm_claim_id:
  keypair_name: lolmykeypair
status:
  status: running
  allocated: false
  public_ip: "54.174.64.35"
  private_ip: "54.174.64.35"
  environment_id: e-asdfsafa
  hostname: ip-54-174-64-35
EOF
done
