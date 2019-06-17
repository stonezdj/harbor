# Backup Harbor data

1. Login to the Harbor tile VM, find a disk which has enough space, usually located in /var/vcap/store, create a backup directory in it
2. Download script from https://github.com/stonezdj/harbor/blob/backup_restore/tools/harbor-backup.sh and copy it to /var/vcap/store/backup
3. Shutdown the Harbor instance  
```
docker-compose down -v
```
4. Run backup script
```
cd /backup
ls
harbor-backup.sh
```
## Check no container is running, if there is, stop and remove it
docker ps
## Backup all data
./harbor-backup.sh 
## Or only backup database data when harbor storage is NFS, GCP or S3
./harbor-backup.sh --dbonly

5. After backup complete, there is a harbor.tgz file in /var/vcap/store, it the backup data, copy it to backup storage.
```
ls /backup
harbor-backup.sh    harbor.tgz
```
6. Start harbor 
```
docker-compose up -d
```
2. Restore Harbor
Install Harbor and Login to the Harbor host.
Download script from https://github.com/goharbor/harbor/blob/backup_restore/tools/harbor-restore.sh and copy it to /restore
Copy the backup data file harbor.tgz to the directory /restore
Shutdown the Harbor instance
```
docker-compose down -v
```
6. Run restore script
```
cd /restore
ls
harbor-restore.sh   harbor.tgz
```
# Check no container is running, if there is, stop and remove it
```
docker ps
# Restore all data
./harbor-restore.sh 
# Or only restore database data when harbor storage is NFS, GCP or S3
./harbor-restore.sh --dbonly
```
7. Start harbor
```
docker-compose up -d
```
8. Remove all unused data in /restore
```
rm -rf /restore
```