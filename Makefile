ANSIBLE_DIR := infra/ansible
ANSIBLE := cd $(ANSIBLE_DIR) &&

# === Deploy ===

deploy-primary:
	$(ANSIBLE) ansible-playbook -i hosts/lxcs deploy.yml -e "host=primary_lxc target=primary deploy_action=up"

deploy-network:
	$(ANSIBLE) ansible-playbook -i hosts/lxcs deploy.yml -e "host=network_lxc target=network deploy_action=up"

deploy-dublin:
	$(ANSIBLE) ansible-playbook -i hosts/dublin deploy.yml -e "host=dublin_pi target=dublin deploy_action=up remote_path=/home/admin/home-server"

update-primary:
	$(ANSIBLE) ansible-playbook -i hosts/lxcs deploy.yml -e "host=primary_lxc target=primary deploy_action=update"

update-network:
	$(ANSIBLE) ansible-playbook -i hosts/lxcs deploy.yml -e "host=network_lxc target=network deploy_action=update"

update-dublin:
	$(ANSIBLE) ansible-playbook -i hosts/dublin deploy.yml -e "host=dublin_pi target=dublin deploy_action=update remote_path=/home/admin/home-server"

down-primary:
	$(ANSIBLE) ansible-playbook -i hosts/lxcs deploy.yml -e "host=primary_lxc target=primary deploy_action=down"

# === Provision ===

provision-primary:
	$(ANSIBLE) ansible-playbook -i hosts/lxcs -l primary_lxc primary.yml

provision-network:
	$(ANSIBLE) ansible-playbook -i vms network.yml

provision-dublin:
	$(ANSIBLE) ansible-playbook -i hosts/dublin dublin.yml

provision-proxmox:
	$(ANSIBLE) ansible-playbook -i hosts/proxmox proxmox.yml

provision-openclaw:
	$(ANSIBLE) ansible-playbook -i hosts/lxcs openclaw.yml

provision-laptop:
	$(ANSIBLE) ansible-playbook laptop.yml

# === Maintenance ===

upgrade-all:
	$(ANSIBLE) ansible -i hosts/proxmox -i hosts/lxcs -i hosts/dublin all -m apt -a "update_cache=yes upgrade=dist" -b

prune-all:
	$(ANSIBLE) ansible -i hosts/lxcs -i hosts/dublin all -m shell -a "docker system prune -af --volumes" -b

status-primary:
	@ssh root@192.168.88.212 'docker ps --format "table {{.Names}}\t{{.Status}}"'

status-network:
	@ssh root@192.168.88.213 'docker ps --format "table {{.Names}}\t{{.Status}}"'

# === Help ===

help:
	@echo "Deploy:     deploy-primary, deploy-network, deploy-dublin"
	@echo "Update:     update-primary, update-network, update-dublin"
	@echo "Down:       down-primary"
	@echo "Provision:  provision-primary, provision-network, provision-dublin, provision-proxmox, provision-openclaw, provision-laptop"
	@echo "Maintain:   upgrade-all, prune-all"
	@echo "Status:     status-primary, status-network"

.PHONY: deploy-primary deploy-network deploy-dublin update-primary update-network update-dublin down-primary provision-primary provision-network provision-dublin provision-proxmox provision-openclaw provision-laptop upgrade-all prune-all status-primary status-network help
