# Harbor Backup and Restore Guide

When the container registry storage is local file system, the simplest way to backup a standalone Harbor data is to backup the /data directory and recover it to /data directory.
For some special cases, if you want more efficient way to backup Harbor data, you can refer the guide following. 

## Backup Harbor data

1. Login to the Harbor host, find a disk which has enough space, create a backup directory in it, for exmaple, /backup
2. Download script from https://github.com/stonezdj/harbor/blob/backup_restore/tools/harbor-backup.sh and copy it to /backup
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
```
docker ps
```
## Backup all data
```
./harbor-backup.sh 
```
## Or only backup database data when harbor storage is NFS, GCP or S3
```
./harbor-backup.sh --dbonly
```
5. After backup complete, there is a harbor.tgz file in /backup, it the backup data.
```
ls /backup
harbor-backup.sh    harbor.tgz
```
6. Start harbor 
```
docker-compose up -d
```
2. Restore Harbor
Install Harbor with the same version and login to the Harbor host.
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